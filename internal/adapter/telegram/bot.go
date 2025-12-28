package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/escalopa/quran-read-bot/internal/application"
	"github.com/escalopa/quran-read-bot/internal/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api      *tgbotapi.BotAPI
	service  *application.BotService
	i18n     domain.I18nPort
	commands map[string]CommandHandler
	cancel   context.CancelFunc
}

func NewBot(token string, service *application.BotService, i18n domain.I18nPort) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	bot := &Bot{
		api:      api,
		service:  service,
		i18n:     i18n,
		commands: make(map[string]CommandHandler),
	}

	// Register commands
	bot.registerCommands()

	return bot, nil
}

func (b *Bot) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	b.cancel = cancel

	log.Printf("Authorized on account %s", b.api.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			go b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) Stop() error {
	if b.cancel != nil {
		b.cancel()
	}
	b.api.StopReceivingUpdates()
	return nil
}

func (b *Bot) handleUpdate(ctx context.Context, update tgbotapi.Update) {
	userID := b.getUserID(update)
	if userID == "" {
		return
	}

	lang := b.service.GetUserLanguage(ctx, userID)

	// Handle commands
	if update.Message != nil && update.Message.IsCommand() {
		b.handleCommand(ctx, update.Message, lang)
		return
	}

	// Handle voice messages
	if update.Message != nil && update.Message.Voice != nil {
		b.handleVoice(ctx, update.Message, lang)
		return
	}

	// Handle callback queries (button presses)
	if update.CallbackQuery != nil {
		b.handleCallback(ctx, update.CallbackQuery, lang)
		return
	}

	// Handle text messages (ayah number input)
	if update.Message != nil && update.Message.Text != "" {
		b.handleText(ctx, update.Message, lang)
		return
	}
}

func (b *Bot) handleCommand(ctx context.Context, msg *tgbotapi.Message, lang domain.Language) {
	cmd := msg.Command()

	handler, exists := b.commands[cmd]
	if !exists {
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.unknown_command"))
		return
	}

	handler(ctx, msg)
}

