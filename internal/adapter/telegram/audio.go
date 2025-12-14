package telegram

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

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

	// Create unique temporary OGG file
	oggFile, err := os.CreateTemp("", "quran-audio-*.ogg")
	if err != nil {
		return nil, fmt.Errorf("create temp ogg file: %w", err)
	}
	oggPath := oggFile.Name()

	// Create unique temporary WAV file
	wavFile, err := os.CreateTemp("", "quran-audio-*.wav")
	if err != nil {
		oggFile.Close()
		os.Remove(oggPath)
		return nil, fmt.Errorf("create temp wav file: %w", err)
	}
	wavPath := wavFile.Name()
	wavFile.Close() // Close immediately since ffmpeg will write to it

	// Cleanup temporary files
	defer func() {
		os.Remove(oggPath)
		os.Remove(wavPath)
	}()

	// Write OGG data to temporary file
	if _, err := oggFile.Write(oggData); err != nil {
		oggFile.Close()
		return nil, fmt.Errorf("write ogg data: %w", err)
	}

	if err := oggFile.Close(); err != nil {
		return nil, fmt.Errorf("close ogg file: %w", err)
	}

	// Convert using FFmpeg
	// -i input file
	// -ar 16000 sample rate (16kHz is good for speech)
	// -ac 1 mono audio
	// -y overwrite output file
	cmd := exec.Command("ffmpeg",
		"-i", oggPath,
		"-ar", "16000",
		"-ac", "1",
		"-y",
		wavPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Printf("FFmpeg error: %s", stderr.String())
		return nil, fmt.Errorf("ffmpeg conversion failed: %w", err)
	}

	// Read converted WAV file
	wavData, err := os.ReadFile(wavPath)
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
