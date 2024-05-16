package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"flag"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/hellodword/suno-radio/frontend"
	"github.com/hellodword/suno-radio/internal/cloudflared"
	"github.com/hellodword/suno-radio/internal/common"
	"github.com/hellodword/suno-radio/internal/config"
	"github.com/hellodword/suno-radio/internal/httperr"
	"github.com/hellodword/suno-radio/internal/mp3toogg"
	"github.com/hellodword/suno-radio/internal/ogg"
	"github.com/hellodword/suno-radio/internal/suno"
	"github.com/jub0bs/cors"
)

func main() {
	var err error

	var loggerLevel = &slog.LevelVar{}
	var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: loggerLevel}))

	configPath := flag.String("config", "server.yml", "")
	flag.Parse()

	conf, err := config.LoadFromYaml(*configPath)
	if err != nil {
		panic(err)
	}

	err = loggerLevel.UnmarshalText([]byte(conf.LogLevel))
	if err != nil {
		panic(err)
	}

	os.MkdirAll(conf.DataDir, 0755)

	err = mp3toogg.MP3ToOggInit(conf.RPC)
	if err != nil {
		panic(err)
	}

	// cache and reuse the *.trycloudflare.com
	// nginx is too heavy for this so ...
	if *conf.Cloudflared {
		cloudflared.Start(filepath.Join(conf.DataDir, "cloudflared.json"), logger)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	var errC = make(chan error)

	pool := suno.NewWorkerPool(logger, time.Minute*30, conf.DataDir)

	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := range *conf.Playlist {
			logger.Info("pool adding", "playlist", (*conf.Playlist)[i])

			var id, alias string
			arr := strings.SplitN((*conf.Playlist)[i], "/", 2)
			alias = arr[0]

			if !validateAlias(alias) {
				logger.Error("invalid playlist alias", "playlist", (*conf.Playlist)[i])
				continue
			}

			if len(arr) == 2 {
				id = arr[1]
			} else {
				if _, ok := suno.PlayListPreset[alias]; ok {
					id = suno.PlayListPreset[alias]
				}
			}

			if len(id) != common.UUIDLength {
				logger.Error("invalid playlist id", "playlist", (*conf.Playlist)[i])
				continue
			}

			err := pool.Add(ctx, id, alias)
			if err != nil {
				logger.Error("pool add", "playlist", (*conf.Playlist)[i], "err", err)
				if os.IsPermission(err) || os.IsNotExist(err) {
					errC <- err
					return
				}
			}

			logger.Info("pool added", "playlist", (*conf.Playlist)[i], "id", id, "alias", alias)
		}
	}()

	corsMw, err := cors.NewMiddleware(cors.Config{
		Origins: []string{"*"},
		Methods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		RequestHeaders: []string{"*"},
		Credentialed:   false,
	})
	if err != nil {
		panic(err)
	}
	corsMw.SetDebug(false)

	r := chi.NewRouter()
	r.Use(corsMw.Wrap)

	dist, err := getDist()
	if err != nil {
		panic(err)
	}
	frontendServer := http.FileServer(dist)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		frontendServer.ServeHTTP(w, r)
	})

	r.Route("/v1", func(r chi.Router) {
		r.Route("/playlist", func(r chi.Router) {
			r.Get("/", GetPlaylists(pool, logger))
			r.Get("/{id}", Radio(pool, logger))
			if conf.Auth != "" {
				r.With(Auth(conf.Auth)).Put("/{id}/{alias}", AddPlaylist(ctx, pool, logger))
				r.With(Auth(conf.Auth)).Delete("/{id}", RemovePlaylist(pool, logger))
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
	case err = <-errC:
	}
	if server != nil {
		// server.Shutdown(ctx)
		server.Close()
	}
	cancel()
	wg.Wait()
	pool.Close()
}

func GetPlaylists(pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.DebugContext(r.Context(), "GetPlaylists")
		infos := pool.Infos()
		if err := render.Render(w, r, infos); err != nil {
			logger.ErrorContext(r.Context(), "GetPlaylists", "err", err)
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusUnprocessableEntity, err))
			return
		}
	}
}

func Radio(pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if !validateAlias(id) && !validateUUID(id) {
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		found := pool.Contains(id)
		if !found {
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		logger.DebugContext(r.Context(), "Radio", "id", id)
		worker := pool.Get(id)

		w.Header().Set("Content-Type", ogg.MIMEType)

		if err := worker.Stream(r.RemoteAddr, r.Context(), w); err != nil {
			// logger.ErrorContext(r.Context(), "Radio", "id", id, "err", err)
			logger.DebugContext(r.Context(), "Radio", "id", id, "err", err)
			// _ = render.Render(w, r, types.ErrHTTPStatus(http.StatusInternalServerError, err))
			return
		}
	}
}

func AddPlaylist(ctx context.Context, pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		alias := strings.ToLower(chi.URLParam(r, "alias"))

		if !validateAlias(alias) && !validateUUID(id) {
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		found := pool.Contains(id) || pool.Contains(alias)
		if !found {
			// the r.Context() wont work for this, pass the ctx from func main
			err := pool.Add(ctx, id, alias)
			if err != nil {
				logger.ErrorContext(r.Context(), "AddPlaylist", "err", err)
				_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusInternalServerError, err))
				return
			}
		}

		infos := pool.Infos()
		if err := render.Render(w, r, infos); err != nil {
			logger.ErrorContext(r.Context(), "AddPlaylist", "err", err)
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusUnprocessableEntity, err))
			return
		}
	}
}

func RemovePlaylist(pool *suno.WorkerPool, logger *slog.Logger) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if !validateAlias(id) && !validateUUID(id) {
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusBadRequest, nil))
			return
		}

		err := pool.Remove(id)
		if err != nil {
			logger.ErrorContext(r.Context(), "RemovePlaylist", "err", err)
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusInternalServerError, err))
			return
		}

		infos := pool.Infos()
		if err := render.Render(w, r, infos); err != nil {
			logger.ErrorContext(r.Context(), "RemovePlaylist", "err", err)
			_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusUnprocessableEntity, err))
			return
		}
	}
}

func Auth(auth string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rauth := strings.TrimSpace(r.Header.Get("SUNO-RADIO-AUTH"))

			if subtle.ConstantTimeCompare([]byte(auth), []byte(rauth)) != 1 {
				_ = render.Render(w, r, httperr.ErrHTTPStatus(http.StatusUnauthorized, nil))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getDist() (http.FileSystem, error) {
	fsys, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		return nil, err
	}
	return http.FS(fsys), nil
}

func validateUUID(s string) bool {
	if len(s) != 36 {
		return false
	}

	u, err := uuid.Parse(s)
	return err == nil && u.String() == s
}

func validateAlias(alias string) bool {
	if len(alias) < 3 || len(alias) > 32 {
		return false
	}

	for _, b := range []byte(alias) {
		if !(('a' <= b && b <= 'z') || ('0' <= b && b <= '9') || b == '_' || b == '-') {
			return false
		}
	}

	return true
}