func (b *Bot) handleCallback(ctx context.Context, callback *tgbotapi.CallbackQuery, lang domain.Language) {
	userID := strconv.FormatInt(callback.From.ID, 10)
	chatID := callback.Message.Chat.ID

	// Answer callback to remove loading state
	b.api.Send(tgbotapi.NewCallback(callback.ID, ""))

	// Parse callback data
	data := callback.Data

	// Handle language selection
	if len(data) > 5 && data[:5] == "lang:" {
		newLang := domain.Language(data[5:])
		if err := b.service.HandleStart(ctx, userID, newLang); err != nil {
			log.Printf("Error setting language: %v", err)
			return
		}
		b.sendMessage(chatID, b.i18n.Get(newLang, "language.changed"))
		b.sendSurahSelection(ctx, chatID, userID, newLang, 0)
		return
	}

	// Handle surah page navigation
	if len(data) > 6 && data[:6] == "spage:" {
		page, _ := strconv.Atoi(data[6:])
		b.editSurahSelection(ctx, callback.Message, userID, lang, page)
		return
	}

	// Handle surah selection
	if len(data) > 6 && data[:6] == "surah:" {
		surahNum, err := strconv.Atoi(data[6:])
		if err != nil {
			b.answerCallbackAlert(callback.ID, b.i18n.Get(lang, "error.invalid_input"))
			return
		}

		if err := b.service.HandleSurahSelection(ctx, userID, surahNum); err != nil {
			log.Printf("Error selecting surah: %v", err)
			b.answerCallbackAlert(callback.ID, b.i18n.Get(lang, "error.generic"))
			return
		}

		// Get selected surah info
		surahs := b.service.GetAllSurahs()
		surah := surahs[surahNum-1]
		surahName := b.i18n.GetSurahName(lang, surahNum)

		// Clear any previous ayah input
		b.service.ClearAyahInput(ctx, userID)

		// Edit the message to show ayah selection
		msg := b.i18n.Get(lang, "ayah.select", surahName, surah.Ayahs)
		b.editMessageWithKeyboard(callback.Message, msg, b.getAyahKeyboard(lang, ""))
		return
	}

	// Handle digit input
	if len(data) > 6 && data[:6] == "digit:" {
		b.handleDigitInput(ctx, callback.Message, userID, lang, data[6:])
		return
	}

	// Handle clear/backspace
	if data == "clear" {
		b.handleClearDigit(ctx, callback.Message, userID, lang)
		return
	}

	// Handle done (when ayah number is entered)
	if data == "done" {
		b.handleAyahDone(ctx, callback.Message, userID, lang)
		return
	}

	// Handle check recording status
	if len(data) > 6 && data[:6] == "check:" {
		recordingID := data[6:]
		b.handleCheckRecording(ctx, callback.Message, userID, lang, recordingID)
		return
	}

	// Handle new recording button
	if data == "newrecord" {
		chatID := callback.Message.Chat.ID
		if err := b.service.HandleStart(ctx, userID, lang); err != nil {
			log.Printf("Error handling start: %v", err)
			return
		}
		// Delete the previous message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.api.Send(deleteMsg)
		// Show surah selection
		b.sendSurahSelection(ctx, chatID, userID, lang, 0)
		return
	}

	// Handle recording list navigation
	if len(data) > 8 && data[:8] == "recpage:" {
		page, _ := strconv.Atoi(data[8:])
		recordings, err := b.service.ListRecordings(ctx, userID, 50)
		if err != nil {
			log.Printf("Error listing recordings: %v", err)
			return
		}
		b.editRecordingsList(callback.Message, userID, lang, recordings, page)
		return
	}

	// Handle view recording details
	if len(data) > 8 && data[:8] == "viewrec:" {
		recordingID := data[8:]
		b.handleViewRecording(ctx, callback.Message, userID, lang, recordingID)
		return
	}

	// Handle back to recordings list
	if data == "backtorecs" {
		recordings, err := b.service.ListRecordings(ctx, userID, 50)
		if err != nil {
			log.Printf("Error listing recordings: %v", err)
			return
		}
		b.editRecordingsList(callback.Message, userID, lang, recordings, 0)
		return
	}

	// Handle new auto-detect
	if data == "new_autodetect" {
		chatID := callback.Message.Chat.ID
		if err := b.service.StartAutoDetectMode(ctx, userID); err != nil {
			log.Printf("Error starting auto-detect mode: %v", err)
			return
		}

		// Delete the previous message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.api.Send(deleteMsg)

		// Send auto-detect instructions
		text := "🎤 <b>Auto-Detect Mode</b>\n\n"
		text += "Send me a voice message and I'll detect which Ayah(s) you're reading!\n\n"
		text += "🎙️ <b>Ready?</b> Send your voice message now!"

		msgToSend := tgbotapi.NewMessage(chatID, text)
		msgToSend.ParseMode = "HTML"
		b.api.Send(msgToSend)
		return
	}

	// Handle cancel auto-detect
	if data == "cancel_autodetect" {
		if err := b.service.HandleStart(ctx, userID, lang); err != nil {
			log.Printf("Error cancelling auto-detect: %v", err)
			return
		}

		chatID := callback.Message.Chat.ID
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.api.Send(deleteMsg)

		b.sendMessage(chatID, "Auto-detect cancelled. Use /start or /newrecord to begin.")
		return
	}

	// Handle mode selection - auto-detect
	if data == "mode_autodetect" {
		chatID := callback.Message.Chat.ID
		if err := b.service.StartAutoDetectMode(ctx, userID); err != nil {
			log.Printf("Error starting auto-detect mode: %v", err)
			return
		}

		// Delete the previous message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.api.Send(deleteMsg)

		// Send auto-detect instructions
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

		msgToSend := tgbotapi.NewMessage(chatID, text)
		msgToSend.ParseMode = "HTML"
		msgToSend.ReplyMarkup = keyboard
		b.api.Send(msgToSend)
		return
	}

	// Handle mode selection - manual
	if data == "mode_manual" {
		chatID := callback.Message.Chat.ID

		// Delete the previous message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.api.Send(deleteMsg)

		// Show surah selection
		b.sendSurahSelection(ctx, chatID, userID, lang, 0)
		return
	}

	// Handle my recordings from start
	if data == "myrecords_start" {
		chatID := callback.Message.Chat.ID

		recordings, err := b.service.ListRecordings(ctx, userID, 50)
		if err != nil {
			log.Printf("Error listing recordings: %v", err)
			b.answerCallbackAlert(callback.ID, "Error loading recordings")
			return
		}

		if len(recordings) == 0 {
			b.answerCallbackAlert(callback.ID, "You don't have any recordings yet!")
			return
		}

		// Delete the previous message
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, callback.Message.MessageID)
		b.api.Send(deleteMsg)

		b.sendRecordingsList(chatID, userID, lang, recordings, 0)
		return
	}
}

