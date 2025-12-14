package quranapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
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
}

type resultResponse struct {
	WER        float64      `json:"wer"`
	Ops        []opResponse `json:"ops"`
	Hypothesis string       `json:"hypothesis"`
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
			WER:        r.Result.WER,
			Hypothesis: r.Result.Hypothesis,
			Ops:        make([]domain.Operation, len(r.Result.Ops)),
		}

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

	return recording
}
