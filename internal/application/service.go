package application

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/escalopa/quran-read-bot/internal/domain"
)

// BotService handles the business logic for the bot
type BotService struct {
	quranAPI domain.QuranAPIPort
	fsm      domain.FSMPort
	i18n     domain.I18nPort
}

func NewBotService(quranAPI domain.QuranAPIPort, fsm domain.FSMPort, i18n domain.I18nPort) *BotService {
	return &BotService{
		quranAPI: quranAPI,
		fsm:      fsm,
		i18n:     i18n,
	}
}

// HandleStart handles the /start command
func (s *BotService) HandleStart(ctx context.Context, userID string, lang domain.Language) error {
	// Set initial state
	if err := s.fsm.SetState(ctx, userID, domain.StateSelectSurah); err != nil {
		return fmt.Errorf("set state: %w", err)
	}

	// Store user language
	if err := s.fsm.SetData(ctx, userID, domain.SessionKeyLanguage, string(lang)); err != nil {
		return fmt.Errorf("set language: %w", err)
	}

	return nil
}

// GetCurrentState returns the current state for a user
func (s *BotService) GetCurrentState(ctx context.Context, userID string) (domain.State, error) {
	return s.fsm.GetState(ctx, userID)
}

// HandleSurahSelection handles when a user selects a Surah
func (s *BotService) HandleSurahSelection(ctx context.Context, userID string, surahNumber int) error {
	// Validate surah number
	surahs := domain.GetAllSurahs()
	if surahNumber < 1 || surahNumber > len(surahs) {
		return fmt.Errorf("invalid surah number: %d", surahNumber)
	}

	// Store selected surah
	if err := s.fsm.SetData(ctx, userID, domain.SessionKeySurah, strconv.Itoa(surahNumber)); err != nil {
		return fmt.Errorf("set surah: %w", err)
	}

	// Move to next state
	if err := s.fsm.SetState(ctx, userID, domain.StateEnterAyah); err != nil {
		return fmt.Errorf("set state: %w", err)
	}

	return nil
}

// HandleAyahInput handles when a user enters an Ayah number
func (s *BotService) HandleAyahInput(ctx context.Context, userID, input string) error {
	// Parse ayah number
	ayahNumber, err := strconv.Atoi(input)
	if err != nil {
		return fmt.Errorf("invalid ayah number: %s", input)
	}

	// Get selected surah
	surahStr, err := s.fsm.GetData(ctx, userID, domain.SessionKeySurah)
	if err != nil {
		return fmt.Errorf("get surah: %w", err)
	}

	surahNumber, err := strconv.Atoi(surahStr)
	if err != nil {
		return fmt.Errorf("parse surah: %w", err)
	}

	// Validate ayah number
	surahs := domain.GetAllSurahs()
	if surahNumber < 1 || surahNumber > len(surahs) {
		return fmt.Errorf("invalid surah: %d", surahNumber)
	}

	surah := surahs[surahNumber-1]
	if ayahNumber < 1 || ayahNumber > surah.Ayahs {
		return fmt.Errorf("invalid ayah number: %d (surah %d has %d ayahs)", ayahNumber, surahNumber, surah.Ayahs)
	}

	// Store ayah number
	if err := s.fsm.SetData(ctx, userID, domain.SessionKeyAyah, strconv.Itoa(ayahNumber)); err != nil {
		return fmt.Errorf("set ayah: %w", err)
	}

	// Move to next state
	if err := s.fsm.SetState(ctx, userID, domain.StateWaitRecording); err != nil {
		return fmt.Errorf("set state: %w", err)
	}

	return nil
}

