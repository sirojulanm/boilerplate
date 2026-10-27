package helper

import "os"

// Env returns the value of key, or fallback when the variable is unset or empty.
func Env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