func (b *Bot) handleText(ctx context.Context, msg *tgbotapi.Message, lang domain.Language) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	chatID := msg.Chat.ID

	state, err := b.service.GetCurrentState(ctx, userID)
	if err != nil {
		log.Printf("Error getting state: %v", err)
		b.sendMessage(chatID, b.i18n.Get(lang, "error.generic"))
		return
	}

	// Handle ayah number input
	if state == domain.StateEnterAyah {
		if err := b.service.HandleAyahInput(ctx, userID, msg.Text); err != nil {
			b.sendMessage(chatID, b.i18n.Get(lang, "error.invalid_ayah"))
			return
		}

		// Prompt for recording
		b.sendMessage(chatID, b.i18n.Get(lang, "recording.prompt"))
		return
	}

	// For other states, show help
	b.sendMessage(chatID, b.i18n.Get(lang, "help.message"))
}

func (b *Bot) handleVoice(ctx context.Context, msg *tgbotapi.Message, lang domain.Language) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	chatID := msg.Chat.ID

	state, err := b.service.GetCurrentState(ctx, userID)
	if err != nil {
		b.sendMessage(chatID, b.i18n.Get(lang, "error.unexpected_voice"))
		return
	}

	// Check if in auto-detect mode
	if state == domain.StateWaitAutoDetect {
		b.handleAutoDetectVoice(ctx, msg, lang)
		return
	}

	// Regular manual mode
	if state != domain.StateWaitRecording {
		b.sendMessage(chatID, b.i18n.Get(lang, "error.unexpected_voice"))
		return
	}

	// Send processing message
	b.sendMessage(chatID, b.i18n.Get(lang, "recording.processing"))

	// Process voice message (download and convert to WAV)
	audioReader, err := b.processVoiceMessage(msg.Voice.FileID)
	if err != nil {
		log.Printf("Error processing voice message: %v", err)
		b.sendMessage(chatID, b.i18n.Get(lang, "error.audio_conversion"))
		return
	}

	// Submit recording to API
	recording, err := b.service.HandleRecording(ctx, userID, audioReader)
	if err != nil {
		log.Printf("Error handling recording: %v", err)
		b.sendMessage(chatID, b.i18n.Get(lang, "error.recording_failed"))
		return
	}

	// Send success message with recording ID
	successMsg := b.i18n.Get(lang, "recording.submitted", recording.ID)
	b.sendMessage(chatID, successMsg)

	// Offer to check status or create new recording
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "recording.check_status"),
				fmt.Sprintf("check:%s", recording.ID),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "recording.new"),
				"newrecord",
			),
		),
	)

	replyMsg := tgbotapi.NewMessage(chatID, b.i18n.Get(lang, "recording.what_next"))
	replyMsg.ReplyMarkup = keyboard
	b.api.Send(replyMsg)
}

func (b *Bot) handleAutoDetectVoice(ctx context.Context, msg *tgbotapi.Message, lang domain.Language) {
	userID := strconv.FormatInt(msg.From.ID, 10)
	chatID := msg.Chat.ID

	// Send processing message
	processingText := "🔍 <b>Auto-Detecting...</b>\n\n"
	processingText += "Processing your recording to identify which Ayah(s) you recited.\n"
	processingText += "This may take 10-30 seconds..."

	processingMsg := tgbotapi.NewMessage(chatID, processingText)
	processingMsg.ParseMode = "HTML"
	sent, err := b.api.Send(processingMsg)
	if err != nil {
		log.Printf("Error sending processing message: %v", err)
	}

	// Process voice message (download and convert to WAV)
	audioReader, err := b.processVoiceMessage(msg.Voice.FileID)
	if err != nil {
		log.Printf("Error processing voice message: %v", err)
		b.sendMessage(chatID, "❌ Error processing audio file. Please try again.")
		return
	}

	// Submit recording to API for auto-detection
	recording, err := b.service.HandleAutoDetectRecording(ctx, userID, audioReader)
	if err != nil {
		log.Printf("Error handling auto-detect recording: %v", err)
		b.sendMessage(chatID, "❌ Error submitting recording. Please try again.")
		return
	}

	// Update the processing message with success
	successText := "✅ <b>Recording Submitted!</b>\n\n"
	successText += fmt.Sprintf("🆔 Recording ID: <code>%s</code>\n", recording.ID)
	successText += fmt.Sprintf("📊 Status: %s\n\n", recording.Status)
	successText += "⏳ Auto-detection is processing...\n"
	successText += "Check back in 10-30 seconds for results."

	edit := tgbotapi.NewEditMessageText(chatID, sent.MessageID, successText)
	edit.ParseMode = "HTML"
	b.api.Send(edit)

	// Offer to check status or create new recording
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🔄 Check Status",
				fmt.Sprintf("check:%s", recording.ID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"🎤 New Auto-Detect",
				"new_autodetect",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"📝 Manual Mode",
				"newrecord",
			),
		),
	)

	replyMsg := tgbotapi.NewMessage(chatID, "What would you like to do?")
	replyMsg.ReplyMarkup = keyboard
	b.api.Send(replyMsg)
}

