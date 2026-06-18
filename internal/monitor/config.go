package monitor

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// newAgentID returns a random UUIDv4-formatted agent ID.
func newAgentID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		log.Fatalf("generating agent id: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

type AgentEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
	Order int    `json:"order"`
}

type Config struct {
	mu                  sync.RWMutex
	Port                string       `json:"port"`
	MonitorUser         string       `json:"monitor_user"`
	MonitorPasswordHash string       `json:"monitor_password_hash"`
	Agents              []AgentEntry `json:"agents"`
	path                string
	verifier            *Verifier
	tokens              map[string]string // token -> agent id; rebuilt whenever Agents changes
}

// rebuildTokenIndexLocked rebuilds the token->id index from c.Agents. Agents
// without a token are omitted. Caller must hold c.mu for writing.
func (c *Config) rebuildTokenIndexLocked() {
	idx := make(map[string]string, len(c.Agents))
	for _, a := range c.Agents {
		if a.Token != "" {
			idx[a.Token] = a.ID
		}
	}
	c.tokens = idx
}

func LoadConfig(path string) *Config {
	cfg := &Config{
		Port:   "9753",
		Agents: []AgentEntry{},
		path:   path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file %s not found, using defaults", path)
			return cfg
		}
		log.Fatalf("Failed to read config file %s: %v", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		log.Fatalf("Failed to parse config file %s: %v", path, err)
	}

	cfg.sortAgents()
	cfg.rebuildTokenIndexLocked()
	log.Printf("Loaded config from %s (%d agents)", path, len(cfg.Agents))
	return cfg
}

func (c *Config) sortAgents() {
	sort.Slice(c.Agents, func(i, j int) bool {
		return c.Agents[i].Order < c.Agents[j].Order
	})
}

// Verifier returns the credential verifier, building it on first use. The same
// instance is reused across requests so its in-memory password cache persists;
// SetMonitorCredentials swaps in a fresh one so a credential change takes effect
// for live auth without a restart.
func (c *Config) Verifier() *Verifier {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.verifier == nil {
		c.verifier = NewVerifier(c.MonitorUser, c.MonitorPasswordHash)
	}
	return c.verifier
}

// validateMonitorPassword reports whether newPass is an acceptable monitor
// password. 72 bytes is bcrypt's input limit. Exposed so callers can validate
// before applying, keeping a multi-field credential update atomic.
func validateMonitorPassword(newPass string) error {
	if newPass == "" {
		return fmt.Errorf("new password must not be empty")
	}
	if len(newPass) > 72 {
		return fmt.Errorf("new password must be at most 72 bytes")
	}
	return nil
}

// normalizeMonitorUsername trims newUser and reports an error if nothing
// remains. Returns the normalized username to apply.
func normalizeMonitorUsername(newUser string) (string, error) {
	newUser = strings.TrimSpace(newUser)
	if newUser == "" {
		return "", fmt.Errorf("new username must not be empty")
	}
	return newUser, nil
}

func hashMonitorPassword(newPass string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing monitor password: %w", err)
	}
	return string(hash), nil
}

// SetMonitorCredentials atomically updates the monitor username and/or password.
// An empty string leaves that field unchanged. Both inputs are validated and the
// password hashed before anything is applied, so the update is all-or-nothing and
// persists in a single write — a rejected field can never leave the other half
// applied. The verifier is rebuilt so the change is live immediately, without a
// restart. Returns an error only on invalid input.
func (c *Config) SetMonitorCredentials(newUser, newPass string) error {
	var newHash string
	if newPass != "" {
		if err := validateMonitorPassword(newPass); err != nil {
			return err
		}
		hash, err := hashMonitorPassword(newPass)
		if err != nil {
			return err
		}
		newHash = hash
	}
	if newUser != "" {
		u, err := normalizeMonitorUsername(newUser)
		if err != nil {
			return err
		}
		newUser = u
	}

	// Everything above is validated and cannot fail past this point, so the apply
	// is a single locked, all-or-nothing write.
	c.mu.Lock()
	defer c.mu.Unlock()
	if newUser != "" {
		c.MonitorUser = newUser
	}
	if newHash != "" {
		c.MonitorPasswordHash = newHash
	}
	c.verifier = NewVerifier(c.MonitorUser, c.MonitorPasswordHash)
	c.save()
	return nil
}

