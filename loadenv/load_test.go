package loadenv

import (
	"encoding/xml"
	"testing"
)

type envJSONConfig struct {
	Host string `json:"host" env:"DYNCONFIG_TEST_JSON_HOST"`
	Port int    `json:"port" env:"DYNCONFIG_TEST_JSON_PORT"`
}

func TestLoadEnvJSON_EnvOverridesJSON(t *testing.T) {
	t.Setenv("DYNCONFIG_TEST_JSON_PORT", "9090")
	file := memFile(t, "config.json", `{"host":"from-json","port":8080}`)

	cfg, err := LoadEnvJSON[envJSONConfig](file)
	if err != nil {
		t.Fatalf("LoadEnvJSON: %s", err)
	}
	// Port overridden by env, host kept from JSON (its env var is unset).
	want := envJSONConfig{Host: "from-json", Port: 9090}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadEnvJSON_NoEnvKeepsJSON(t *testing.T) {
	file := memFile(t, "config.json", `{"host":"from-json","port":8080}`)

	cfg, err := LoadEnvJSON[envJSONConfig](file)
	if err != nil {
		t.Fatalf("LoadEnvJSON: %s", err)
	}
	want := envJSONConfig{Host: "from-json", Port: 8080}
	if cfg != want {
		t.Errorf("got %+v, want %+v", cfg, want)
	}
}

func TestLoadEnvJSON_Invalid(t *testing.T) {
	file := memFile(t, "config.json", `not json`)

	_, err := LoadEnvJSON[envJSONConfig](file)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

type envXMLConfig struct {
	XMLName xml.Name `xml:"config"`
	Host    string   `xml:"host" env:"DYNCONFIG_TEST_XML_HOST"`
	Port    int      `xml:"port" env:"DYNCONFIG_TEST_XML_PORT"`
}

func TestLoadEnvXML_EnvOverridesXML(t *testing.T) {
	t.Setenv("DYNCONFIG_TEST_XML_PORT", "9090")
	file := memFile(t, "config.xml", `<config><host>from-xml</host><port>8080</port></config>`)

	cfg, err := LoadEnvXML[envXMLConfig](file)
	if err != nil {
		t.Fatalf("LoadEnvXML: %s", err)
	}
	if cfg.Host != "from-xml" {
		t.Errorf("host = %q, want from-xml (kept from XML)", cfg.Host)
	}
	if cfg.Port != 9090 {
		t.Errorf("port = %d, want 9090 (overridden by env)", cfg.Port)
	}
}

func TestLoadEnvXML_NoEnvKeepsXML(t *testing.T) {
	file := memFile(t, "config.xml", `<config><host>from-xml</host><port>8080</port></config>`)

	cfg, err := LoadEnvXML[envXMLConfig](file)
	if err != nil {
		t.Fatalf("LoadEnvXML: %s", err)
	}
	if cfg.Host != "from-xml" || cfg.Port != 8080 {
		t.Errorf("got host=%q port=%d, want from-xml/8080", cfg.Host, cfg.Port)
	}
}

func TestLoadEnvXML_Invalid(t *testing.T) {
	file := memFile(t, "config.xml", `<config><port>not-a-number</port></config>`)

	_, err := LoadEnvXML[envXMLConfig](file)
	if err == nil {
		t.Error("expected error for invalid XML int value")
	}
}
