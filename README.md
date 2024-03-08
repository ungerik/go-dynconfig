# go-dynconfig

Dynamic reload of a watched config file

Example:

```go
var emailBlackist = dynconfig.MustLoadAndWatch(
	"email-blacklist.txt",
	dynconfig.LoadStringLineSetTrimSpace[map[string]struct{}],
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
	log.Printf("Blacklisted: %s", emailBlackist.Get())
}
```