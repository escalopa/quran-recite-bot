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
		"start":     b.commandStart,
		"help":      b.commandHelp,
		"language":  b.commandLanguage,
		"myrecords": b.commandMyRecords,
		"newrecord": b.commandNewRecord,
	}

	// Set bot commands for Telegram UI
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "newrecord", Description: "Create a new recording"},
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

	// Send welcome message
	b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "welcome.message"))

	// Show surah selection
	b.sendSurahSelection(ctx, msg.Chat.ID, userID, lang, 0)
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
