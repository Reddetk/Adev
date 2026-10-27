// Package env reads process configuration from the process environment and the
// .env file next to the working directory. Windows does not put .env values
// into the process environment by itself, so godotenv loads them once.
package env

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

var loadOnce sync.Once

// ModelKeyEnvName is the .env variable that holds the OpenRouter API key.
// The name "OUPENROUTER" is a typo the .env has lived with since it was written.
const ModelKeyEnvName = "OUPENROUTER_APIKEY"

func ensureLoaded() {
	loadOnce.Do(func() {
		// Missing file is fine (the variable may sit in the real environment),
		// existing environment variables are not overridden.
		_ = godotenv.Load()
	})
}

// ModelKey returns the OpenRouter API key from the environment or .env.
func ModelKey() (string, error) {
	ensureLoaded()
	key := os.Getenv(ModelKeyEnvName)
	if key == "" {
		return "", fmt.Errorf("%s is not set: put it in .env next to the working directory or in the environment", ModelKeyEnvName)
	}
	return key, nil
}
