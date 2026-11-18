package config

import "testing"

func TestAuthConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, id, origins, android string
		enabled, invalid           bool
	}{
		{name: "disabled"},
		{name: "valid", id: "connect.test", origins: "https://connect.test", enabled: true},
		{name: "subdomain", id: "connect.test", origins: "https://app.connect.test:8443", enabled: true},
		{name: "missing origin", id: "connect.test", invalid: true},
		{name: "wildcard", id: "connect.test", origins: "https://*.connect.test", invalid: true},
		{name: "http", id: "connect.test", origins: "http://connect.test", invalid: true},
		{name: "foreign domain", id: "connect.test", origins: "https://evil.test", invalid: true},
		{name: "path", id: "connect.test", origins: "https://connect.test/", invalid: true},
		{name: "userinfo", id: "connect.test", origins: "https://user@connect.test", invalid: true},
		{name: "duplicates", id: "connect.test", origins: "https://connect.test,https://connect.test", invalid: true},
		{name: "bad android hash", id: "connect.test", origins: "https://connect.test", android: "android:apk-key-hash:invalid", invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadAuth(func(key string) string {
				switch key {
				case "AUTH_RP_ID":
					return tc.id
				case "AUTH_RP_ORIGINS":
					return tc.origins
				case "AUTH_ANDROID_ORIGINS":
					return tc.android
				}
				return ""
			})
			if (err != nil) != tc.invalid || cfg.Enabled != tc.enabled {
				t.Fatal("unexpected authentication configuration result")
			}
		})
	}
}
