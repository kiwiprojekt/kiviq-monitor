package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/michal/kiviq/internal/shared"
)

// reportReq builds a /report request carrying the given authenticated agent ID
// in its context, mirroring what PerAgentTokenAuth injects.
func reportReq(agentID string, body []byte) *http.Request {
	req := httptest.NewRequest("POST", "/api/v1/report", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req.WithContext(context.WithValue(req.Context(), agentIDContextKey, agentID))
}

func setupTestHandlers(t *testing.T) (*Handlers, *Store, *Hub) {
	t.Helper()
	store := NewStore(0)
	hub := NewHub()
	go hub.Run()
	cfg := &Config{
		MonitorUser:         "admin",
		MonitorPasswordHash: "secret",
		Agents:              []AgentEntry{},
		path:                t.TempDir() + "/config.json",
	}
	handlers := NewHandlers(store, hub, cfg)
	return handlers, store, hub
}

func TestHandleReport(t *testing.T) {
	handlers, store, _ := setupTestHandlers(t)
	// The display name is derived from config by the authenticated ID, not from
	// the request body.
	handlers.cfg.Agents = []AgentEntry{{ID: "test-agent", Name: "test-agent", Token: "tok"}}

	reqBody := shared.ReportRequest{
		Hostname: "host-1",
		CPU:      shared.CPUInfo{Cores: 2, UsagePercent: 33.3},
		Memory:   shared.MemoryInfo{TotalBytes: 4096, UsagePercent: 50},
	}
	body, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	handlers.HandleReport(w, reportReq("test-agent", body))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp shared.ReportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}

	snap, ok := store.Get("test-agent")
	if !ok {
		t.Fatal("expected agent to be stored")
	}
	if snap.AgentName != "test-agent" {
		t.Errorf("AgentName = %q, want %q", snap.AgentName, "test-agent")
	}
}

// A report whose token has no resolved agent identity (no context value) is
// rejected — the agent cannot report without an authenticated identity.
func TestHandleReportNoIdentity(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)

	req := httptest.NewRequest("POST", "/api/v1/report", bytes.NewReader([]byte(`{}`)))
	w := httptest.NewRecorder()
	handlers.HandleReport(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHandleReportBadJSON(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)

	w := httptest.NewRecorder()
	handlers.HandleReport(w, reportReq("a1", []byte("not json")))

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleGetAgents(t *testing.T) {
	handlers, store, _ := setupTestHandlers(t)

	store.Update("a1", "s1", shared.ReportRequest{})
	store.Update("a2", "s2", shared.ReportRequest{})

	req := httptest.NewRequest("GET", "/api/v1/agents", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetAgents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var agents []shared.AgentSnapshot
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}
}

func TestHandleGetAgent(t *testing.T) {
	handlers, store, _ := setupTestHandlers(t)

	store.Update("a1", "s1", shared.ReportRequest{})

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", handlers.HandleGetAgent)

	req := httptest.NewRequest("GET", "/api/v1/agents/a1", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var snap shared.AgentSnapshot
	if err := json.NewDecoder(w.Body).Decode(&snap); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if snap.AgentName != "s1" {
		t.Errorf("AgentName = %q, want %q", snap.AgentName, "s1")
	}
}

func TestHandleGetAgentNotFound(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}", handlers.HandleGetAgent)

	req := httptest.NewRequest("GET", "/api/v1/agents/nonexistent", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetHistory(t *testing.T) {
	handlers, store, _ := setupTestHandlers(t)

	store.Update("a1", "a1", shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: 10}})
	time.Sleep(10 * time.Millisecond)
	store.Update("a1", "a1", shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: 20}})

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}/history", handlers.HandleGetHistory)

	req := httptest.NewRequest("GET", "/api/v1/agents/a1/history?window=1m", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp shared.HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if resp.AgentID != "a1" {
		t.Errorf("AgentID = %q, want %q", resp.AgentID, "a1")
	}
	if len(resp.Points) < 1 {
		t.Errorf("expected at least 1 point, got %d", len(resp.Points))
	}
}

