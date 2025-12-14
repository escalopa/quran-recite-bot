package telegram

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/escalopa/quran-read-bot/internal/domain"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// handleCheckRecording checks the status of a recording
func (b *Bot) handleCheckRecording(ctx context.Context, msg *tgbotapi.Message, userID string, lang domain.Language, recordingID string) {
	chatID := msg.Chat.ID

	recording, err := b.service.GetRecording(ctx, userID, recordingID)
	if err != nil {
		log.Printf("Error getting recording: %v", err)
		b.sendMessage(chatID, b.i18n.Get(lang, "error.recording_not_found"))
		return
	}

	// Format recording details
	text := b.formatRecordingDetails(lang, recording)

	// Send as new message or edit existing
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, msg.MessageID)
	b.api.Send(deleteMsg)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "recording.refresh"),
				fmt.Sprintf("check:%s", recordingID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "recording.new"),
				"newrecord",
			),
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "nav.back"),
				"backtorecs",
			),
		),
	)

	newMsg := tgbotapi.NewMessage(chatID, text)
	newMsg.ReplyMarkup = keyboard
	newMsg.ParseMode = "HTML"
	b.api.Send(newMsg)
}

// handleViewRecording shows details of a specific recording
func (b *Bot) handleViewRecording(ctx context.Context, msg *tgbotapi.Message, userID string, lang domain.Language, recordingID string) {
	recording, err := b.service.GetRecording(ctx, userID, recordingID)
	if err != nil {
		log.Printf("Error getting recording: %v", err)
		b.sendMessage(msg.Chat.ID, b.i18n.Get(lang, "error.recording_not_found"))
		return
	}

	text := b.formatRecordingDetails(lang, recording)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "recording.refresh"),
				fmt.Sprintf("viewrec:%s", recordingID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "nav.back"),
				"backtorecs",
			),
		),
	)

	b.editMessageWithKeyboard(msg, text, keyboard)
}

// sendRecordingsList sends a paginated list of recordings
func (b *Bot) sendRecordingsList(chatID int64, userID string, lang domain.Language, recordings []*domain.Recording, page int) {
	text, keyboard := b.formatRecordingsList(lang, recordings, page)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard
	msg.ParseMode = "HTML"
	b.api.Send(msg)
}

// editRecordingsList edits message with paginated list of recordings
func (b *Bot) editRecordingsList(msg *tgbotapi.Message, userID string, lang domain.Language, recordings []*domain.Recording, page int) {
	text, keyboard := b.formatRecordingsList(lang, recordings, page)

	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, text)
	edit.ReplyMarkup = &keyboard
	edit.ParseMode = "HTML"
	b.api.Send(edit)
}

// formatRecordingsList formats recordings into paginated list with keyboard
func (b *Bot) formatRecordingsList(lang domain.Language, recordings []*domain.Recording, page int) (string, tgbotapi.InlineKeyboardMarkup) {
	const itemsPerPage = 5
	totalPages := (len(recordings) + itemsPerPage - 1) / itemsPerPage

	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * itemsPerPage
	end := start + itemsPerPage
	if end > len(recordings) {
		end = len(recordings)
	}

	var text strings.Builder
	text.WriteString(fmt.Sprintf("<b>%s</b>\n\n", b.i18n.Get(lang, "recordings.title")))
	text.WriteString(fmt.Sprintf("%s: %d\n\n", b.i18n.Get(lang, "recordings.total"), len(recordings)))

	var rows [][]tgbotapi.InlineKeyboardButton

	// Add recording buttons
	for i := start; i < end; i++ {
		rec := recordings[i]
		status := b.getStatusEmoji(rec.Status)
		date := rec.CreatedAt.Format("2006-01-02 15:04")

		btnText := fmt.Sprintf("%s %s - %s", status, rec.AyahID, date)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("viewrec:%s", rec.ID)),
		))
	}

	// Add navigation buttons
	if totalPages > 1 {
		var navRow []tgbotapi.InlineKeyboardButton
		if page > 0 {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(
				"⬅️ "+b.i18n.Get(lang, "nav.prev"),
				fmt.Sprintf("recpage:%d", page-1),
			))
		}
		navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d/%d", page+1, totalPages),
			"noop",
		))
		if page < totalPages-1 {
			navRow = append(navRow, tgbotapi.NewInlineKeyboardButtonData(
				b.i18n.Get(lang, "nav.next")+" ➡️",
				fmt.Sprintf("recpage:%d", page+1),
			))
		}
		rows = append(rows, navRow)
	}

	// Add new recording button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(
			"➕ "+b.i18n.Get(lang, "recording.new"),
			"newrecord",
		),
	))

	return text.String(), tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// formatRecordingDetails formats detailed recording information
