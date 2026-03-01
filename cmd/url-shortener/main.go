package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"url-shortener/internal/config"
	"url-shortener/internal/storage/postgres"
	"url-shortener/internal/storage/sqlite"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/http-server/middleware/logger/handlers/delete"
	"url-shortener/internal/http-server/middleware/logger/handlers/redirect"
	"url-shortener/internal/http-server/middleware/logger/handlers/url/save"
)

const (
	EnvLocal = "local"
	EnvDev   = "dev"
	EnvProd  = "prod"
)

func main() {

	// TODO: init config
	cfg := config.MustLoad()

	// TODO: init logger
	log := setupLogger(cfg.Env)

	log.Info("config loaded",
		"env", cfg.Env,
		"storage_type", cfg.StorageType,
		"storage_path", cfg.StoragePath,
		"http_server", fmt.Sprintf("%+v", cfg.HTTPServer),
	)
	log.Debug("debug log")
	// TODO: init storage
	storage, err := initStorage(cfg)

	if err != nil {
		log.Error("failed to init storage", "error", err)
		os.Exit(1)
	}
	defer storage.Close()

	// TODO: init rouder

	router := chi.NewRouter()

	//middleware

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"OK"}`))
	})

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password,
		}))

		r.Post("/", save.New(log, storage))
		r.Delete("/{alias}", delete.New(log, storage))

	})

	router.Get("/{alias}", redirect.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.HTTPServer.Address))

	// TODO: run server
	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("failed to start server", "error", err)
	}

	log.Error("server stopped")

}

func setupLogger(env string) *slog.Logger {

	var log *slog.Logger

	switch env {
	case EnvLocal:
		//log = setupPrettySlog()
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case EnvDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	case EnvProd:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	default:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	}

	return log
}

type URLStorage interface {
	SaveURL(alias, urlToSave string) (int64, error)
	GetURL(alias string) (string, error)
	DeleteURL(alias string) error
	Close() error
}

func initStorage(cfg *config.Config) (URLStorage, error) {
	switch strings.ToLower(cfg.StorageType) {
	case "sqlite":
		if cfg.StoragePath == "" {
			return nil, fmt.Errorf("storage_path is required for sqlite storage")
		}
		return sqlite.New(cfg.StoragePath)
	case "postgres":
		return postgres.New(cfg.Postgres.ConnString())
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.StorageType)
	}
}
