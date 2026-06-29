package dynconfig

import (
	"encoding/xml"
	"testing"
)

type xmlConfig struct {
	XMLName xml.Name `xml:"config"`
	Host    string   `xml:"host"`
	Port    int      `xml:"port"`
}

func TestLoadXML(t *testing.T) {
	file := memFile(t, "config.xml", `<config><host>example.com</host><port>8080</port></config>`)

	cfg, err := LoadXML[xmlConfig](file)
	if err != nil {
		t.Fatalf("LoadXML: %s", err)
	}
	if cfg.Host != "example.com" || cfg.Port != 8080 {
		t.Errorf("got %+v, want host=example.com port=8080", cfg)
	}
}

func TestLoadXML_Pointer(t *testing.T) {
	file := memFile(t, "config.xml", `<config><host>h</host><port>1</port></config>`)

	cfg, err := LoadXML[*xmlConfig](file)
	if err != nil {
		t.Fatalf("LoadXML: %s", err)
	}
	if cfg == nil {
		t.Fatal("got nil config")
	}
	if cfg.Host != "h" || cfg.Port != 1 {
		t.Errorf("got %+v", cfg)
	}
}

func TestLoadXML_Invalid(t *testing.T) {
	file := memFile(t, "config.xml", `<config><host>unclosed`)

	_, err := LoadXML[xmlConfig](file)
	if err == nil {
		t.Error("expected error for invalid XML")
	}
}

func TestLoadXML_FileNotExist(t *testing.T) {
	file := missingMemFile(t, "config.xml")

	_, err := LoadXML[xmlConfig](file)
	if err == nil {
		t.Error("expected error for missing file")
	}
}
