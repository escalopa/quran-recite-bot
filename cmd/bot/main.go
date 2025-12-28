package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/escalopa/quran-read-bot/internal/adapter/i18n"
	"github.com/escalopa/quran-read-bot/internal/adapter/quranapi"
	"github.com/escalopa/quran-read-bot/internal/adapter/redis"
	"github.com/escalopa/quran-read-bot/internal/adapter/telegram"
	"github.com/escalopa/quran-read-bot/internal/application"
	"github.com/escalopa/quran-read-bot/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	log.Println("Configuration loaded successfully")

	// Initialize i18n
	i18nService, err := i18n.NewI18n(cfg.App.LocalesDir)
	if err != nil {
		return err
	}
	log.Println("i18n initialized")

	// Initialize Redis FSM
	fsm, err := redis.NewFSM(cfg.Redis.URI)
	if err != nil {
		return err
	}
	defer fsm.Close()
	log.Println("Redis FSM connected")

	// Initialize Quran API client
	quranAPIClient := quranapi.NewClient(cfg.QuranAPI.BaseURL, cfg.QuranAPI.APIKey)
	log.Println("Quran API client initialized")

	// Initialize application service
	botService := application.NewBotService(quranAPIClient, fsm, i18nService)
	log.Println("Bot service initialized")

	// Initialize Telegram bot
	bot, err := telegram.NewBot(cfg.Telegram.Token, botService, i18nService)
	if err != nil {
		return err
	}
	log.Println("Telegram bot initialized")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start bot in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Println("Starting bot...")
		if err := bot.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("Received shutdown signal, stopping bot...")
		cancel()
		if err := bot.Stop(); err != nil {
			log.Printf("Error stopping bot: %v", err)
		}
	case err := <-errChan:
		log.Printf("Bot error: %v", err)
		return err
	}

	log.Println("Bot stopped successfully")
	return nil
}
