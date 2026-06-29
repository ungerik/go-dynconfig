package dynconfig

import "testing"

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
