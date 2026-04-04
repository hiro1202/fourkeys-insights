package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hiro1202/fourkeys-insights/internal/api"
	"github.com/hiro1202/fourkeys-insights/internal/config"
	"github.com/hiro1202/fourkeys-insights/internal/db"
	gh "github.com/hiro1202/fourkeys-insights/internal/github"
	"github.com/hiro1202/fourkeys-insights/internal/jobs"
	"github.com/hiro1202/fourkeys-insights/internal/static"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	logger, err := newLogger(cfg.Log.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger error: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Open DB
	store, err := db.NewSQLiteStore("./data/fourkeys.db")
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer store.Close()

	// Recover interrupted jobs from previous run
	if err := store.RecoverInterruptedJobs(context.Background()); err != nil {
		logger.Error("failed to recover interrupted jobs", zap.Error(err))
	}

	logger.Info("database initialized")

	// Setup GitHub client (optional at startup, required for sync)
	var ghClient *gh.Client
	if cfg.GitHub.Token != "" {
		ghClient, err = gh.NewClient(cfg.GitHub.Token, cfg.GitHub.APIBaseURL, logger)
		if err != nil {
			logger.Fatal("failed to create github client", zap.Error(err))
		}
		logger.Info("github client initialized")
	} else {
		logger.Warn("no github token configured, sync will not work until token is provided")
	}

	// Setup job queue
	var queue *jobs.Queue
	if ghClient != nil {
		queue = jobs.NewQueue(store, ghClient, logger)
	}

	// Setup router
	handler := &api.Handler{
		Store:  store,
		GitHub: ghClient,
		Queue:  queue,
		Logger: logger,
	}
	router := api.NewRouter(handler, logger)

	// Serve embedded frontend (SPA fallback)
	router.Handle("/*", static.Handler())

	addr := fmt.Sprintf("%s:%d", cfg.App.Bind, cfg.App.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-done
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("server stopped")
}

func newLogger(level string) (*zap.Logger, error) {
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapLevel),
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    zap.NewProductionEncoderConfig(),
	}
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}
