package main

import (
	"log"

	"github.com/ungerik/go-dynconfig"
)

type Config struct {
	A string
	B bool
	C int
}

// Load config of type *Config from "config.json" and watch for changes.
// config.Get() will return the latest configuration of type *Config
// independent of any errors during loading.
var config = dynconfig.MustLoadAndWatch(
	"config.json",
	dynconfig.LoadEnvJSON[*Config],
	nil, // onLoad
	nil, // onError: nil means that errors will panic
	nil, // onInvalidate
)

// Use the callbacks to log what's happening
// and return a default configuration in case of an error.
var emailBlackist = dynconfig.MustLoadAndWatch(
	"email-blacklist.txt",
	dynconfig.LoadStringLineSetTrimSpace,
	// onLoad
	func(loaded map[string]struct{}) map[string]struct{} {
		log.Printf("Loaded email blacklist with %d addresses", len(loaded))
		return loaded
	},
	// onError
	func(err error) map[string]struct{} {
		log.Printf("Can't load email blacklist because: %s", err)
		return map[string]struct{}{"spam1@example.com": {}} // default in case of an error
	},
	// onInvalidate
	func() {
		log.Print("Invalidated email blacklist")
	},
)

func main() {
	// Get will always return the latest configuration
	// independent of any errors during loading
	log.Printf("Loaded config: %#v", config.Get())
	log.Printf("Loaded blacklist: %s", emailBlackist.Get())
}
