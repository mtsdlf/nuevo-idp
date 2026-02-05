package config

import "os"

// Get returns the value of the environment variable key, or fallback if empty.
func Get(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

// Require returns the value of the environment variable key.
// If it is empty, it returns an error message string so callers can decide how to log/exit.
func Require(key string) (string, bool) {
	v := os.Getenv(key)
	if v == "" {
		return "", false
	}
	return v, true
}
