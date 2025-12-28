package quranapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/escalopa/quran-read-bot/internal/domain"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SubmitAutoDetect(ctx context.Context, learnerID string, audioFile io.Reader, expectedStartAyahID string, minSimilarity float64) (*domain.Recording, error) {
	// Read audio data
	audioData, err := io.ReadAll(audioFile)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("file", "recording.mp3")
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("write audio data: %w", err)
	}

	// Add optional expected_start_ayah_id as form data
	if expectedStartAyahID != "" {
		if err := writer.WriteField("expected_start_ayah_id", expectedStartAyahID); err != nil {
			return nil, fmt.Errorf("write expected_start_ayah_id: %w", err)
		}
	}

	// Add optional min_similarity as form data
	if minSimilarity > 0 {
		if err := writer.WriteField("min_similarity", fmt.Sprintf("%.2f", minSimilarity)); err != nil {
			return nil, fmt.Errorf("write min_similarity: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	// Create request with learner_id as query parameter
	reqURL := fmt.Sprintf("%s/recordings/auto-detect?learner_id=%s", c.baseURL, url.QueryEscape(learnerID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-api-key", c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		RecordingID string `json:"recording_id"`
		Status      string `json:"status"`
		TaskID      string `json:"task_id"`
		Message     string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	recording := &domain.Recording{
		ID:        result.RecordingID,
		LearnerID: learnerID,
		AyahID:    "", // Will be determined by auto-detection
		Status:    domain.RecordingStatus(result.Status),
		CreatedAt: time.Now(),
	}

	return recording, nil
}

// SubmitRecording submits a voice recording for analysis
func (c *Client) SubmitRecording(ctx context.Context, learnerID, ayahID string, audioFile io.Reader) (*domain.Recording, error) {
	// Read audio data
	audioData, err := io.ReadAll(audioFile)
	if err != nil {
		return nil, fmt.Errorf("read audio: %w", err)
	}

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", "recording.wav")
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}

	if _, err := part.Write(audioData); err != nil {
		return nil, fmt.Errorf("write audio data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/recordings?learner_id=%s&ayah_id=%s", c.baseURL, learnerID, ayahID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("x-api-key", c.apiKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result struct {
		RecordingID string `json:"recording_id"`
		Status      string `json:"status"`
		TaskID      string `json:"task_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	recording := &domain.Recording{
		ID:        result.RecordingID,
		LearnerID: learnerID,
		AyahID:    ayahID,
		Status:    domain.RecordingStatus(result.Status),
		CreatedAt: time.Now(),
	}

	return recording, nil
}

// GetRecording retrieves a recording by ID
func (c *Client) GetRecording(ctx context.Context, learnerID, recordingID string) (*domain.Recording, error) {
	url := fmt.Sprintf("%s/recordings?learner_id=%s&recording_ids=%s", c.baseURL, learnerID, recordingID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Recordings []recordingResponse `json:"recordings"`
		NotFound   []string            `json:"not_found"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(result.Recordings) == 0 {
		return nil, fmt.Errorf("recording not found")
	}

	return mapRecording(&result.Recordings[0]), nil
}

// ListRecordings lists all recordings for a learner
func (c *Client) ListRecordings(ctx context.Context, learnerID string, limit int) ([]*domain.Recording, error) {
	url := fmt.Sprintf("%s/recordings/%s?limit=%d", c.baseURL, learnerID, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Items []recordingResponse `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	recordings := make([]*domain.Recording, len(result.Items))
	for i, item := range result.Items {
		recordings[i] = mapRecording(&item)
	}

	return recordings, nil
}

type recordingResponse struct {
	RecordingID string          `json:"recording_id"`
	LearnerID   string          `json:"learner_id"`
	AyahID      string          `json:"ayah_id"`
	Status      string          `json:"status"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Result      *resultResponse `json:"result"`
	Error       string          `json:"error,omitempty"`
}

type resultResponse struct {
	// Auto-detect fields
	Status              string                  `json:"status"`
	DetectionMethod     string                  `json:"detection_method"`
	StartingAyah        string                  `json:"starting_ayah"`
	DetectionConfidence string                  `json:"detection_confidence"`
	Hypothesis          string                  `json:"hypothesis"`
	DetectedRange       *detectedRangeResponse  `json:"detected_range"`
	OverallStatistics   *statisticsResponse     `json:"overall_statistics"`
	PerAyahResults      []perAyahResultResponse `json:"per_ayah_results"`
	ProcessingTime      float64                 `json:"processing_time"`
	Error               string                  `json:"error"`
	Suggestion          string                  `json:"suggestion"`
	Transcript          string                  `json:"transcript"`
	TranscriptLength    int                     `json:"transcript_length"`
	// Legacy fields
	WER float64      `json:"wer"`
	Ops []opResponse `json:"ops"`
}

type detectedRangeResponse struct {
	StartAyah  string `json:"start_ayah"`
	EndAyah    string `json:"end_ayah"`
	TotalAyahs int    `json:"total_ayahs"`
}

type statisticsResponse struct {
	TotalWords    int     `json:"total_words"`
	Correct       int     `json:"correct"`
	Substitutions int     `json:"substitutions"`
	Deletions     int     `json:"deletions"`
	Insertions    int     `json:"insertions"`
	WER           float64 `json:"wer"`
	Accuracy      float64 `json:"accuracy"`
}

type perAyahResultResponse struct {
	AyahID        string              `json:"ayah_id"`
	Surah         string              `json:"surah"`
	Ayah          string              `json:"ayah"`
	Words         int                 `json:"words"`
	Correct       int                 `json:"correct"`
	Substitutions int                 `json:"substitutions"`
	Deletions     int                 `json:"deletions"`
	Insertions    int                 `json:"insertions"`
	WER           float64             `json:"wer"`
	ReferenceText string              `json:"reference_text"`
	Errors        []ayahErrorResponse `json:"errors"`
}

type ayahErrorResponse struct {
	Type     string `json:"type"`
	RefWord  string `json:"ref_word"`
	HypWord  string `json:"hyp_word"`
	Position int    `json:"position"`
}

type opResponse struct {
	RefAr    string  `json:"ref_ar"`
	RefClean string  `json:"ref_clean"`
	HypAr    string  `json:"hyp_ar"`
	HypClean string  `json:"hyp_clean"`
	Op       string  `json:"op"`
	TStart   float64 `json:"t_start"`
	TEnd     float64 `json:"t_end"`
}

func mapRecording(r *recordingResponse) *domain.Recording {
	recording := &domain.Recording{
		ID:        r.RecordingID,
		LearnerID: r.LearnerID,
		AyahID:    r.AyahID,
		Status:    domain.RecordingStatus(r.Status),
	}

	if r.CreatedAt != "" {
		recording.CreatedAt, _ = time.Parse(time.RFC3339, r.CreatedAt)
	}
	if r.UpdatedAt != "" {
		recording.UpdatedAt, _ = time.Parse(time.RFC3339, r.UpdatedAt)
	}

	if r.Result != nil {
		recording.Result = &domain.RecordingResult{
			Status:              r.Result.Status,
			DetectionMethod:     r.Result.DetectionMethod,
			StartingAyah:        r.Result.StartingAyah,
			DetectionConfidence: r.Result.DetectionConfidence,
			Hypothesis:          r.Result.Hypothesis,
			ProcessingTime:      r.Result.ProcessingTime,
			Error:               r.Result.Error,
			Suggestion:          r.Result.Suggestion,
			Transcript:          r.Result.Transcript,
			TranscriptLength:    r.Result.TranscriptLength,
			WER:                 r.Result.WER,
		}

		// Map detected range
		if r.Result.DetectedRange != nil {
			recording.Result.DetectedRange = &domain.DetectedRange{
				StartAyah:  r.Result.DetectedRange.StartAyah,
				EndAyah:    r.Result.DetectedRange.EndAyah,
				TotalAyahs: r.Result.DetectedRange.TotalAyahs,
			}
		}

		// Map overall statistics
		if r.Result.OverallStatistics != nil {
			recording.Result.OverallStatistics = &domain.Statistics{
				TotalWords:    r.Result.OverallStatistics.TotalWords,
				Correct:       r.Result.OverallStatistics.Correct,
				Substitutions: r.Result.OverallStatistics.Substitutions,
				Deletions:     r.Result.OverallStatistics.Deletions,
				Insertions:    r.Result.OverallStatistics.Insertions,
				WER:           r.Result.OverallStatistics.WER,
				Accuracy:      r.Result.OverallStatistics.Accuracy,
			}
		}

		// Map per-ayah results
		if len(r.Result.PerAyahResults) > 0 {
			recording.Result.PerAyahResults = make([]domain.PerAyahResult, len(r.Result.PerAyahResults))
			for i, ayahResult := range r.Result.PerAyahResults {
				perAyah := domain.PerAyahResult{
					AyahID:        ayahResult.AyahID,
					Surah:         ayahResult.Surah,
					Ayah:          ayahResult.Ayah,
					Words:         ayahResult.Words,
					Correct:       ayahResult.Correct,
					Substitutions: ayahResult.Substitutions,
					Deletions:     ayahResult.Deletions,
					Insertions:    ayahResult.Insertions,
					WER:           ayahResult.WER,
					ReferenceText: ayahResult.ReferenceText,
				}

				// Map errors
				if len(ayahResult.Errors) > 0 {
					perAyah.Errors = make([]domain.AyahError, len(ayahResult.Errors))
					for j, err := range ayahResult.Errors {
						perAyah.Errors[j] = domain.AyahError{
							Type:     err.Type,
							RefWord:  err.RefWord,
							HypWord:  err.HypWord,
							Position: err.Position,
						}
					}
				}

				recording.Result.PerAyahResults[i] = perAyah
			}
		}

		// Map legacy ops field if present
		if len(r.Result.Ops) > 0 {
			recording.Result.Ops = make([]domain.Operation, len(r.Result.Ops))
			for i, op := range r.Result.Ops {
				recording.Result.Ops[i] = domain.Operation{
					RefAr:    op.RefAr,
					RefClean: op.RefClean,
					HypAr:    op.HypAr,
					HypClean: op.HypClean,
					Op:       domain.OpType(op.Op),
					TStart:   op.TStart,
					TEnd:     op.TEnd,
				}
			}
		}
	}

	return recording
}
