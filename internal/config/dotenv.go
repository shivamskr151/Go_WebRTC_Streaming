package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv loads environment variables from one or more .env files.
// Later files override earlier ones. Existing env vars are preserved.
func LoadDotEnv(paths ...string) {
	for _, path := range paths {
		loadSingle(path)
	}
}

func loadSingle(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Allow KEY=VALUE and export KEY=VALUE
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])

		// Remove optional surrounding quotes
		val = strings.Trim(val, " \t\"'")

		// Preserve existing env vars
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
