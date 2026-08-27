package config

import (
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "19009")
	t.Setenv("DATA_FILE", "/tmp/grid-test.json")
	t.Setenv("PERSIST", "false")
	t.Setenv("LONG_OUTAGE_MINUTES", "90")
	t.Setenv("SWEEP_INTERVAL", "5m")
	t.Setenv("REQUEST_BODY_LIMIT", "2048")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "19009" || cfg.Persist || cfg.LongOutageMinutes != 90 || cfg.SweepInterval != 5*time.Minute || cfg.RequestBodyLimit != 2048 || cfg.LogLevel != "debug" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.DataFile != "" {
		t.Fatalf("PERSIST=false should clear DataFile, got %q", cfg.DataFile)
	}
}

func TestValidateRejectsInvalid(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
	}{
		{"bad port", Config{Port: "abc"}},
		{"port zero", Config{Port: "0"}},
		{"missing data file", Config{Port: "8080", Persist: true}},
		{"negative threshold", Config{Port: "8080", Persist: true, DataFile: "x.json", LongOutageMinutes: 0}},
		{"bad log level", Config{Port: "8080", Persist: true, DataFile: "x.json", LongOutageMinutes: 1, SweepInterval: time.Minute, RequestBodyLimit: 1, LogLevel: "verbose"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.cfg.Validate(); err == nil {
				t.Fatalf("expected validation error for %+v", c.cfg)
			}
		})
	}
}