func (b *Bot) formatRecordingDetails(lang domain.Language, recording *domain.Recording) string {
	var text strings.Builder

	text.WriteString(fmt.Sprintf("<b>%s</b>\n\n", b.i18n.Get(lang, "recording.details")))
	text.WriteString(fmt.Sprintf("🆔 ID: <code>%s</code>\n", recording.ID))
	text.WriteString(fmt.Sprintf("📖 Ayah: <b>%s</b>\n", recording.AyahID))
	text.WriteString(fmt.Sprintf("📅 %s: %s\n",
		b.i18n.Get(lang, "recording.created"),
		recording.CreatedAt.Format(time.RFC822),
	))
	text.WriteString(fmt.Sprintf("🔄 %s: %s %s\n\n",
		b.i18n.Get(lang, "recording.status"),
		b.getStatusEmoji(recording.Status),
		recording.Status,
	))

	// Show results if available
	if recording.Result != nil {
		text.WriteString(fmt.Sprintf("<b>%s</b>\n", b.i18n.Get(lang, "recording.results")))
		text.WriteString(fmt.Sprintf("📊 WER: <b>%.2f%%</b>\n\n", recording.Result.WER*100))

		if len(recording.Result.Ops) > 0 {
			text.WriteString(fmt.Sprintf("<b>%s:</b>\n", b.i18n.Get(lang, "recording.analysis")))
			for i, op := range recording.Result.Ops {
				if i >= 20 { // Limit to first 20 words
					text.WriteString(fmt.Sprintf("\n... (%d %s)\n",
						len(recording.Result.Ops)-20,
						b.i18n.Get(lang, "recording.more_words"),
					))
					break
				}
				emoji := b.getOpEmoji(op.Op)
				text.WriteString(fmt.Sprintf("%s <code>%s</code>\n", emoji, op.RefAr))
			}
		}

		if recording.Result.Hypothesis != "" {
			text.WriteString(fmt.Sprintf("\n<b>%s:</b>\n<code>%s</code>\n",
				b.i18n.Get(lang, "recording.transcription"),
				recording.Result.Hypothesis,
			))
		}
	} else if recording.Status == domain.StatusQueued {
		text.WriteString(fmt.Sprintf("⏳ %s\n", b.i18n.Get(lang, "recording.processing")))
	}

	return text.String()
}

// getStatusEmoji returns emoji for recording status
func (b *Bot) getStatusEmoji(status domain.RecordingStatus) string {
	switch status {
	case domain.StatusQueued:
		return "⏳"
	case domain.StatusDone:
		return "✅"
	case domain.StatusFailed:
		return "❌"
	default:
		return "❓"
	}
}

// getOpEmoji returns emoji for operation type
func (b *Bot) getOpEmoji(op domain.OpType) string {
	switch op {
	case domain.OpCorrect:
		return "✅"
	case domain.OpSubstitution:
		return "🔄"
	case domain.OpDeletion:
		return "❌"
	case domain.OpInsertion:
		return "➕"
	default:
		return "❓"
	}
}
