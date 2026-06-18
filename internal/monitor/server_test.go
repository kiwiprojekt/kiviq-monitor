package monitor

import "testing"

func TestRedactSensitiveQuery(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"ws creds", "/ws?user=admin&pass=secret", "/ws?user=***&pass=***"},
		{"bare query (no leading ?)", "user=admin&pass=secret", "user=***&pass=***"},
		{"reordered", "/ws?pass=s3cr3t&user=admin", "/ws?pass=***&user=***"},
		{"encoded value", "/ws?pass=a%20b&foo=bar", "/ws?pass=***&foo=bar"},
		{"no query", "/api/v1/agents", "/api/v1/agents"},
		// Must key on the param boundary, not a substring match.
		{"substrings untouched", "/x?bypass=1&surpass=2", "/x?bypass=1&surpass=2"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := redactSensitiveQuery(c.in); got != c.want {
				t.Errorf("redactSensitiveQuery(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