func TestHandleGetHistoryNoWindowReturnsAll(t *testing.T) {
	handlers, store, _ := setupTestHandlers(t)

	store.Update("a1", "a1", shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: 10}})
	time.Sleep(10 * time.Millisecond)
	store.Update("a1", "a1", shared.ReportRequest{CPU: shared.CPUInfo{UsagePercent: 20}})

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{id}/history", handlers.HandleGetHistory)

	req := httptest.NewRequest("GET", "/api/v1/agents/a1/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp shared.HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(resp.Points) != 2 {
		t.Errorf("expected all 2 stored points, got %d", len(resp.Points))
	}
}

func TestHandleAdminGetAgents(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)
	handlers.cfg.SetAgents([]AgentEntry{
		{ID: "a", Name: "Alpha", Token: "tok-a"},
		{ID: "b", Name: "Beta", Token: "tok-b"},
	})

	req := httptest.NewRequest("GET", "/api/v1/admin/agents", nil)
	w := httptest.NewRecorder()

	handlers.HandleAdminGetAgents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var agents []AgentEntry
	if err := json.NewDecoder(w.Body).Decode(&agents); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].ID != "a" || agents[1].ID != "b" {
		t.Errorf("unexpected agent order: %+v", agents)
	}
}

func TestHandleAdminSetAgents(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)

	body, _ := json.Marshal([]AgentEntry{
		{ID: "a", Name: "Alpha", Token: "tok-a"},
		{ID: "b", Name: "Beta", Token: "tok-b"},
	})
	req := httptest.NewRequest("PUT", "/api/v1/admin/agents", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.HandleAdminSetAgents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Agents must be persisted with Order assigned by position.
	stored := handlers.cfg.GetAgents()
	if len(stored) != 2 {
		t.Fatalf("expected 2 stored agents, got %d", len(stored))
	}
	if stored[0].Order != 0 || stored[1].Order != 1 {
		t.Errorf("expected Order 0,1, got %d,%d", stored[0].Order, stored[1].Order)
	}

	// And actually written to disk.
	reloaded := LoadConfig(handlers.cfg.path)
	if len(reloaded.Agents) != 2 {
		t.Errorf("expected 2 agents on reload, got %d", len(reloaded.Agents))
	}
}

func TestHandleAdminSetAgentsValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"bad json", "not json"},
		{"empty id", `[{"id":"","name":"x"}]`},
		{"duplicate id", `[{"id":"a"},{"id":"a"}]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers, _, _ := setupTestHandlers(t)
			req := httptest.NewRequest("PUT", "/api/v1/admin/agents", bytes.NewReader([]byte(tt.body)))
			w := httptest.NewRecorder()

			handlers.HandleAdminSetAgents(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
			// Nothing should have been persisted.
			if got := handlers.cfg.GetAgents(); len(got) != 0 {
				t.Errorf("expected no agents persisted, got %d", len(got))
			}
		})
	}
}

func TestHandleAdminProvision(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)
	handlers.cfg.SetAgents([]AgentEntry{
		{ID: "srv-1", Name: "Agent One", Token: "secret-token"},
	})

	r := chi.NewRouter()
	r.Get("/api/v1/admin/provision/{id}", handlers.HandleAdminProvision)

	req := httptest.NewRequest("GET", "/api/v1/admin/provision/srv-1", nil)
	req.Host = "monitor.example.com"
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	// Identity (ID and name) is derived by the monitor from the token, so the
	// provisioning output carries only the token, connection host, and image —
	// not the agent ID or display name.
	for _, want := range []string{"secret-token", "monitor.example.com"} {
		if !strings.Contains(resp["install"], want) {
			t.Errorf("install command missing %q: %s", want, resp["install"])
		}
	}
	for _, unwanted := range []string{"srv-1", "Agent One"} {
		if strings.Contains(resp["install"], unwanted) {
			t.Errorf("install command should not leak identity %q: %s", unwanted, resp["install"])
		}
	}
	if resp["remove"] == "" {
		t.Error("expected a remove command")
	}
	for _, want := range []string{"secret-token", "monitor.example.com", "kiviq-agent-ca:"} {
		if !strings.Contains(resp["compose"], want) {
			t.Errorf("compose template missing %q: %s", want, resp["compose"])
		}
	}
}

func TestHandleAdminProvisionForwardedHost(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)
	handlers.cfg.SetAgents([]AgentEntry{{ID: "srv-1", Token: "t"}})

	r := chi.NewRouter()
	r.Get("/api/v1/admin/provision/{id}", handlers.HandleAdminProvision)

	req := httptest.NewRequest("GET", "/api/v1/admin/provision/srv-1", nil)
	req.Host = "internal:8080"
	req.Header.Set("X-Forwarded-Host", "public.example.com")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var resp map[string]string
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp["install"], "public.example.com") {
		t.Errorf("expected forwarded host in install command: %s", resp["install"])
	}
	if strings.Contains(resp["install"], "internal:8080") {
		t.Errorf("forwarded host should override r.Host: %s", resp["install"])
	}
}

func TestHandleAdminProvisionNotFound(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)

	r := chi.NewRouter()
	r.Get("/api/v1/admin/provision/{id}", handlers.HandleAdminProvision)

	req := httptest.NewRequest("GET", "/api/v1/admin/provision/missing", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleAdminChangeCredentials(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)
	cfg := handlers.cfg

	r := chi.NewRouter()
	r.Post("/api/v1/admin/password", handlers.HandleAdminChangePassword)

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/admin/password", strings.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// An empty payload changes nothing and is rejected.
	if w := post(`{}`); w.Code != http.StatusBadRequest {
		t.Errorf("empty change: got %d, want 400", w.Code)
	}

	// Username + password together both take effect for live auth.
	if w := post(`{"new_username":"root","new_password":"hunter2"}`); w.Code != http.StatusOK {
		t.Fatalf("change both: got %d, want 200", w.Code)
	}
	if !cfg.Verifier().Verify("root", "hunter2") {
		t.Error("new credentials not accepted after change")
	}

	// A username-only change retains the existing password.
	if w := post(`{"new_username":"operator"}`); w.Code != http.StatusOK {
		t.Fatalf("username-only change: got %d, want 200", w.Code)
	}
	if !cfg.Verifier().Verify("operator", "hunter2") {
		t.Error("password not retained after username-only change")
	}
}

// A change carrying a valid password alongside an invalid username must be
// rejected wholesale — the password must not be partially applied, or the
// operator gets an error response while their password has silently rotated.
func TestHandleAdminChangeCredentialsAtomic(t *testing.T) {
	handlers, _, _ := setupTestHandlers(t)
	cfg := handlers.cfg

	r := chi.NewRouter()
	r.Post("/api/v1/admin/password", handlers.HandleAdminChangePassword)

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/admin/password", strings.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	// Establish a known-good credential.
	if w := post(`{"new_username":"admin","new_password":"original"}`); w.Code != http.StatusOK {
		t.Fatalf("setup change: got %d, want 200", w.Code)
	}

	// Valid password + whitespace-only (invalid) username.
	w := post(`{"new_username":"   ","new_password":"newsecret"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("whitespace username: got %d, want 400", w.Code)
	}
	if cfg.Verifier().Verify("admin", "newsecret") {
		t.Error("password was partially applied despite a rejected request — not atomic")
	}
	if !cfg.Verifier().Verify("admin", "original") {
		t.Error("original password no longer valid after a rejected change")
	}
}

func TestHandleGetCA(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureCerts(dir); err != nil {
		t.Fatalf("EnsureCerts: %v", err)
	}

	caPEM, _ := os.ReadFile(filepath.Join(dir, "ca.pem"))
	handlers := &Handlers{cfg: &Config{path: filepath.Join(dir, "config.json")}}

	req := httptest.NewRequest("GET", "/api/v1/ca", nil)
	w := httptest.NewRecorder()

	handlers.HandleGetCA(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != string(caPEM) {
		t.Error("CA response does not match generated CA")
	}
}
