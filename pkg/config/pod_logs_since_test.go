package config

import "testing"

func TestNormalizePodLogsSince(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"  ", ""},
		{"24", "24h"},
		{"024", "24h"},
		{"1", "1h"},
		{"24h", "24h"},
		{"45m", "45m"},
		{"90s", "90s"},
		{"  12h  ", "12h"},
	}
	for _, tc := range cases {
		got, err := NormalizePodLogsSince(tc.in)
		if err != nil {
			t.Errorf("NormalizePodLogsSince(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizePodLogsSince(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizePodLogsSince_errors(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"0", "-1", "24x", "not-a-duration", "0s", "-5m"} {
		if _, err := NormalizePodLogsSince(in); err == nil {
			t.Fatalf("expected error for %q", in)
		}
	}
}
