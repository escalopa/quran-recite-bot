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

	edit := tgbotapi.NewEditMessageText(msg.Chat.ID, msg.MessageID, text)
	edit.ReplyMarkup = &keyboard
	edit.ParseMode = "HTML"
	b.api.Send(edit)
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

		// Parse ayah ID to get surah and ayah numbers
		surahNum, ayahNum := b.parseAyahID(rec.AyahID)
		surahName := b.i18n.GetSurahName(lang, surahNum)

		btnText := fmt.Sprintf("%s %s:%d - %s", status, surahName, ayahNum, date)
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

	// Check if this is an auto-detect recording
	isAutoDetect := recording.Result != nil && recording.Result.DetectionMethod != ""

	if isAutoDetect && recording.Result.Status == "matched" {
		// Auto-detect result - show detected ayahs
		text.WriteString(fmt.Sprintf("🔍 Mode: <b>Auto-Detect (%s)</b>\n", recording.Result.DetectionConfidence))

		if recording.Result.DetectedRange != nil {
			text.WriteString(fmt.Sprintf("📖 Detected: <b>Ayah %s", recording.Result.DetectedRange.StartAyah))
			if recording.Result.DetectedRange.StartAyah != recording.Result.DetectedRange.EndAyah {
				text.WriteString(fmt.Sprintf(" - %s", recording.Result.DetectedRange.EndAyah))
			}
			text.WriteString(fmt.Sprintf("</b> (%d ayah", recording.Result.DetectedRange.TotalAyahs))
			if recording.Result.DetectedRange.TotalAyahs > 1 {
				text.WriteString("s")
			}
			text.WriteString(")\n")
		}
	} else if recording.AyahID != "" {
		// Manual mode - show specified ayah
		surahNum, ayahNum := b.parseAyahID(recording.AyahID)
		surahName := b.i18n.GetSurahName(lang, surahNum)
		text.WriteString(fmt.Sprintf("📖 Surah: <b>%s</b>\n", surahName))
		text.WriteString(fmt.Sprintf("📄 %s: <b>%d</b>\n", b.i18n.Get(lang, "ayah.ayah"), ayahNum))
	}

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
		if recording.Result.Status == "no_match" {
			// Auto-detect failed
			text.WriteString("<b>❌ No Match Found</b>\n\n")
			text.WriteString(fmt.Sprintf("Error: %s\n", recording.Result.Error))
			if recording.Result.Suggestion != "" {
				text.WriteString(fmt.Sprintf("💡 Suggestion: %s\n", recording.Result.Suggestion))
			}
		} else if recording.Result.Status == "matched" {
			// Auto-detect succeeded - show statistics
			text.WriteString(fmt.Sprintf("<b>%s</b>\n", b.i18n.Get(lang, "recording.results")))

			if recording.Result.OverallStatistics != nil {
				stats := recording.Result.OverallStatistics
				text.WriteString(fmt.Sprintf("📊 Accuracy: <b>%.1f%%</b>\n", stats.Accuracy*100))
				text.WriteString(fmt.Sprintf("📊 WER: <b>%.1f%%</b>\n", stats.WER*100))
				text.WriteString(fmt.Sprintf("✅ Correct: %d/%d words\n", stats.Correct, stats.TotalWords))
				if stats.Substitutions > 0 {
					text.WriteString(fmt.Sprintf("🔄 Substitutions: %d\n", stats.Substitutions))
				}
				if stats.Deletions > 0 {
					text.WriteString(fmt.Sprintf("❌ Deletions: %d\n", stats.Deletions))
				}
				if stats.Insertions > 0 {
					text.WriteString(fmt.Sprintf("➕ Insertions: %d\n", stats.Insertions))
				}
				text.WriteString("\n")
			}

			// Show per-ayah results if available
			if len(recording.Result.PerAyahResults) > 0 {
				text.WriteString("<b>Per-Ayah Results:</b>\n")
				for _, ayahResult := range recording.Result.PerAyahResults {
					errorCount := len(ayahResult.Errors)
					text.WriteString(fmt.Sprintf("📄 %s: %.1f%% WER",
						ayahResult.AyahID,
						ayahResult.WER*100))
					if errorCount > 0 {
						text.WriteString(fmt.Sprintf(" (%d error", errorCount))
						if errorCount > 1 {
							text.WriteString("s")
						}
						text.WriteString(")")
					}
					text.WriteString("\n")
				}
				text.WriteString("\n")
			}

			if recording.Result.Hypothesis != "" {
				text.WriteString(fmt.Sprintf("<b>%s:</b>\n<code>%s</code>\n",
					b.i18n.Get(lang, "recording.transcription"),
					recording.Result.Hypothesis,
				))
			}
		} else {
			// Legacy format
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
		}
	} else if recording.Status == domain.StatusQueued || recording.Status == domain.StatusProcessing {
		text.WriteString(fmt.Sprintf("⏳ %s\n", b.i18n.Get(lang, "recording.processing")))
	}

	return text.String()
}

// getStatusEmoji returns emoji for recording status
func (b *Bot) getStatusEmoji(status domain.RecordingStatus) string {
	switch status {
	case domain.StatusQueued:
		return "⏳"
	case domain.StatusProcessing:
		return "🔄"
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

// parseAyahID parses ayah ID (XXXYYY format) to surah and ayah numbers
func (b *Bot) parseAyahID(ayahID string) (surahNum int, ayahNum int) {
	if len(ayahID) != 6 {
		return 0, 0
	}

	// Parse surah number (first 3 digits)
	fmt.Sscanf(ayahID[:3], "%d", &surahNum)
	// Parse ayah number (last 3 digits)
	fmt.Sscanf(ayahID[3:], "%d", &ayahNum)

	return surahNum, ayahNum
}
