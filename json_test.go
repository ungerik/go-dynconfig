package dynconfig

import (
	"context"
	"errors"
	"testing"
)

type jsonConfig struct {
	Host  string `json:"host"`
	Port  int    `json:"port"`
	Debug bool   `json:"debug"`
}

func TestLoadJSON(t *testing.T) {
	file := memFile(t, "config.json", `{"host":"example.com","port":8080,"debug":true}`)

	cfg, err := LoadJSON[jsonConfig](file)
	if err != nil {
		t.Fatalf("LoadJSON: %s", err)
	}
	want := jsonConfig{Host: "example.com", Port: 8080, Debug: true}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadJSON_Pointer(t *testing.T) {
	file := memFile(t, "config.json", `{"host":"h","port":1}`)

	cfg, err := LoadJSON[*jsonConfig](file)
	if err != nil {
		t.Fatalf("LoadJSON: %s", err)
	}
	if cfg == nil {
		t.Fatal("got nil config")
	}
	if cfg.Host != "h" || cfg.Port != 1 {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadJSON_Invalid(t *testing.T) {
	file := memFile(t, "config.json", `{not valid json`)

	_, err := LoadJSON[jsonConfig](file)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadJSON_FileNotExist(t *testing.T) {
	file := missingMemFile(t, "config.json")

	_, err := LoadJSON[jsonConfig](file)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSaveJSONContext(t *testing.T) {
	cfg := jsonConfig{Host: "h", Port: 1}
	tests := []struct {
		name   string
		indent []string
		want   string
	}{
		{"no indent is compact", nil, `{"host":"h","port":1,"debug":false}`},
		{"empty indent is compact", []string{""}, `{"host":"h","port":1,"debug":false}`},
		{"indent strings are concatenated", []string{" ", " "}, "{\n  \"host\": \"h\",\n  \"port\": 1,\n  \"debug\": false\n}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := memFile(t, "config.json", "old")

			err := SaveJSONContext[jsonConfig](t.Context(), tt.indent...)(file, cfg)
			if err != nil {
				t.Fatalf("SaveJSONContext: %s", err)
			}
			if got := readBack(t, file); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveJSONContext_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	file := memFile(t, "config.json", "old")

	err := SaveJSONContext[jsonConfig](ctx)(file, jsonConfig{Host: "h"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
	if got := readBack(t, file); got != "old" {
		t.Errorf("file changed to %q, want %q", got, "old")
	}
}

func TestLoadJSONContext(t *testing.T) {
	file := memFile(t, "config.json", `{"host":"h","port":1}`)

	cfg, err := LoadJSONContext[jsonConfig](t.Context())(file)
	if err != nil {
		t.Fatalf("LoadJSONContext: %s", err)
	}
	if want := (jsonConfig{Host: "h", Port: 1}); cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = LoadJSONContext[jsonConfig](ctx)(file)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("canceled context: got error %v, want context.Canceled", err)
	}
}
