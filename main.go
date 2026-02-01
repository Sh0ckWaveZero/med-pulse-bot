package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot-med/bot"
	"telegram-bot-med/config"
	"telegram-bot-med/internal/handlers"
	"telegram-bot-med/internal/repository"
	"telegram-bot-med/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Config loaded successfully")

	// Create application context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received, initiating graceful shutdown...")
		cancel()
	}()

	// Initialize application dependencies
	handler := initApplication(cfg)

	// Initialize Telegram Bot
	if err := initBot(cfg); err != nil {
		log.Printf("Warning: Failed to init Telegram Bot: %v", err)
	}

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/detect", handler.HandleDetect)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// initBot initializes the Telegram bot
func initBot(cfg *config.Config) error {
	if err := bot.Init(cfg.TelegramBotToken, cfg.AuthorizedChatID); err != nil {
		return err
	}

	// Set PocketBase URL and token for bot
	bot.SetPocketBaseURL(cfg.PocketBaseURL)
	bot.SetPocketBaseToken(cfg.PocketBaseToken)
	bot.StartPolling()

	log.Println("Telegram Bot Initialized")
	return nil
}

// initApplication initializes all application dependencies
func initApplication(cfg *config.Config) *handlers.DetectionHandler {
	// Initialize repositories with PocketBase REST API
	employeeRepo := repository.NewPocketBaseRESTEmployeeRepository(cfg.PocketBaseURL)
	attendanceRepo := repository.NewPocketBaseRESTAttendanceRepository(cfg.PocketBaseURL)
	detectionRepo := repository.NewPocketBaseRESTDetectionRepository(cfg.PocketBaseURL)
	scannerRepo := repository.NewPocketBaseRESTScannerRepository(cfg.PocketBaseURL)

	// Create bot notifier wrapper
	botNotifier := bot.NewNotifier()

	// Initialize services
	attendanceService := services.NewAttendanceService(
		employeeRepo,
		attendanceRepo,
		detectionRepo,
		scannerRepo,
		botNotifier,
	)

	// Initialize handlers
	detectionHandler := handlers.NewDetectionHandler(attendanceService)

	return detectionHandler
}
