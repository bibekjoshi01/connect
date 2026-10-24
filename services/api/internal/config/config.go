package config

import (
	"fmt"
	"net"
	"strconv"
)

// Config contains only settings used by the Stage 0 HTTP process.
type Config struct {
	ListenAddr string
}

func Load(getenv func(string) string) (Config, error) {
	addr := getenv("API_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return Config{}, fmt.Errorf("API_LISTEN_ADDR must be host:port: %w", err)
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("API_LISTEN_ADDR port must be between 1 and 65535")
	}
	return Config{ListenAddr: addr}, nil
}
