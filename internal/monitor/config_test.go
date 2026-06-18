package monitor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestLoadConfigMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")

	cfg := LoadConfig(path)

	if cfg.GetPort() != "9753" {
		t.Errorf("Port = %q, want default 9753", cfg.GetPort())
	}
	if len(cfg.GetAgents()) != 0 {
		t.Errorf("expected no agents, got %d", len(cfg.GetAgents()))
	}
	if cfg.path != path {
		t.Errorf("path = %q, want %q", cfg.path, path)
	}
}

func TestLoadConfigParsesAndSorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	contents := `{
		"port": "9090",
		"monitor_user": "admin",
		"monitor_password_hash": "hashed-secret",
		"agents": [
			{"id": "b", "name": "Beta", "order": 1},
			{"id": "a", "name": "Alpha", "order": 0}
		]
	}`
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := LoadConfig(path)

	if cfg.GetPort() != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.GetPort())
	}
	if cfg.MonitorUser != "admin" || cfg.MonitorPasswordHash != "hashed-secret" {
		t.Errorf("auth = %q/%q, want admin/hashed-secret", cfg.MonitorUser, cfg.MonitorPasswordHash)
	}

	agents := cfg.GetAgents()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].ID != "a" || agents[1].ID != "b" {
		t.Errorf("agents not sorted by order: %+v", agents)
	}
}

func TestGetPortDefault(t *testing.T) {
	cfg := &Config{}
	if cfg.GetPort() != "9753" {
		t.Errorf("Port = %q, want 9753", cfg.GetPort())
	}
}

func TestGetAgentsReturnsCopy(t *testing.T) {
	cfg := &Config{Agents: []AgentEntry{{ID: "a", Name: "Alpha"}}}

	got := cfg.GetAgents()
	got[0].Name = "mutated"

	if cfg.GetAgents()[0].Name != "Alpha" {
		t.Error("GetAgents returned a slice that aliases internal state")
	}
}

func TestSetAgentsPersistsAndSorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := LoadConfig(path)

	cfg.SetAgents([]AgentEntry{
		{ID: "b", Name: "Beta", Order: 1},
		{ID: "a", Name: "Alpha", Order: 0},
	})

	// In-memory state is sorted by order.
	agents := cfg.GetAgents()
	if agents[0].ID != "a" || agents[1].ID != "b" {
		t.Errorf("in-memory agents not sorted: %+v", agents)
	}

	// And the change survives a reload from disk.
	reloaded := LoadConfig(path)
	if len(reloaded.Agents) != 2 {
		t.Fatalf("expected 2 agents on reload, got %d", len(reloaded.Agents))
	}
	if reloaded.Agents[0].ID != "a" {
		t.Errorf("reloaded agents not sorted: %+v", reloaded.Agents)
	}
}

func TestAgentMaps(t *testing.T) {
	cfg := &Config{Agents: []AgentEntry{
		{ID: "a", Name: "Alpha", Token: "tok-a", Order: 0},
		{ID: "b", Name: "Beta", Order: 1}, // no token
	}}

	nameMap, orderMap := cfg.AgentMaps()

	if nameMap["a"] != "Alpha" || nameMap["b"] != "Beta" {
		t.Errorf("nameMap = %v", nameMap)
	}
	if orderMap["a"] != 0 || orderMap["b"] != 1 {
		t.Errorf("orderMap = %v", orderMap)
	}
}

func TestAuthByToken(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	cfg.SetAgents([]AgentEntry{
		{ID: "a", Name: "Alpha", Token: "tok-a"},
		{ID: "b", Name: "Beta"}, // no token
	})

	if id, ok := cfg.AuthByToken("tok-a"); !ok || id != "a" {
		t.Errorf("AuthByToken(tok-a) = %q,%v; want a,true", id, ok)
	}
	if _, ok := cfg.AuthByToken("nope"); ok {
		t.Error("unknown token should not match")
	}
	if _, ok := cfg.AuthByToken(""); ok {
		t.Error("empty token must never match (e.g. the token-less agent b)")
	}

	// The index tracks SetAgents changes.
	cfg.SetAgents([]AgentEntry{{ID: "c", Token: "tok-c"}})
	if _, ok := cfg.AuthByToken("tok-a"); ok {
		t.Error("removed agent's token should no longer match")
	}
	if id, ok := cfg.AuthByToken("tok-c"); !ok || id != "c" {
		t.Errorf("AuthByToken(tok-c) = %q,%v; want c,true", id, ok)
	}
}

func TestBootstrapFromEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := LoadConfig(path) // file missing -> empty hash

	t.Setenv("KIVIQ_MONITOR_USER", "root")
	t.Setenv("KIVIQ_MONITOR_PASSWORD", "s3cret")

	if err := cfg.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	if cfg.MonitorUser != "root" {
		t.Errorf("MonitorUser = %q, want root", cfg.MonitorUser)
	}
	if bcrypt.CompareHashAndPassword([]byte(cfg.MonitorPasswordHash), []byte("s3cret")) != nil {
		t.Error("stored hash does not verify against the bootstrap password")
	}

	// The hash is persisted so a reload needs no env.
	reloaded := LoadConfig(path)
	if reloaded.MonitorPasswordHash != cfg.MonitorPasswordHash {
		t.Error("bootstrapped hash was not persisted to disk")
	}
}

