package domain

import (
	"context"
	"io"
)

// QuranAPIPort defines the interface for interacting with the Quran reading API
type QuranAPIPort interface {
	// SubmitRecording submits a voice recording for analysis
	SubmitRecording(ctx context.Context, learnerID, ayahID string, audioFile io.Reader) (*Recording, error)

	// SubmitAutoDetect submits a voice recording for auto-detection of ayah(s)
	SubmitAutoDetect(ctx context.Context, learnerID string, audioFile io.Reader, expectedStartAyahID string, minSimilarity float64) (*Recording, error)

	// GetRecording retrieves a recording by ID
	GetRecording(ctx context.Context, learnerID, recordingID string) (*Recording, error)

	// ListRecordings lists all recordings for a learner
	ListRecordings(ctx context.Context, learnerID string, limit int) ([]*Recording, error)
}

// FSMPort defines the interface for finite state machine storage
type FSMPort interface {
	// SetState sets the current state for a user
	SetState(ctx context.Context, userID string, state State) error

	// GetState gets the current state for a user
	GetState(ctx context.Context, userID string) (State, error)

	// DeleteState deletes the state for a user
	DeleteState(ctx context.Context, userID string) error

	// SetData sets temporary data for a user's current session
	SetData(ctx context.Context, userID, key, value string) error

	// GetData gets temporary data for a user's current session
	GetData(ctx context.Context, userID, key string) (string, error)

	// DeleteData deletes temporary data for a user
	DeleteData(ctx context.Context, userID, key string) error
}

// I18nPort defines the interface for internationalization
type I18nPort interface {
	// Get retrieves a translated message
	Get(lang Language, key string, args ...interface{}) string

	// GetSurahName retrieves the localized name of a Surah
	GetSurahName(lang Language, surahNumber int) string
}

// BotPort defines the interface for the bot adapter
type BotPort interface {
	// Start starts the bot
	Start(ctx context.Context) error

	// Stop stops the bot
	Stop() error
}

// State represents the FSM states
type State string

const (
	StateStart            State = "start"
	StateSelectSurah      State = "select_surah"
	StateEnterAyah        State = "enter_ayah"
	StateWaitRecording    State = "wait_recording"
	StateWaitAutoDetect   State = "wait_auto_detect"
	StateProcessing       State = "processing"
)

// SessionData keys
const (
	SessionKeySurah     = "surah"
	SessionKeyAyah      = "ayah"
	SessionKeyAyahInput = "ayah_input" // Accumulated digit input for ayah number
	SessionKeyLanguage  = "language"
)
