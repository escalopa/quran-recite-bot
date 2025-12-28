package telegram

import (
	"context"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CommandHandler func(ctx context.Context, msg *tgbotapi.Message)

// registerCommands registers all bot commands
func (b *Bot) registerCommands() {
	// Register command handlers
	b.commands = map[string]CommandHandler{
		"start":      b.commandStart,
		"help":       b.commandHelp,
		"language":   b.commandLanguage,
		"myrecords":  b.commandMyRecords,
		"newrecord":  b.commandNewRecord,
		"autodetect": b.commandAutoDetect,
	}

	// Set bot commands for Telegram UI
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "newrecord", Description: "Create a new recording (manual)"},
		{Command: "autodetect", Description: "Auto-detect recording (no ayah selection)"},
		{Command: "myrecords", Description: "View my recordings"},
		{Command: "language", Description: "Change language"},
		{Command: "help", Description: "Show help"},
	}

	cmdConfig := tgbotapi.NewSetMyCommands(commands...)
	if _, err := b.api.Request(cmdConfig); err != nil {
		log.Printf("Error setting bot commands: %v", err)
	}
}

func (b *Bot) commandStart(ctx context.Context, msg *tgbotapi.Message) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	lang := b.service.GetUserLanguage(ctx, userID)

	if err := b.service.HandleStart(ctx, userID, lang); err != nil {
		log.Printf("Error handling start: %v", err)
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.generic"))
		return
	}

	// Send welcome message with mode selection
	text := "🌟 <b>Welcome to Quran Reading Bot!</b>\n\n"
	text += "I can help you improve your Quran recitation.\n\n"
	text += "<b>Choose a mode:</b>"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🎤 Auto-Detect (Easy)",
				"mode_autodetect",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"📝 Manual (Choose Ayah)",
				"mode_manual",
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"📚 My Recordings",
				"myrecords_start",
			),
		),
	)

	welcomeMsg := tgbotapi.NewMessage(msg.Chat.ID, text)
	welcomeMsg.ParseMode = "HTML"
	welcomeMsg.ReplyMarkup = keyboard
	b.api.Send(welcomeMsg)
}

func (b *Bot) commandHelp(ctx context.Context, msg *tgbotapi.Message) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	lang := b.service.GetUserLanguage(ctx, userID)
	b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "help.message"))
}

func (b *Bot) commandLanguage(ctx context.Context, msg *tgbotapi.Message) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	lang := b.service.GetUserLanguage(ctx, userID)
	b.sendLanguageSelection(msg.Chat.ID, lang)
}

func (b *Bot) commandNewRecord(ctx context.Context, msg *tgbotapi.Message) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	lang := b.service.GetUserLanguage(ctx, userID)

	if err := b.service.HandleStart(ctx, userID, lang); err != nil {
		log.Printf("Error handling start: %v", err)
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.generic"))
		return
	}

	b.sendSurahSelection(ctx, msg.Chat.ID, userID, lang, 0)
}

func (b *Bot) commandMyRecords(ctx context.Context, msg *tgbotapi.Message) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	lang := b.service.GetUserLanguage(ctx, userID)

	// Fetch recordings
	recordings, err := b.service.ListRecordings(ctx, userID, 10)
	if err != nil {
		log.Printf("Error listing recordings: %v", err)
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.generic"))
		return
	}

	if len(recordings) == 0 {
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "recordings.empty"))
		return
	}

	b.sendRecordingsList(msg.Chat.ID, userID, lang, recordings, 0)
}

func (b *Bot) commandAutoDetect(ctx context.Context, msg *tgbotapi.Message) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	lang := b.service.GetUserLanguage(ctx, userID)

	// Set auto-detect mode
	if err := b.service.StartAutoDetectMode(ctx, userID); err != nil {
		log.Printf("Error starting auto-detect mode: %v", err)
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.generic"))
		return
	}

	// Send instructions
	text := "🎤 <b>Auto-Detect Mode</b>\n\n"
	text += "Just send me a voice message of your Quran recitation and I'll automatically detect which Ayah(s) you're reading!\n\n"
	text += "📝 <b>Tips:</b>\n"
	text += "• Speak clearly in a quiet environment\n"
	text += "• Start from the beginning of an Ayah\n"
	text += "• You can recite multiple consecutive Ayahs\n"
	text += "• Processing takes 10-30 seconds\n\n"
	text += "🎙️ <b>Ready?</b> Send your voice message now!"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"❌ Cancel",
				"cancel_autodetect",
			),
		),
	)

	msgToSend := tgbotapi.NewMessage(msg.Chat.ID, text)
	msgToSend.ParseMode = "HTML"
	msgToSend.ReplyMarkup = keyboard
	b.api.Send(msgToSend)
}
