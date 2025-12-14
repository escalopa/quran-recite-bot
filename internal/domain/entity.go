package domain

import "time"

// Surah represents a chapter in the Quran
type Surah struct {
	Number int
	Name   string
	Ayahs  int
}

// Ayah represents a verse in the Quran
type Ayah struct {
	SurahNumber int
	AyahNumber  int
}

// AyahID returns the formatted ayah ID (XXXYYY format)
func (a Ayah) AyahID() string {
	return FormatAyahID(a.SurahNumber, a.AyahNumber)
}

// Recording represents a Quran recording submission
type Recording struct {
	ID        string
	LearnerID string
	AyahID    string
	Status    RecordingStatus
	Result    *RecordingResult
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RecordingStatus string

const (
	StatusQueued RecordingStatus = "queued"
	StatusDone   RecordingStatus = "done"
	StatusFailed RecordingStatus = "failed"
)

// RecordingResult represents the analysis result of a recording
type RecordingResult struct {
	WER        float64
	Ops        []Operation
	Hypothesis string
}

// Operation represents a word-level operation in the recording analysis
type Operation struct {
	RefAr    string  `json:"ref_ar"`
	RefClean string  `json:"ref_clean"`
	HypAr    string  `json:"hyp_ar"`
	HypClean string  `json:"hyp_clean"`
	Op       OpType  `json:"op"`
	TStart   float64 `json:"t_start"`
	TEnd     float64 `json:"t_end"`
}

type OpType string

const (
	OpCorrect      OpType = "C" // Correct
	OpSubstitution OpType = "S" // Substitution (wrong word)
	OpDeletion     OpType = "D" // Deletion (missing word)
	OpInsertion    OpType = "I" // Insertion (extra word)
)

// Language represents supported languages
type Language string

const (
	LangEnglish Language = "en"
	LangArabic  Language = "ar"
	LangRussian Language = "ru"
)
