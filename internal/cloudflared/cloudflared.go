package cloudflared

import (
	"io"
	"log/slog"
	"net/http"
	"os"
)

var server *http.Server

func Start(p string, logger *slog.Logger) {
	logger.Info("cloudflared starting", "p", p)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/tunnel", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("cloudflared")
		b, _ := os.ReadFile(p)
		if len(b) > 0 {
			logger.Info("cloudflared", "hit", true)
			w.Header().Set("content-type", "application/json")
			w.Write(b)
			return
		}

		// https://github.com/cloudflare/cloudflared/blob/bb29a0e19437c3baa6a6e64f44b5de769206ed18/cmd/cloudflared/tunnel/quick_tunnel.go#L38-L47
		req, err := http.NewRequest(http.MethodPost, "https://api.trycloudflare.com/tunnel", nil)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Info("cloudflared", "req", err)
			return
		}
		req.Header = r.Header.Clone()
		req.Header.Del("Host")
		req.Header.Del("Accept-Encoding")
		req.Header.Set("Host", "api.trycloudflare.com")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Info("cloudflared", "res", err)
			return
		}

		defer res.Body.Close()
		b, err = io.ReadAll(res.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			logger.Info("cloudflared", "read", err)
			return
		}

		logger.Debug("cloudflared", "b", string(b))

		err = os.WriteFile(p, b, 0644)
		if err != nil {
			logger.Info("cloudflared", "write", err)
		}

		w.Header().Set("content-type", "application/json")
		w.Write(b)
	})
	server = &http.Server{
		Addr:    "0.0.0.0:7890",
		Handler: mux,
	}
	go func() {
		logger.Info("cloudflared listening")
		err := server.ListenAndServe()
		logger.Error("cloudflared stop", "err", err)
	}()
}

func Stop() {
	if server != nil {
		server.Close()
		server = nil
	}
}