func TestBootstrapDefaultsUser(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("KIVIQ_MONITOR_PASSWORD", "pw")

	if err := cfg.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if cfg.MonitorUser != "admin" {
		t.Errorf("MonitorUser = %q, want default admin", cfg.MonitorUser)
	}
}

func TestBootstrapNoPasswordErrors(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("KIVIQ_MONITOR_PASSWORD", "")

	if err := cfg.Bootstrap(); err == nil {
		t.Fatal("expected error when no hash and no KIVIQ_MONITOR_PASSWORD")
	}
}

func TestBootstrapNoopWhenHashPresent(t *testing.T) {
	cfg := &Config{MonitorUser: "admin", MonitorPasswordHash: "existing"}
	t.Setenv("KIVIQ_MONITOR_PASSWORD", "ignored")

	if err := cfg.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	if cfg.MonitorPasswordHash != "existing" {
		t.Error("Bootstrap overwrote an existing hash")
	}
}

func TestSeedAgentFromEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := LoadConfig(path)

	t.Setenv("KIVIQ_SEED_AGENT_NAME", "Web Agent")
	t.Setenv("KIVIQ_SEED_AGENT_TOKEN", "tok-1")

	cfg.SeedAgent()

	agents := cfg.GetAgents()
	if len(agents) != 1 || agents[0].Token != "tok-1" || agents[0].Name != "Web Agent" {
		t.Fatalf("seeded agents = %+v", agents)
	}
	if agents[0].ID == "" {
		t.Fatal("seeded agent has no generated ID")
	}

	// The generated ID is persisted and stable across a reload.
	reloaded := LoadConfig(path)
	if len(reloaded.Agents) != 1 || reloaded.Agents[0].ID != agents[0].ID {
		t.Errorf("seeded ID not stable: reloaded %+v vs %+v", reloaded.Agents, agents)
	}
}

func TestSeedAgentGeneratesIDDefaultsName(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("KIVIQ_SEED_AGENT_TOKEN", "tok")

	cfg.SeedAgent()

	agents := cfg.GetAgents()
	if len(agents) != 1 || agents[0].Name != "default" {
		t.Fatalf("seeded agents = %+v, want name default", agents)
	}
	if agents[0].ID == "" || agents[0].ID == "default" {
		t.Errorf("expected a generated ID, got %q", agents[0].ID)
	}
}

func TestSeedAgentNoopWithoutToken(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("KIVIQ_SEED_AGENT_TOKEN", "")

	cfg.SeedAgent()

	if len(cfg.GetAgents()) != 0 {
		t.Error("seeded a agent without a token")
	}
}

func TestSetMonitorCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := LoadConfig(path)
	if err := cfg.SetMonitorCredentials("admin", "first-pass"); err != nil {
		t.Fatalf("SetMonitorCredentials: %v", err)
	}

	// Both fields change together and take effect for live auth.
	if err := cfg.SetMonitorCredentials("root", "second-pass"); err != nil {
		t.Fatalf("SetMonitorCredentials: %v", err)
	}
	if cfg.Verifier().Verify("admin", "first-pass") {
		t.Error("old credentials still accepted")
	}
	if !cfg.Verifier().Verify("root", "second-pass") {
		t.Error("new credentials not accepted")
	}

	// An empty field is left unchanged: change only the password.
	if err := cfg.SetMonitorCredentials("", "third-pass"); err != nil {
		t.Fatalf("password-only change: %v", err)
	}
	if !cfg.Verifier().Verify("root", "third-pass") {
		t.Error("username should be retained when only password changes")
	}

	// And the result is persisted.
	if reloaded := LoadConfig(path); !reloaded.Verifier().Verify("root", "third-pass") {
		t.Error("credentials not persisted across reload")
	}
}

// A password longer than bcrypt's 72-byte input limit must be rejected, and the
// existing credentials left untouched.
func TestSetMonitorCredentialsRejectsLongPassword(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	if err := cfg.SetMonitorCredentials("admin", "original"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := cfg.SetMonitorCredentials("", strings.Repeat("x", 73)); err == nil {
		t.Fatal("expected error for over-length password")
	}
	if !cfg.Verifier().Verify("admin", "original") {
		t.Error("original password no longer valid after a rejected change")
	}
}

// An invalid field must reject the whole update — the other field must not be
// applied, even partially.
func TestSetMonitorCredentialsAtomicOnInvalid(t *testing.T) {
	cfg := LoadConfig(filepath.Join(t.TempDir(), "config.json"))
	if err := cfg.SetMonitorCredentials("admin", "original"); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Valid password + invalid (blank) username.
	if err := cfg.SetMonitorCredentials("   ", "newsecret"); err == nil {
		t.Fatal("expected error for blank username")
	}
	if cfg.Verifier().Verify("admin", "newsecret") {
		t.Error("password was applied despite invalid username — not atomic")
	}
	if !cfg.Verifier().Verify("admin", "original") {
		t.Error("original credentials no longer valid after a rejected change")
	}
}

func TestSeedAgentNoopWhenAgentsExist(t *testing.T) {
	cfg := &Config{Agents: []AgentEntry{{ID: "a", Name: "A", Token: "existing"}}}
	t.Setenv("KIVIQ_SEED_AGENT_TOKEN", "tok")

	cfg.SeedAgent()

	agents := cfg.GetAgents()
	if len(agents) != 1 || agents[0].ID != "a" {
		t.Errorf("SeedAgent modified existing agents: %+v", agents)
	}
}
