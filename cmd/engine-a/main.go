// Engine A — deterministic simulator runtime with snapshot/delta WS streaming.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	enginerepo "github.com/sidsharma96/SysEscape/internal/engine/a/repo"
	"github.com/sidsharma96/SysEscape/internal/engine/a/transport"
	platformlog "github.com/sidsharma96/SysEscape/internal/platform/log"
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
)

func main() {
	logger := platformlog.NewLogger()

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}

	store, err := storage.NewS3BundleStore(storage.StorageConfig{
		Endpoint:       cfg.S3Endpoint,
		Bucket:         cfg.S3Bucket,
		AccessKey:      cfg.S3AccessKey,
		SecretKey:      cfg.S3SecretKey,
		Region:         cfg.S3Region,
		ForcePathStyle: cfg.S3ForcePathStyle,
	})
	if err != nil {
		logger.Error("failed to create bundle store", slog.String("error", err.Error()))
		os.Exit(1)
	}

	runtime := NewEngineARuntime(EngineARuntimeConfig{
		DB:          pool,
		RunRepo:     enginerepo.NewPostgresRunRepo(pool),
		BundleStore: store,
	})

	mux := http.NewServeMux()
	mux.Handle("/ws/engineA/", transport.NewWSHandler(transport.HandlerConfig{
		Secret:  cfg.RunTokenSecret,
		Runtime: runtime,
	}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("engine-a starting", slog.String("addr", srv.Addr))
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("engine-a stopped")
}
