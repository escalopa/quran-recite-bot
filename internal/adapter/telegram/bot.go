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
	api     *tgbotapi.BotAPI
	service *application.BotService
	i18n    domain.I18nPort
	cancel  context.CancelFunc
}

func NewBot(token string, service *application.BotService, i18n domain.I18nPort) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}

	return &Bot{
		api:     api,
		service: service,
		i18n:    i18n,
	}, nil
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
	userID := strconv.FormatInt(msg.From.ID, 10)

	switch msg.Command() {
	case "start":
		b.handleStart(ctx, msg.Chat.ID, userID, lang)
	case "help":
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "help.message"))
	case "language":
		b.sendLanguageSelection(msg.Chat.ID, lang)
	default:
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.unknown_command"))
	}
}

func (b *Bot) handleStart(ctx context.Context, chatID int64, userID string, lang domain.Language) {
	if err := b.service.HandleStart(ctx, userID, lang); err != nil {
		log.Printf("Error handling start: %v", err)
		b.sendMessage(chatID, b.i18n.Get(lang, "error.generic"))
		return
	}

	// Send welcome message
	b.sendMessage(chatID, b.i18n.Get(lang, "welcome.message"))

	// Show surah selection
	b.sendSurahSelection(ctx, chatID, userID, lang, 0)
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
		b.sendSurahSelection(ctx, chatID, userID, lang, page)
		return
	}

	// Handle surah selection
	if len(data) > 6 && data[:6] == "surah:" {
		surahNum, err := strconv.Atoi(data[6:])
		if err != nil {
			b.sendMessage(chatID, b.i18n.Get(lang, "error.invalid_input"))
			return
		}

		if err := b.service.HandleSurahSelection(ctx, userID, surahNum); err != nil {
			log.Printf("Error selecting surah: %v", err)
			b.sendMessage(chatID, b.i18n.Get(lang, "error.generic"))
			return
		}

		// Get selected surah info
		surahs := b.service.GetAllSurahs()
		surah := surahs[surahNum-1]
		surahName := b.i18n.GetSurahName(lang, surahNum)

		// Send ayah number keyboard
		msg := b.i18n.Get(lang, "ayah.select", surahName, surah.Ayahs)
		b.sendMessage(chatID, msg)
		b.sendAyahKeyboard(chatID, lang)
		return
	}

	// Handle digit input
	if len(data) > 6 && data[:6] == "digit:" {
		b.handleDigitInput(ctx, chatID, userID, lang, data[6:])
		return
	}

	// Handle clear/backspace
	if data == "clear" {
		// Just inform the user
		b.sendMessage(chatID, b.i18n.Get(lang, "ayah.cleared"))
		return
	}

	// Handle done (when ayah number is entered)
	if data == "done" {
		// State will be updated when user sends voice
		msg := b.i18n.Get(lang, "recording.prompt")
		b.sendMessage(chatID, msg)
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
	if err != nil || state != domain.StateWaitRecording {
		b.sendMessage(chatID, b.i18n.Get(lang, "error.unexpected_voice"))
		return
	}

	// Download voice file
	fileConfig := tgbotapi.FileConfig{FileID: msg.Voice.FileID}
	file, err := b.api.GetFile(fileConfig)
	if err != nil {
		log.Printf("Error getting file: %v", err)
		b.sendMessage(chatID, b.i18n.Get(lang, "error.download_failed"))
		return
	}

	// Get file URL
	fileURL := file.Link(b.api.Token)

	// Download and convert to WAV (for now, we'll send a message about processing)
	// In production, you'd want to download the file and convert it to WAV format
	b.sendMessage(chatID, b.i18n.Get(lang, "recording.processing"))

	// For demo purposes, show that we would process it
	// In real implementation, download the file, convert to WAV, and submit
	_ = fileURL // Use this to download the file

	msg2 := tgbotapi.NewMessage(chatID, b.i18n.Get(lang, "recording.note_wav"))
	msg2.ParseMode = "HTML"
	b.api.Send(msg2)

	// Reset to start
	b.sendMessage(chatID, b.i18n.Get(lang, "recording.complete"))
	b.handleStart(ctx, chatID, userID, lang)
}

func (b *Bot) handleDigitInput(ctx context.Context, chatID int64, userID string, lang domain.Language, digit string) {
	// For simplicity, we'll handle direct text input instead
	// This is a placeholder for the digit keyboard interaction
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

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)
	msg := tgbotapi.NewMessage(chatID, b.i18n.Get(lang, "surah.select"))
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) sendAyahKeyboard(chatID int64, lang domain.Language) {
	// Telephone-style number keyboard (3x3 + bottom row)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
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

	msg := tgbotapi.NewMessage(chatID, b.i18n.Get(lang, "ayah.enter_number"))
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
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