func (b *Bot) handleDigitInput(ctx context.Context, msg *tgbotapi.Message, userID string, lang domain.Language, digit string) {
	// Get current input
	currentInput := b.service.GetAyahInput(ctx, userID)

	// Append digit (limit to 3 digits for ayah number)
	if len(currentInput) < 3 {
		currentInput += digit
		if err := b.service.SetAyahInput(ctx, userID, currentInput); err != nil {
			log.Printf("Error setting ayah input: %v", err)
			return
		}
	}

	// Get selected surah info
	surahNum, err := b.service.GetSelectedSurah(ctx, userID)
	if err != nil {
		log.Printf("Error getting selected surah: %v", err)
		return
	}

	surahs := b.service.GetAllSurahs()
	if surahNum < 1 || surahNum > len(surahs) {
		return
	}
	surah := surahs[surahNum-1]
	surahName := b.i18n.GetSurahName(lang, surahNum)

	// Update message with current input
	text := b.i18n.Get(lang, "ayah.select", surahName, surah.Ayahs)
	if currentInput != "" {
		text += fmt.Sprintf("\n\n📝 %s", currentInput)
	}

	b.editMessageWithKeyboard(msg, text, b.getAyahKeyboard(lang, currentInput))
}

func (b *Bot) handleClearDigit(ctx context.Context, msg *tgbotapi.Message, userID string, lang domain.Language) {
	// Get current input
	currentInput := b.service.GetAyahInput(ctx, userID)

	// Remove last digit
	if len(currentInput) > 0 {
		currentInput = currentInput[:len(currentInput)-1]
		if err := b.service.SetAyahInput(ctx, userID, currentInput); err != nil {
			log.Printf("Error setting ayah input: %v", err)
			return
		}
	}

	// Get selected surah info
	surahNum, err := b.service.GetSelectedSurah(ctx, userID)
	if err != nil {
		log.Printf("Error getting selected surah: %v", err)
		return
	}

	surahs := b.service.GetAllSurahs()
	if surahNum < 1 || surahNum > len(surahs) {
		return
	}
	surah := surahs[surahNum-1]
	surahName := b.i18n.GetSurahName(lang, surahNum)

	// Update message with current input
	text := b.i18n.Get(lang, "ayah.select", surahName, surah.Ayahs)
	if currentInput != "" {
		text += fmt.Sprintf("\n\n📝 %s", currentInput)
	}

	b.editMessageWithKeyboard(msg, text, b.getAyahKeyboard(lang, currentInput))
}

func (b *Bot) handleAyahDone(ctx context.Context, msg *tgbotapi.Message, userID string, lang domain.Language) {
	chatID := msg.Chat.ID

	// Get accumulated input
	ayahInput := b.service.GetAyahInput(ctx, userID)

	if ayahInput == "" {
		// Edit message to show error
		surahNum, _ := b.service.GetSelectedSurah(ctx, userID)
		surahs := b.service.GetAllSurahs()
		if surahNum >= 1 && surahNum <= len(surahs) {
			surah := surahs[surahNum-1]
			surahName := b.i18n.GetSurahName(lang, surahNum)
			text := b.i18n.Get(lang, "ayah.select", surahName, surah.Ayahs)
			text += "\n\n⚠️ " + b.i18n.Get(lang, "error.invalid_ayah")
			b.editMessageWithKeyboard(msg, text, b.getAyahKeyboard(lang, ""))
		}
		return
	}

	// Process ayah number
	if err := b.service.HandleAyahInput(ctx, userID, ayahInput); err != nil {
		log.Printf("Error handling ayah input: %v", err)

		// Edit message to show error
		surahNum, _ := b.service.GetSelectedSurah(ctx, userID)
		surahs := b.service.GetAllSurahs()
		if surahNum >= 1 && surahNum <= len(surahs) {
			surah := surahs[surahNum-1]
			surahName := b.i18n.GetSurahName(lang, surahNum)
			text := b.i18n.Get(lang, "ayah.select", surahName, surah.Ayahs)
			text += "\n\n⚠️ " + b.i18n.Get(lang, "error.invalid_ayah")
			b.editMessageWithKeyboard(msg, text, b.getAyahKeyboard(lang, ayahInput))
		}
		return
	}

	// Clear input after successful submission
	b.service.ClearAyahInput(ctx, userID)

	// Delete the keyboard message
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, msg.MessageID)
	b.api.Send(deleteMsg)

	// Send prompt for recording
	b.sendMessage(chatID, b.i18n.Get(lang, "recording.prompt"))
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}

