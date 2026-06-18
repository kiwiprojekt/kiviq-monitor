package monitor

import (
	"context"
	"crypto/subtle"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const agentIDContextKey contextKey = "agentID"

// AgentIDFromContext returns the agent ID that owns the token validated by
// PerAgentTokenAuth. This is the trusted identity of the reporting agent —
// callers must use it instead of any ID supplied in the request body.
func AgentIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(agentIDContextKey).(string)
	return id, ok
}

// Verifier checks monitor credentials against a stored bcrypt hash. bcrypt is
// deliberately slow, but the protected surface (admin login, WS-connect auth)
// is low-frequency and single-user, so a per-call compare is fine and avoids
// holding any password-equivalent in memory.
type Verifier struct {
	user     string
	passHash []byte
}

func NewVerifier(user, passHash string) *Verifier {
	return &Verifier{user: user, passHash: []byte(passHash)}
}

func (v *Verifier) Verify(user, pass string) bool {
	if subtle.ConstantTimeCompare([]byte(user), []byte(v.user)) != 1 {
		return false
	}
	return bcrypt.CompareHashAndPassword(v.passHash, []byte(pass)) == nil
}

func BasicAuth(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || !cfg.Verifier().Verify(u, p) {
				w.Header().Set("WWW-Authenticate", `Basic realm="kiviq"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func PerAgentTokenAuth(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			bearer := "Bearer "
			if len(auth) <= len(bearer) || auth[:len(bearer)] != bearer {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := auth[len(bearer):]

			// A single keyed lookup resolves the token to the agent that owns
			// it. That ID is carried forward in the request context and is the
			// only identity the report handler trusts — the request body cannot
			// claim to be a different agent.
			id, ok := cfg.AuthByToken(token)
			if !ok {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), agentIDContextKey, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func QueryAuth(cfg *Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := r.URL.Query().Get("user")
			p := r.URL.Query().Get("pass")
			if u == "" || p == "" || !cfg.Verifier().Verify(u, p) {
				w.Header().Set("WWW-Authenticate", `Basic realm="kiviq"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
