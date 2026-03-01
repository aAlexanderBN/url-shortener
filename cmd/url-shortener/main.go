package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"url-shortener/internal/config"
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

	log.Info("config loaded", "env", cfg.Env, "storage_path", cfg.StoragePath, "http_server", fmt.Sprintf("%+v", cfg.HTTPServer))
	log.Debug("debug log")
	log.Error("err log")
	// TODO: init storage

	storage, err := sqlite.New(cfg.StoragePath)

	if err != nil {
		log.Error("failed to init storage", "error", err)
		os.Exit(1)
	}

	err = storage.DeleteURL("test")

	if err != nil {
		log.Error("failed to get url", "error", err)
	}

	// TODO: init rouder

	router := chi.NewRouter()

	//middleware

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/url", save.New(log, storage))
	router.Get("/{alias}", redirect.New(log, storage))
	router.Delete("/{alias}", delete.New(log, storage))

	log.Info("starting server", slog.String("address", cfg.Address))

	// TODO: run server
	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("failed to start server")
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
