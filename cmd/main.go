package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/conduitio/bwlimit"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"
	"github.com/hellodword/suno-radio/pkg/cloudflared"
	"github.com/hellodword/suno-radio/pkg/suno"
	"github.com/hellodword/suno-radio/pkg/types"
	"gopkg.in/yaml.v3"
)

const (
	UUIDLength = 36
)

func main() {
	var err error

	var loggerLevel = &slog.LevelVar{}
	var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggerLevel}))

	configPath := flag.String("config", "server.yml", "")
	flag.Parse()

	var conf = &types.ServerConfig{}

	{
		b, err := os.ReadFile(*configPath)
		if err != nil {
			panic(err)
		}

		err = yaml.Unmarshal(b, conf)
		if err != nil {
			panic(err)
		}

		err = loggerLevel.UnmarshalText([]byte(conf.LogLevel))
		if err != nil {
			panic(err)
		}

		if conf.DataDir == "" {
			conf.DataDir = "data"
		}

		logger.Info("load config", "config", *conf)
	}

	os.MkdirAll(conf.DataDir, 0755)

	// cache and reuse the *.trycloudflare.com
	// nginx is too heavy for this so ...
	cloudflared.Start(filepath.Join(conf.DataDir, "cloudflared.json"), logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var errC = make(chan error)

	pool := suno.NewWorkerPool(ctx, logger, time.Minute*30, conf.DataDir)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for _, playlist := range []string{suno.PlaylistTrending, suno.PlaylistNew} {
			err := pool.Add(playlist)
			if err != nil {
				logger.Error("pool add", "playlist", playlist, "err", err)
				if os.IsPermission(err) || os.IsNotExist(err) {
					errC <- err
					return
				}
			}
		}
	}()

	r := chi.NewRouter()
	r.Use(cors.AllowAll().Handler)

	// TODO embed a frontend player

	r.Route("/v1", func(r chi.Router) {
		r.Route("/playlist", func(r chi.Router) {
			r.Get("/", GetPlaylists(pool, logger))
			r.Get("/{id}", Radio(pool, logger))
			if conf.Auth != "" {
				r.With(Auth(conf.Auth)).Put("/{id}", AddPlaylist(pool, logger))
			}
		})
	})

	server := &http.Server{
		Addr:    conf.Addr,
		Handler: r,
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = server.ListenAndServe()
		if err != nil {
			logger.Error("ListenAndServe", "err", err)
			if !errors.Is(err, http.ErrServerClosed) {
				errC <- err
			}
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	select {
	case <-exit:
	case <-errC:
	}
	if server != nil {
		// server.Shutdown(ctx)
		server.Close()
	}
	cancel()
	wg.Wait()
	for _, id := range pool.IDs() {
		pool.Get(id).Close()
	}

}

func GetPlaylists(pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.DebugContext(r.Context(), "GetPlaylists")
		ids := pool.IDs()
		if err := render.Render(w, r, types.StringSlice(ids)); err != nil {
			logger.ErrorContext(r.Context(), "GetPlaylists", "err", err)
			_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusUnprocessableEntity, err))
			return
		}
	}
}

func Radio(pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		{
			// alias
			l := len(id)
			if l == len("new") && id == "new" {
				id = suno.PlaylistNew
			} else if l == len("trending") && id == "trending" {
				id = suno.PlaylistTrending
			}
		}

		if len(id) != UUIDLength {
			_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		found := pool.Contains(id)
		if !found {
			_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		logger.DebugContext(r.Context(), "Radio", "id", id)
		worker := pool.Get(id)

		w.Header().Set("Content-Type", "audio/mp3")

		// TODO find and explain a proper value
		responseWriter := bwlimit.NewWriter(w, 32*bwlimit.KB)

		if err := worker.Stream(r.Context(), responseWriter); err != nil {
			// logger.ErrorContext(r.Context(), "Radio", "id", id, "err", err)
			logger.DebugContext(r.Context(), "Radio", "id", id, "err", err)
			// _ = render.Render(w, r, types.ErrHTTPStatus(http.StatusInternalServerError, err))
			return
		}
	}
}

func AddPlaylist(pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if len(id) != UUIDLength {
			_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		found := pool.Contains(id)
		if !found {
			err := pool.Add(id)
			if err != nil {
				logger.ErrorContext(r.Context(), "AddPlaylist", "err", err)
				_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusInternalServerError, err))
				return
			}
		}

		ids := pool.IDs()
		if err := render.Render(w, r, types.StringSlice(ids)); err != nil {
			logger.ErrorContext(r.Context(), "AddPlaylist", "err", err)
			_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusUnprocessableEntity, err))
			return
		}
	}
}

func Auth(auth string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rauth := strings.TrimSpace(r.Header.Get("SUNO-RADIO-AUTH"))

			if subtle.ConstantTimeCompare([]byte(auth), []byte(rauth)) != 1 {
				_ = render.Render(w, r, types.ErrHTTPStatus(http.StatusUnauthorized, nil))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
