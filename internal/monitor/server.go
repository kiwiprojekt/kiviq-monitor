package monitor

import (
	"bytes"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// sensitiveQueryRe matches the credential query params carried on the WebSocket
// connect URL. They must never reach the request log (or anyone reading it).
// The leading (^|[?&]) keys on a param boundary — start of a bare query string
// or a ?/& separator — so substrings like "bypass" are never matched.
var sensitiveQueryRe = regexp.MustCompile(`(^|[?&])((?:user|pass)=)[^&]*`)

// redactSensitiveQuery masks the value of any user/pass query param, whether
// given a full request URI ("/ws?user=...") or a bare query ("user=...").
func redactSensitiveQuery(uri string) string {
	return sensitiveQueryRe.ReplaceAllString(uri, "${1}${2}***")
}

// redactingLogFormatter delegates to chi's default request logger but feeds it a
// request whose URI has credentials masked, so the WS connect query (?user&pass)
// never lands in the logs.
type redactingLogFormatter struct {
	inner middleware.LogFormatter
}

func (f *redactingLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	if r.URL.RawQuery == "" {
		return f.inner.NewLogEntry(r)
	}
	r2 := r.Clone(r.Context())
	r2.RequestURI = redactSensitiveQuery(r2.RequestURI)
	r2.URL.RawQuery = redactSensitiveQuery(r2.URL.RawQuery)
	return f.inner.NewLogEntry(r2)
}

func NewServer(store *Store, hub *Hub, cfg *Config) http.Handler {
	handlers := NewHandlers(store, hub, cfg)

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(&redactingLogFormatter{
		inner: &middleware.DefaultLogFormatter{Logger: log.New(os.Stdout, "", log.LstdFlags)},
	}))
	r.Use(middleware.Recoverer)

	r.Group(func(r chi.Router) {
		r.Use(PerAgentTokenAuth(cfg))
		r.Post("/api/v1/report", handlers.HandleReport)
	})

	r.Group(func(r chi.Router) {
		r.Use(BasicAuth(cfg))
		r.Get("/api/v1/agents", handlers.HandleGetAgents)
		r.Get("/api/v1/agents/{id}", handlers.HandleGetAgent)
		r.Get("/api/v1/agents/{id}/history", handlers.HandleGetHistory)

		r.Get("/api/v1/admin/agents", handlers.HandleAdminGetAgents)
		r.Put("/api/v1/admin/agents", handlers.HandleAdminSetAgents)
		r.Get("/api/v1/admin/provision/{id}", handlers.HandleAdminProvision)
		r.Post("/api/v1/admin/password", handlers.HandleAdminChangePassword)
	})

	r.Group(func(r chi.Router) {
		r.Use(QueryAuth(cfg))
		r.Get("/ws", HandleWS(hub))
	})

	r.Get("/api/v1/ca", handlers.HandleGetCA)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	for _, webDir := range []string{filepath.Join(".", "web", "dist"), "/app/web/dist"} {
		if _, err := os.Stat(webDir); err == nil {
			r.Handle("/*", spaHandler{root: webDir, fileServer: http.FileServer(http.Dir(webDir))})
			break
		}
	}

	return r
}

type spaHandler struct {
	root       string
	fileServer http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.root, r.URL.Path)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		h.fileServer.ServeHTTP(w, r)
		return
	}
	h.serveIndex(w, r)
}

// serveIndex serves the SPA entry point. When the request comes through a proxy
// that mounts the app under a sub-path — Home Assistant ingress sends that
// prefix in X-Ingress-Path — it injects a <base> tag so the app's relative
// asset, API, and WebSocket URLs resolve under the prefix. Served directly (no
// such header) the file is returned unchanged.
func (h spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(h.root, "index.html")
	if prefix := r.Header.Get("X-Ingress-Path"); strings.HasPrefix(prefix, "/") {
		if data, err := os.ReadFile(indexPath); err == nil {
			base := []byte(`<head><base href="` + html.EscapeString(prefix) + `/">`)
			data = bytes.Replace(data, []byte("<head>"), base, 1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(data)
			return
		}
	}
	http.ServeFile(w, r, indexPath)
}