// Bootstrap ensures the config has monitor credentials. If a password hash is
// already present it is a no-op. Otherwise it reads KIVIQ_MONITOR_USER (default
// "admin") and KIVIQ_MONITOR_PASSWORD, hashes the password, and persists the
// config so the monitor can start on a fresh mount with no config file.
func (c *Config) Bootstrap() error {
	if c.MonitorPasswordHash != "" {
		return nil
	}

	user := os.Getenv("KIVIQ_MONITOR_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("KIVIQ_MONITOR_PASSWORD")
	if pass == "" {
		return fmt.Errorf("no monitor_password_hash in %s and KIVIQ_MONITOR_PASSWORD is not set; cannot bootstrap admin credentials", c.path)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing monitor password: %w", err)
	}

	c.mu.Lock()
	c.MonitorUser = user
	c.MonitorPasswordHash = string(hash)
	c.save()
	c.mu.Unlock()

	log.Printf("Bootstrapped admin credentials for user %q, wrote %s", user, c.path)
	return nil
}

// SeedAgent registers a single agent from the KIVIQ_SEED_AGENT_* env vars
// when no agents are configured yet, so a bundled agent can authenticate on
// first boot without manual registration via the Admin UI. It is a no-op if
// agents already exist or no seed token is provided; the monitor runs fine with
// no agents, so a missing seed is not an error.
func (c *Config) SeedAgent() {
	c.mu.Lock()
	defer c.mu.Unlock()

	token := os.Getenv("KIVIQ_SEED_AGENT_TOKEN")
	if len(c.Agents) > 0 || token == "" {
		return
	}

	// The ID is an opaque handle, generated once and persisted, so it is stable
	// across restarts without the operator having to choose one.
	id := newAgentID()
	name := os.Getenv("KIVIQ_SEED_AGENT_NAME")
	if name == "" {
		name = "default"
	}

	c.Agents = []AgentEntry{{ID: id, Name: name, Token: token, Order: 0}}
	c.rebuildTokenIndexLocked()
	c.save()
	log.Printf("Seeded agent %q (id %s) from KIVIQ_SEED_AGENT_* env, wrote %s", name, id, c.path)
}

func (c *Config) GetPort() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Port == "" {
		return "9753"
	}
	return c.Port
}

func (c *Config) GetAgents() []AgentEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]AgentEntry, len(c.Agents))
	copy(out, c.Agents)
	return out
}

func (c *Config) SetAgents(agents []AgentEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Agents = agents
	c.sortAgents()
	c.rebuildTokenIndexLocked()
	c.save()
}

// AgentName returns the configured display name for a agent ID, falling back
// to the ID itself when the agent has no name (or is unknown).
func (c *Config) AgentName(id string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, s := range c.Agents {
		if s.ID == id {
			if s.Name != "" {
				return s.Name
			}
			break
		}
	}
	return id
}

func (c *Config) save() {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}
	if err := os.WriteFile(c.path, data, 0644); err != nil {
		log.Printf("Failed to write config file %s: %v", c.path, err)
		return
	}
	log.Printf("Config saved to %s", c.path)
}

// AuthByToken resolves a presented bearer token to the agent ID that owns it.
// Tokens are high-entropy secrets, so a single keyed lookup is the right model —
// no scan over all agents. An empty token never matches.
func (c *Config) AuthByToken(token string) (string, bool) {
	if token == "" {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.tokens[token]
	return id, ok
}

// AgentMaps returns ID-keyed display-name and order maps for the configured
// agents, used to decorate and sort snapshots for the dashboard.
func (c *Config) AgentMaps() (nameMap map[string]string, orderMap map[string]int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	nameMap = make(map[string]string, len(c.Agents))
	orderMap = make(map[string]int, len(c.Agents))
	for _, s := range c.Agents {
		nameMap[s.ID] = s.Name
		orderMap[s.ID] = s.Order
	}
	return
}

func (c *Config) ConfigDir() string {
	return filepath.Dir(c.path)
}
