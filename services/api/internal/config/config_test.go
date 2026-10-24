package config

import "testing"

func TestLoad(t *testing.T) {
	for _, tt := range []struct {
		name, value, want string
		invalid           bool
	}{
		{name: "default", want: "127.0.0.1:8080"},
		{name: "ipv6", value: "[::1]:8081", want: "[::1]:8081"},
		{name: "missing port", value: "localhost", invalid: true},
		{name: "zero port", value: "127.0.0.1:0", invalid: true},
		{name: "overflow", value: "127.0.0.1:65536", invalid: true},
		{name: "nonnumeric", value: "127.0.0.1:http", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := Load(func(string) string { return tt.value })
			if (err != nil) != tt.invalid {
				t.Fatalf("error = %v, want invalid %v", err, tt.invalid)
			}
			if !tt.invalid && cfg.ListenAddr != tt.want {
				t.Fatalf("address = %q, want %q", cfg.ListenAddr, tt.want)
			}
		})
	}
}
