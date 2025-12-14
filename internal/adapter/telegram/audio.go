package telegram

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// downloadFile downloads a file from Telegram
func (b *Bot) downloadFile(fileURL string) ([]byte, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return data, nil
}

// convertOGGtoWAV converts OGG audio to WAV format using FFmpeg
func convertOGGtoWAV(oggData []byte) ([]byte, error) {
	// Check if FFmpeg is available
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("ffmpeg not found: %w", err)
	}

	// Create temporary files
	tmpDir := os.TempDir()
	oggFile := filepath.Join(tmpDir, fmt.Sprintf("audio_%d.ogg", os.Getpid()))
	wavFile := filepath.Join(tmpDir, fmt.Sprintf("audio_%d.wav", os.Getpid()))

	// Cleanup
	defer func() {
		os.Remove(oggFile)
		os.Remove(wavFile)
	}()

	// Write OGG data to file
	if err := os.WriteFile(oggFile, oggData, 0644); err != nil {
		return nil, fmt.Errorf("write ogg file: %w", err)
	}

	// Convert using FFmpeg
	// -i input file
	// -ar 16000 sample rate (16kHz is good for speech)
	// -ac 1 mono audio
	// -y overwrite output file
	cmd := exec.Command("ffmpeg",
		"-i", oggFile,
		"-ar", "16000",
		"-ac", "1",
		"-y",
		wavFile,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error: %s", stderr.String())
		return nil, fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	// Read WAV file
	wavData, err := os.ReadFile(wavFile)
	if err != nil {
		return nil, fmt.Errorf("read wav file: %w", err)
	}

	return wavData, nil
}

// processVoiceMessage downloads and converts a Telegram voice message to WAV
func (b *Bot) processVoiceMessage(fileID string) (io.Reader, error) {
	// Get file info from Telegram
	fileConfig := tgbotapi.FileConfig{FileID: fileID}
	file, err := b.api.GetFile(fileConfig)
	if err != nil {
		return nil, fmt.Errorf("get file info: %w", err)
	}

	// Download OGG file
	fileURL := file.Link(b.api.Token)
	oggData, err := b.downloadFile(fileURL)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}

	// Convert to WAV
	wavData, err := convertOGGtoWAV(oggData)
	if err != nil {
		return nil, fmt.Errorf("convert audio: %w", err)
	}

	return bytes.NewReader(wavData), nil
}
