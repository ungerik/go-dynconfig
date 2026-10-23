package dynconfig

import (
	"context"
	"encoding/xml"
	"errors"
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

	err := SaveXML[xmlConfig]("  ")(file, xmlConfig{Host: "h", Port: 1})
	if err != nil {
		t.Fatalf("SaveXML: %s", err)
	}
	want := xml.Header + "<config>\n  <host>h</host>\n  <port>1</port>\n</config>"
	if got := readBack(t, file); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSaveXMLContext(t *testing.T) {
	cfg := xmlConfig{Host: "h", Port: 1}
	tests := []struct {
		name   string
		indent []string
		want   string
	}{
		{"no indent is compact", nil, xml.Header + "<config><host>h</host><port>1</port></config>"},
		{"empty indent is compact", []string{""}, xml.Header + "<config><host>h</host><port>1</port></config>"},
		{"indent strings are concatenated", []string{" ", " "}, xml.Header + "<config>\n  <host>h</host>\n  <port>1</port>\n</config>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := memFile(t, "config.xml", "old")

			err := SaveXMLContext[xmlConfig](t.Context(), tt.indent...)(file, cfg)
			if err != nil {
				t.Fatalf("SaveXMLContext: %s", err)
			}
			if got := readBack(t, file); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSaveXMLContext_Canceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	file := memFile(t, "config.xml", "old")

	err := SaveXMLContext[xmlConfig](ctx)(file, xmlConfig{Host: "h"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
	if got := readBack(t, file); got != "old" {
		t.Errorf("file changed to %q, want %q", got, "old")
	}
}

func TestLoadXMLContext(t *testing.T) {
	file := memFile(t, "config.xml", `<config><host>h</host><port>1</port></config>`)

	cfg, err := LoadXMLContext[xmlConfig](t.Context())(file)
	if err != nil {
		t.Fatalf("LoadXMLContext: %s", err)
	}
	if cfg.Host != "h" || cfg.Port != 1 {
		t.Errorf("got %+v, want host=h port=1", cfg)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = LoadXMLContext[xmlConfig](ctx)(file)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("canceled context: got error %v, want context.Canceled", err)
	}
}
