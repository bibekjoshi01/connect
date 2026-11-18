package main

import (
	"bytes"
	"log"
	"log/slog"
	"strings"
	"testing"
)

func TestTransportLogRedactsPeerAndError(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	log.New(transportLog{logger}, "", 0).Print("TLS handshake error from 192.0.2.1:1234 with credential=secret")
	text := output.String()
	if !strings.Contains(text, "http transport error") || strings.Contains(text, "192.0.2.1") || strings.Contains(text, "secret") {
		t.Fatal("transport log leaked untrusted connection details")
	}
}
