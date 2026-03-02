// GraphQL BFF — control-plane API for auth, catalog, run lifecycle, and admin publishing.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/jackc/pgx/v5/pgxpool"
	authrepo "github.com/sidsharma96/SysEscape/internal/auth/repo"
	authservice "github.com/sidsharma96/SysEscape/internal/auth/service"
	authtransport "github.com/sidsharma96/SysEscape/internal/auth/transport"
	catalogrepo "github.com/sidsharma96/SysEscape/internal/catalog/repo"
	"github.com/sidsharma96/SysEscape/internal/graphql/generated"
	"github.com/sidsharma96/SysEscape/internal/graphql/resolvers"
	platformlog "github.com/sidsharma96/SysEscape/internal/platform/log"
	publishsvc "github.com/sidsharma96/SysEscape/internal/platform/publish"
	"github.com/sidsharma96/SysEscape/internal/platform/storage"
)

// Config holds GraphQL BFF runtime configuration loaded from env vars.
type Config struct {
	DatabaseURL          string
	GitHubClientID       string
	GitHubClientSecret   string
	GitHubRedirectURL    string
	PostLoginRedirectURL string
	AdminAPIKey          string
	S3Endpoint           string
	S3Bucket             string
	S3AccessKey          string
	S3SecretKey          string
	S3Region             string
	S3ForcePathStyle     bool
	Port                 string
}

// LoadConfigFromEnv loads and validates all required GraphQL BFF env vars.
func LoadConfigFromEnv() (Config, error) {
	s3ForcePathStyle, err := parseBoolEnv("S3_FORCE_PATH_STYLE", true)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		GitHubClientID:       os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret:   os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubRedirectURL:    os.Getenv("GITHUB_REDIRECT_URL"),
		PostLoginRedirectURL: os.Getenv("POST_LOGIN_REDIRECT_URL"),
		AdminAPIKey:          os.Getenv("ADMIN_API_KEY"),
		S3Endpoint:           envOrDefault("S3_ENDPOINT", "http://localhost:9000"),
		S3Bucket:             envOrDefault("S3_BUCKET", "ser-bundles"),
		S3AccessKey:          envOrDefault("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:          envOrDefault("S3_SECRET_KEY", "minioadmin"),
		S3Region:             envOrDefault("S3_REGION", "us-east-1"),
		S3ForcePathStyle:     s3ForcePathStyle,
		Port:                 os.Getenv("PORT"),
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	missing := make([]string, 0, 4)
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.GitHubClientID == "" {
		missing = append(missing, "GITHUB_CLIENT_ID")
	}
	if cfg.GitHubClientSecret == "" {
		missing = append(missing, "GITHUB_CLIENT_SECRET")
	}
	if cfg.GitHubRedirectURL == "" {
		missing = append(missing, "GITHUB_REDIRECT_URL")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("invalid %s value %q: %w", key, val, err)
	}
	return parsed, nil
}

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
		logger.Error("failed to connect to postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("failed to ping postgres", slog.String("error", err.Error()))
		os.Exit(1)
	}

	userRepo := authrepo.NewPostgresUserRepo(pool)
	sessionRepo := authrepo.NewPostgresSessionRepo(pool)
	roomRepo := catalogrepo.NewPostgresRoomRepo(pool)
	bundleStore, err := storage.NewS3BundleStore(storage.StorageConfig{
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
	publishService := publishsvc.NewPublishService(pool, bundleStore)

	oauthCfg := authservice.OAuthConfig{
		ClientID:             cfg.GitHubClientID,
		ClientSecret:         cfg.GitHubClientSecret,
		RedirectURL:          cfg.GitHubRedirectURL,
		PostLoginRedirectURL: cfg.PostLoginRedirectURL,
	}

	githubClient := authservice.NewRealGitHubClient(http.DefaultClient, oauthCfg)
	authSvc := authservice.NewAuthService(userRepo, sessionRepo, githubClient, oauthCfg)

	resolver := &resolvers.Resolver{
		CatalogRepo:    roomRepo,
		PublishService: publishService,
	}
	gqlSrv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	gqlHandler := authtransport.SessionMiddleware(authSvc)(gqlSrv)
	if cfg.AdminAPIKey != "" {
		gqlHandler = authtransport.AdminAPIKeyMiddleware(cfg.AdminAPIKey)(gqlHandler)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /auth/github/login", authtransport.HandleGitHubLogin(oauthCfg))
	mux.Handle("GET /auth/github/callback", authtransport.HandleGitHubCallback(authSvc, oauthCfg))
	mux.Handle("POST /auth/logout", authtransport.HandleLogout(authSvc))
	mux.Handle("POST /graphql", gqlHandler)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("graphql-bff starting", slog.String("addr", httpServer.Addr))
		errCh <- httpServer.ListenAndServe()
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

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("graphql-bff stopped")
}
