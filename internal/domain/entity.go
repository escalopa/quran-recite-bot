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
	StatusQueued     RecordingStatus = "queued"
	StatusProcessing RecordingStatus = "processing"
	StatusDone       RecordingStatus = "done"
	StatusFailed     RecordingStatus = "failed"
)

// RecordingResult represents the analysis result of a recording
type RecordingResult struct {
	Status              string               `json:"status"`               // "matched" or "no_match"
	DetectionMethod     string               `json:"detection_method"`     // "auto" or "hint"
	StartingAyah        string               `json:"starting_ayah"`        // e.g., "110001"
	DetectionConfidence string               `json:"detection_confidence"` // "high", "medium", "low"
	Hypothesis          string               `json:"hypothesis"`
	DetectedRange       *DetectedRange       `json:"detected_range"`
	OverallStatistics   *Statistics          `json:"overall_statistics"`
	PerAyahResults      []PerAyahResult      `json:"per_ayah_results"`
	ProcessingTime      float64              `json:"processing_time"`
	Error               string               `json:"error,omitempty"`
	Suggestion          string               `json:"suggestion,omitempty"`
	Transcript          string               `json:"transcript,omitempty"`
	TranscriptLength    int                  `json:"transcript_length,omitempty"`
	WER                 float64              `json:"wer,omitempty"`         // Legacy field
	Ops                 []Operation          `json:"ops,omitempty"`         // Legacy field
}

// DetectedRange represents the range of ayahs detected in the recording
type DetectedRange struct {
	StartAyah  string `json:"start_ayah"`
	EndAyah    string `json:"end_ayah"`
	TotalAyahs int    `json:"total_ayahs"`
}

// Statistics represents overall statistics for the recording
type Statistics struct {
	TotalWords    int     `json:"total_words"`
	Correct       int     `json:"correct"`
	Substitutions int     `json:"substitutions"`
	Deletions     int     `json:"deletions"`
	Insertions    int     `json:"insertions"`
	WER           float64 `json:"wer"`
	Accuracy      float64 `json:"accuracy"`
}

// PerAyahResult represents the result for a single ayah
type PerAyahResult struct {
	AyahID        string       `json:"ayah_id"`
	Surah         string       `json:"surah"`
	Ayah          string       `json:"ayah"`
	Words         int          `json:"words"`
	Correct       int          `json:"correct"`
	Substitutions int          `json:"substitutions"`
	Deletions     int          `json:"deletions"`
	Insertions    int          `json:"insertions"`
	WER           float64      `json:"wer"`
	ReferenceText string       `json:"reference_text"`
	Errors        []AyahError  `json:"errors"`
}

// AyahError represents an error in an ayah
type AyahError struct {
	Type     string `json:"type"`      // "substitution", "deletion", "insertion"
	RefWord  string `json:"ref_word"`
	HypWord  string `json:"hyp_word"`
	Position int    `json:"position"`
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
