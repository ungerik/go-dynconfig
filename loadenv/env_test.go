package loadenv

import "testing"

func TestParseEnv_BasicPointerToStruct(t *testing.T) {
	t.Setenv("DYNCONFIG_TEST_PE_HOST", "example.com")
	t.Setenv("DYNCONFIG_TEST_PE_PORT", "9090")

	type config struct {
		Host string `env:"DYNCONFIG_TEST_PE_HOST"`
		Port int    `env:"DYNCONFIG_TEST_PE_PORT"`
	}
	cfg := &config{}
	if err := ParseEnv(cfg); err != nil {
		t.Fatalf("ParseEnv: %s", err)
	}
	if cfg.Host != "example.com" || cfg.Port != 9090 {
		t.Errorf("got %+v, want {Host:example.com Port:9090}", *cfg)
	}
}

// TestParseEnv_PointerToPointer covers the dereferencing logic unique to the
// default implementation: LoadEnvJSON[*T] / LoadEnvXML[*T] call ParseEnv(&config)
// where config is already a *T, so ParseEnv receives a **T. It must dereference
// one level so env.Parse gets the *T it requires.
func TestParseEnv_PointerToPointer(t *testing.T) {
	t.Setenv("DYNCONFIG_TEST_PE2_PORT", "1234")

	type config struct {
		Port int `env:"DYNCONFIG_TEST_PE2_PORT"`
	}
	cfg := &config{}
	pp := &cfg // **config
	if err := ParseEnv(pp); err != nil {
		t.Fatalf("ParseEnv: %s", err)
	}
	if cfg.Port != 1234 {
		t.Errorf("Port = %d, want 1234", cfg.Port)
	}
}

func TestParseEnv_DefaultValue(t *testing.T) {
	// The env var is intentionally never set, so the envDefault must apply.
	type config struct {
		Port int `env:"DYNCONFIG_TEST_PE_UNSET_PORT" envDefault:"8080"`
	}
	cfg := &config{}
	if err := ParseEnv(cfg); err != nil {
		t.Fatalf("ParseEnv: %s", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080 (default)", cfg.Port)
	}
}

func TestParseEnv_RequiredMissing(t *testing.T) {
	type config struct {
		Key string `env:"DYNCONFIG_TEST_PE_MISSING_REQUIRED,required"`
	}
	cfg := &config{}
	if err := ParseEnv(cfg); err == nil {
		t.Error("expected error for missing required env var")
	}
}

func TestParseEnv_ParseError(t *testing.T) {
	t.Setenv("DYNCONFIG_TEST_PE_BADINT", "not-a-number")

	type config struct {
		Port int `env:"DYNCONFIG_TEST_PE_BADINT"`
	}
	cfg := &config{}
	if err := ParseEnv(cfg); err == nil {
		t.Error("expected error for invalid int env value")
	}
}

// TestParseEnv_Customizable verifies ParseEnv is a replaceable var; restoring it
// afterwards keeps other tests using the default implementation.
func TestParseEnv_Customizable(t *testing.T) {
	original := ParseEnv
	t.Cleanup(func() { ParseEnv = original })

	called := false
	ParseEnv = func(dest any) error {
		called = true
		return nil
	}
	if err := ParseEnv(struct{}{}); err != nil {
		t.Fatalf("custom ParseEnv: %s", err)
	}
	if !called {
		t.Error("custom ParseEnv was not invoked")
	}
}