// HandleRecording handles when a user sends a voice recording
func (s *BotService) HandleRecording(ctx context.Context, userID string, audioFile io.Reader) (*domain.Recording, error) {
	// Get surah and ayah
	surahStr, err := s.fsm.GetData(ctx, userID, domain.SessionKeySurah)
	if err != nil {
		return nil, fmt.Errorf("get surah: %w", err)
	}

	ayahStr, err := s.fsm.GetData(ctx, userID, domain.SessionKeyAyah)
	if err != nil {
		return nil, fmt.Errorf("get ayah: %w", err)
	}

	surahNumber, _ := strconv.Atoi(surahStr)
	ayahNumber, _ := strconv.Atoi(ayahStr)

	ayahID := domain.FormatAyahID(surahNumber, ayahNumber)

	// Submit recording to API
	recording, err := s.quranAPI.SubmitRecording(ctx, userID, ayahID, audioFile)
	if err != nil {
		return nil, fmt.Errorf("submit recording: %w", err)
	}

	// Reset state to allow new recording
	if err := s.fsm.SetState(ctx, userID, domain.StateSelectSurah); err != nil {
		return nil, fmt.Errorf("reset state: %w", err)
	}

	return recording, nil
}

// GetUserLanguage retrieves the user's preferred language
func (s *BotService) GetUserLanguage(ctx context.Context, userID string) domain.Language {
	langStr, err := s.fsm.GetData(ctx, userID, domain.SessionKeyLanguage)
	if err != nil || langStr == "" {
		return domain.LangEnglish // default
	}
	return domain.Language(langStr)
}

// FormatRecordingResult formats the recording result for display
func (s *BotService) FormatRecordingResult(lang domain.Language, recording *domain.Recording) string {
	if recording.Result == nil {
		return s.i18n.Get(lang, "recording.processing")
	}

	var sb strings.Builder

	// Show WER (Word Error Rate)
	sb.WriteString(fmt.Sprintf("%s: %.2f%%\n\n", s.i18n.Get(lang, "recording.wer"), recording.Result.WER*100))

	// Show word-by-word analysis
	sb.WriteString(s.i18n.Get(lang, "recording.analysis"))
	sb.WriteString("\n")

	for _, op := range recording.Result.Ops {
		emoji := ""
		switch op.Op {
		case domain.OpCorrect:
			emoji = "✅"
		case domain.OpSubstitution:
			emoji = "🔄"
		case domain.OpDeletion:
			emoji = "❌"
		case domain.OpInsertion:
			emoji = "➕"
		}

		sb.WriteString(fmt.Sprintf("%s %s (%s)\n", emoji, op.RefAr, op.Op))
	}

	return sb.String()
}

// GetSelectedSurah returns the currently selected surah for a user
func (s *BotService) GetSelectedSurah(ctx context.Context, userID string) (int, error) {
	surahStr, err := s.fsm.GetData(ctx, userID, domain.SessionKeySurah)
	if err != nil {
		return 0, fmt.Errorf("get surah: %w", err)
	}

	return strconv.Atoi(surahStr)
}

// GetAllSurahs returns all surahs
func (s *BotService) GetAllSurahs() []domain.Surah {
	return domain.GetAllSurahs()
}

// GetAyahInput gets the accumulated ayah input for a user
func (s *BotService) GetAyahInput(ctx context.Context, userID string) string {
	input, err := s.fsm.GetData(ctx, userID, domain.SessionKeyAyahInput)
	if err != nil {
		return ""
	}
	return input
}

// SetAyahInput sets the accumulated ayah input for a user
func (s *BotService) SetAyahInput(ctx context.Context, userID, input string) error {
	return s.fsm.SetData(ctx, userID, domain.SessionKeyAyahInput, input)
}

// ClearAyahInput clears the accumulated ayah input for a user
func (s *BotService) ClearAyahInput(ctx context.Context, userID string) error {
	return s.fsm.DeleteData(ctx, userID, domain.SessionKeyAyahInput)
}

// GetRecording retrieves a specific recording by ID
func (s *BotService) GetRecording(ctx context.Context, userID, recordingID string) (*domain.Recording, error) {
	return s.quranAPI.GetRecording(ctx, userID, recordingID)
}

// ListRecordings retrieves all recordings for a user
func (s *BotService) ListRecordings(ctx context.Context, userID string, limit int) ([]*domain.Recording, error) {
	return s.quranAPI.ListRecordings(ctx, userID, limit)
}
