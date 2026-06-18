package monitor

import (
	"fmt"
	"strings"
)

const (
	agentImage         = "ghcr.io/kiwiprojekt/kiviq-agent:latest"
	agentContainerName = "kiviq-agent"
	agentDataVolume    = "kiviq-agent-ca"
)

// agentDeployment is the single source of truth for how a provisioned agent
// container is configured. The `docker run` command and the compose file are
// both rendered from it, so the two representations can never drift apart.
type agentDeployment struct {
	env     [][2]string // ordered key/value pairs
	volumes []string    // "source:target[:opts]" mounts
}

func newAgentDeployment(monitorHost string, agent AgentEntry) agentDeployment {
	return agentDeployment{
		// The agent's identity (ID + name) is derived by the monitor from the
		// token, so only the token and connection settings are passed.
		env: [][2]string{
			{"MONITOR_URL", "https://" + monitorHost},
			{"AGENT_TOKEN", agent.Token},
			{"AGENT_CA_DIR", "/data"},
		},
		volumes: []string{
			agentDataVolume + ":/data",
			"/var/run/docker.sock:/var/run/docker.sock",
			"/etc/os-release:/host/etc/os-release:ro",
		},
	}
}

// runCommand renders a single-line `docker run` invocation.
func (d agentDeployment) runCommand() string {
	parts := []string{
		"docker run -d",
		"--name " + agentContainerName,
		"--network host",
		"--restart unless-stopped",
	}
	for _, kv := range d.env {
		parts = append(parts, fmt.Sprintf("-e %s=%q", kv[0], kv[1]))
	}
	for _, v := range d.volumes {
		parts = append(parts, "-v "+v)
	}
	parts = append(parts, agentImage)
	return strings.Join(parts, " ")
}

// composeFile renders an equivalent docker-compose.yml.
func (d agentDeployment) composeFile() string {
	var b strings.Builder
	b.WriteString("services:\n")
	fmt.Fprintf(&b, "  %s:\n", agentContainerName)
	fmt.Fprintf(&b, "    image: %s\n", agentImage)
	fmt.Fprintf(&b, "    container_name: %s\n", agentContainerName)
	b.WriteString("    restart: unless-stopped\n")
	b.WriteString("    network_mode: host\n")
	b.WriteString("    environment:\n")
	for _, kv := range d.env {
		fmt.Fprintf(&b, "      - %s=%s\n", kv[0], kv[1])
	}
	b.WriteString("    volumes:\n")
	for _, v := range d.volumes {
		fmt.Fprintf(&b, "      - %s\n", v)
	}
	fmt.Fprintf(&b, "\nvolumes:\n  %s:\n", agentDataVolume)
	return b.String()
}

// removeCommand stops and removes the provisioned agent container.
func (d agentDeployment) removeCommand() string {
	return fmt.Sprintf("docker stop %s && docker rm %s", agentContainerName, agentContainerName)
}
