package helper_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirojulanam/boilerplate/internal/helper"
)

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		key := key
		value, existed := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(key, value)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func writeEnvFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	return path
}

func TestLoadEnvFileLoadsValues(t *testing.T) {
	keys := []string{"BOILERPLATE_ENV_PLAIN", "BOILERPLATE_ENV_EXPORTED", "BOILERPLATE_ENV_QUOTED", "BOILERPLATE_ENV_SINGLE", "BOILERPLATE_ENV_EQUALS"}
	unsetEnv(t, keys...)

	path := writeEnvFile(t, strings.Join([]string{
		"# comment",
		"",
		"BOILERPLATE_ENV_PLAIN=plain-value",
		"export BOILERPLATE_ENV_EXPORTED=exported-value",
		`BOILERPLATE_ENV_QUOTED="quoted value"`,
		"BOILERPLATE_ENV_SINGLE='single value'",
		"BOILERPLATE_ENV_EQUALS=left=right",
	}, "\n"))

	if err := helper.LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() error = %v", err)
	}

	want := map[string]string{
		"BOILERPLATE_ENV_PLAIN":    "plain-value",
		"BOILERPLATE_ENV_EXPORTED": "exported-value",
		"BOILERPLATE_ENV_QUOTED":   "quoted value",
		"BOILERPLATE_ENV_SINGLE":   "single value",
		"BOILERPLATE_ENV_EQUALS":   "left=right",
	}
	for key, value := range want {
		if got := os.Getenv(key); got != value {
			t.Fatalf("%s = %q, want %q", key, got, value)
		}
	}
}

func TestLoadEnvFileDoesNotOverrideExistingValues(t *testing.T) {
	unsetEnv(t, "BOILERPLATE_ENV_EXISTING")
	t.Setenv("BOILERPLATE_ENV_EXISTING", "from-process")

	path := writeEnvFile(t, "BOILERPLATE_ENV_EXISTING=from-file\n")
	if err := helper.LoadEnvFile(path); err != nil {
		t.Fatalf("LoadEnvFile() error = %v", err)
	}

	if got := os.Getenv("BOILERPLATE_ENV_EXISTING"); got != "from-process" {
		t.Fatalf("existing environment variable was overridden: %q", got)
	}
}

func TestLoadEnvFileIgnoresMissingFile(t *testing.T) {
	if err := helper.LoadEnvFile(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("LoadEnvFile() error = %v, want nil", err)
	}
}

func TestLoadEnvFileRejectsInvalidContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "missing separator", content: "NO_EQUALS", want: ":2:"},
		{name: "empty key", content: "=value", want: "empty variable name"},
		{name: "invalid quoted value", content: `BROKEN="bad\q"`, want: "invalid quoted value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeEnvFile(t, "\n"+test.content)
			err := helper.LoadEnvFile(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("LoadEnvFile() error = %v, want containing %q", err, test.want)
			}
		})
	}
}
