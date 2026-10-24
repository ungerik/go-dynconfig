package dynconfig

import (
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

func TestSaveJSON(t *testing.T) {
	file := memFile(t, "config.json", "old")
	if err := SaveJSON[jsonConfig](file, jsonConfig{Host: "h", Port: 1}); err != nil {
		t.Fatalf("SaveJSON: %s", err)
	}
	if got, want := readBack(t, file), `{"host":"h","port":1,"debug":false}`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveJSONIndent(t *testing.T) {
	cfg := jsonConfig{Host: "h", Port: 1}
	tests := []struct {
		name   string
		indent string
		want   string
	}{
		{"empty indent is compact", "", `{"host":"h","port":1,"debug":false}`},
		{"two-space indent", "  ", "{\n  \"host\": \"h\",\n  \"port\": 1,\n  \"debug\": false\n}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := memFile(t, "config.json", "old")

			err := SaveJSONIndent[jsonConfig](tt.indent)(file, cfg)
			if err != nil {
				t.Fatalf("SaveJSONIndent: %s", err)
			}
			if got := readBack(t, file); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
