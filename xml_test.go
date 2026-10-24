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

func TestSaveXML(t *testing.T) {
	file := memFile(t, "config.xml", "old")
	loader := NewLoader(file, LoadXML[xmlConfig], SaveXML[xmlConfig], nil, nil, nil)

	err := loader.Set(xmlConfig{Host: "h", Port: 1})
	if err != nil {
		t.Fatalf("Set with SaveXML: %s", err)
	}
	want := xml.Header + "<config><host>h</host><port>1</port></config>"
	if got := readBack(t, file); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveXMLIndent(t *testing.T) {
	cfg := xmlConfig{Host: "h", Port: 1}
	tests := []struct {
		name   string
		indent string
		want   string
	}{
		{"empty indent is compact", "", xml.Header + "<config><host>h</host><port>1</port></config>"},
		{"two-space indent", "  ", xml.Header + "<config>\n  <host>h</host>\n  <port>1</port>\n</config>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := memFile(t, "config.xml", "old")

			err := SaveXMLIndent[xmlConfig](tt.indent)(file, cfg)
			if err != nil {
				t.Fatalf("SaveXMLIndent: %s", err)
			}
			if got := readBack(t, file); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
