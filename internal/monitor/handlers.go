package monitor

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/michal/kiviq/internal/shared"
)

type Handlers struct {
	store *Store
	hub   *Hub
	cfg   *Config
}

func NewHandlers(store *Store, hub *Hub, cfg *Config) *Handlers {
	return &Handlers{store: store, hub: hub, cfg: cfg}
}

func (h *Handlers) HandleReport(w http.ResponseWriter, r *http.Request) {
	// Identity comes from the authenticated token, set by PerAgentTokenAuth —
	// not from anything in the request body.
	id, ok := AgentIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req shared.ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	snap, hp := h.store.Update(id, h.cfg.AgentName(id), req)

	h.hub.Broadcast(shared.WSMessage{
		Type:    "snapshot",
		Data:    snap,
		History: &hp,
	})

	resp := shared.ReportResponse{Status: "ok"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handlers) HandleGetAgents(w http.ResponseWriter, r *http.Request) {
	agents := h.store.GetAll()
	nameMap, orderMap := h.cfg.AgentMaps()

	for i := range agents {
		if name, ok := nameMap[agents[i].AgentID]; ok {
			agents[i].AgentName = name
		}
	}

	sortSnapshotsByOrder(agents, orderMap)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (h *Handlers) HandleGetAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	snap, ok := h.store.Get(id)
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

var historyWindows = map[string]time.Duration{
	"1m":  time.Minute,
	"5m":  5 * time.Minute,
	"15m": 15 * time.Minute,
	"1h":  time.Hour,
}

func (h *Handlers) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	window := r.URL.Query().Get("window")

	// A known window slices to that range; anything else (including no window)
	// returns the entire stored buffer so the client can load all and slice
	// client-side.
	since := time.Time{}
	if dur, ok := historyWindows[window]; ok {
		since = time.Now().Add(-dur)
	}

	points := h.store.GetHistory(id, since)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shared.HistoryResponse{AgentID: id, Points: points})
}

func (h *Handlers) HandleAdminGetAgents(w http.ResponseWriter, r *http.Request) {
	agents := h.cfg.GetAgents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (h *Handlers) HandleAdminSetAgents(w http.ResponseWriter, r *http.Request) {
	var agents []AgentEntry
	if err := json.NewDecoder(r.Body).Decode(&agents); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	seen := make(map[string]bool)
	for i := range agents {
		if agents[i].ID == "" {
			http.Error(w, "Agent ID is required", http.StatusBadRequest)
			return
		}
		if seen[agents[i].ID] {
			http.Error(w, "Duplicate agent ID", http.StatusBadRequest)
			return
		}
		seen[agents[i].ID] = true
		agents[i].Order = i
	}

	h.cfg.SetAgents(agents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleAdminChangePassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewUsername string `json:"new_username"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.NewUsername == "" && req.NewPassword == "" {
		http.Error(w, "Provide a new username or password", http.StatusBadRequest)
		return
	}

	// One atomic call validates both fields up front and applies them together,
	// so a rejected username can never leave the password silently rotated.
	if err := h.cfg.SetMonitorCredentials(req.NewUsername, req.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handlers) HandleAdminProvision(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	agents := h.cfg.GetAgents()
	var agent *AgentEntry
	for i := range agents {
		if agents[i].ID == id {
			agent = &agents[i]
			break
		}
	}
	if agent == nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	monitorHost := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		monitorHost = fwd
	}

	dep := newAgentDeployment(monitorHost, *agent)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"install": dep.runCommand(),
		"compose": dep.composeFile(),
		"remove":  dep.removeCommand(),
	})
}

func (h *Handlers) HandleGetCA(w http.ResponseWriter, r *http.Request) {
	configDir := h.cfg.ConfigDir()
	caPath := filepath.Join(configDir, "ca.pem")

	data, err := os.ReadFile(caPath)
	if err != nil {
		http.Error(w, "CA not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Write(data)
}

func sortSnapshotsByOrder(snapshots []shared.AgentSnapshot, orderMap map[string]int) {
	sort.Slice(snapshots, func(i, j int) bool {
		oi, okI := orderMap[snapshots[i].AgentID]
		oj, okJ := orderMap[snapshots[j].AgentID]
		if !okI {
			oi = 999999
		}
		if !okJ {
			oj = 999999
		}
		return oi < oj
	})
}