func (b *Bot) sendLanguageSelection(chatID int64, currentLang domain.Language) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🇬🇧 English", "lang:en"),
			tgbotapi.NewInlineKeyboardButtonData("🇸🇦 العربية", "lang:ar"),
			tgbotapi.NewInlineKeyboardButtonData("🇷🇺 Русский", "lang:ru"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, b.i18n.Get(currentLang, "language.select"))
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendSurahSelection(ctx context.Context, chatID int64, userID string, lang domain.Language, page int) {
	keyboard := b.getSurahKeyboard(lang, page)
	msg := tgbotapi.NewMessage(chatID, b.i18n.Get(lang, "surah.select"))
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) editSurahSelection(ctx context.Context, msg *tgbotapi.Message, userID string, lang domain.Language, page int) {
	keyboard := b.getSurahKeyboard(lang, page)
	b.editMessageWithKeyboard(msg, b.i18n.Get(lang, "surah.select"), keyboard)
}

func (b *Bot) getSurahKeyboard(lang domain.Language, page int) tgbotapi.InlineKeyboardMarkup {
	surahs := b.service.GetAllSurahs()

	const itemsPerPage = 10
	totalPages := (len(surahs) + itemsPerPage - 1) / itemsPerPage

	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(surahs) {
		end = len(surahs)
	}

	var rows [][]tgbotapi.InlineKeyboardButton

	// Add surah buttons (2 per row)
	for i := start; i < end; i += 2 {
		surah1 := surahs[i]
		name1 := b.i18n.GetSurahName(lang, surah1.Number)
		btn1 := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d. %s", surah1.Number, name1),
			fmt.Sprintf("surah:%d", surah1.Number),
		)

		if i+1 < end {
			surah2 := surahs[i+1]
			name2 := b.i18n.GetSurahName(lang, surah2.Number)
			btn2 := tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%d. %s", surah2.Number, name2),
				fmt.Sprintf("surah:%d", surah2.Number),
			)
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn1, btn2))
		} else {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn1))
		}
	}

	// Add navigation buttons
	if totalPages > 1 {
		var navRow []tgbotapi.InlineKeyboardButton
		if page > 0 {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData("⬅️ "+b.i18n.Get(lang, "nav.prev"), fmt.Sprintf("spage:%d", page-1)))
		}
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d/%d", page+1, totalPages),
			"noop",
		))
		if page < totalPages-1 {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(b.i18n.Get(lang, "nav.next")+" ➡️", fmt.Sprintf("spage:%d", page+1)))
		}
		rows = append(rows, navRow)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) getAyahKeyboard(lang domain.Language, currentInput string) tgbotapi.InlineKeyboardMarkup {
	// Telephone-style number keyboard (3x3 + bottom row)
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1", "digit:1"),
			tgbotapi.NewInlineKeyboardButtonData("2", "digit:2"),
			tgbotapi.NewInlineKeyboardButtonData("3", "digit:3"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("4", "digit:4"),
			tgbotapi.NewInlineKeyboardButtonData("5", "digit:5"),
			tgbotapi.NewInlineKeyboardButtonData("6", "digit:6"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("7", "digit:7"),
			tgbotapi.NewInlineKeyboardButtonData("8", "digit:8"),
			tgbotapi.NewInlineKeyboardButtonData("9", "digit:9"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⬅️ "+b.i18n.Get(lang, "nav.back"), "clear"),
			tgbotapi.NewInlineKeyboardButtonData("0", "digit:0"),
			tgbotapi.NewInlineKeyboardButtonData("✅ "+b.i18n.Get(lang, "nav.done"), "done"),
		),
	)
}

func (b *Bot) editMessageWithKeyboard(msg *tgbotapi.Message, text string, keyboard tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, text)
	edit.ReplyMarkup = &keyboard
	if _, err := b.api.Send(edit); err != nil {
		log.Printf("Error editing message: %v", err)
	}
}

func (b *Bot) answerCallbackAlert(callbackID, text string) {
	callback := tgbotapi.NewCallbackWithAlert(callbackID, text)
	if _, err := b.api.Request(callback); err != nil {
		log.Printf("Error answering callback: %v", err)
	}
}

func (b *Bot) getUserID(update tgbotapi.Update) string {
	if update.Message != nil && update.Message.From != nil {
		return strconv.FormatInt(update.Message.From.ID, 10)
	}
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		return strconv.FormatInt(update.CallbackQuery.From.ID, 10)
	}
	return ""
}
