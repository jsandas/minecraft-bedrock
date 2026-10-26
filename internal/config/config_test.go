package config_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/jsandas/bedrock-server/internal/config"
)

//nolint:paralleltest // t.Setenv cannot be used with t.Parallel().
func TestUpdateServerProperties(t *testing.T) {
	tempDir := t.TempDir()
	propsFile := writePropertiesFile(t, tempDir, `# Minecraft server properties
server-name=Dedicated Server
gamemode=survival
difficulty=normal
allow-cheats=false
max-players=10
server-port=19132
server-portv6=19133
`)

	setEnvMultiple(t,
		"CFG_SERVER_NAME", "Test Server",
		"CFG_GAMEMODE", "creative",
		"CFG_MAX_PLAYERS", "20",
		"SOME_OTHER_VAR", "should-be-ignored",
	)

	logger := slog.New(slog.DiscardHandler)
	if err := configpkg.UpdateServerProperties(tempDir, logger); err != nil {
		t.Errorf("UpdateServerProperties failed: %v", err)
	}

	content, err := os.ReadFile(propsFile)
	if err != nil {
		t.Fatalf("Failed to read updated properties file: %v", err)
	}

	updatedContent := string(content)
	assertContainsAll(t, updatedContent,
		"server-name=Test Server",
		"gamemode=creative",
		"max-players=20",
	)
	assertContainsAll(t, updatedContent,
		"difficulty=normal",
		"allow-cheats=false",
		"server-port=19132",
		"server-portv6=19133",
		"server-udp-ports=19132:19132",
	)
}

func TestUpdateServerPropertiesNoChanges(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	propsFile := writePropertiesFile(t, tempDir, `# Minecraft server properties
server-name=Dedicated Server
gamemode=survival
server-port=19132
server-udp-ports=19132:19132
`)

	origInfo, err := os.Stat(propsFile)
	if err != nil {
		t.Fatalf("Failed to get original file info: %v", err)
	}

	logger := slog.New(slog.DiscardHandler)
	if updateErr := configpkg.UpdateServerProperties(tempDir, logger); updateErr != nil {
		t.Errorf("UpdateServerProperties failed: %v", updateErr)
	}

	// Get new file info
	newInfo, err := os.Stat(propsFile)
	if err != nil {
		t.Fatalf("Failed to get new file info: %v", err)
	}

	// Check that the file wasn't modified
	if newInfo.ModTime() != origInfo.ModTime() {
		t.Error("File was modified when it shouldn't have been")
	}
}

func TestUpdateServerPropertiesRedactsValues(t *testing.T) {
	tempDir := t.TempDir()
	writePropertiesFile(t, tempDir, "server-password=old-value\n")
	t.Setenv("CFG_SERVER_PASSWORD", "new-secret-value")

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	if err := configpkg.UpdateServerProperties(tempDir, logger); err != nil {
		t.Fatalf("UpdateServerProperties failed: %v", err)
	}

	logOutput := logBuf.String()
	if strings.Contains(logOutput, "old-value") || strings.Contains(logOutput, "new-secret-value") {
		t.Fatalf("log output leaked property values: %s", logOutput)
	}
	if !strings.Contains(logOutput, "key=server-password") {
		t.Fatalf("log output did not include the property key: %s", logOutput)
	}
}

func TestUpdateServerProperties_CustomPortEnv(t *testing.T) {
	tempDir := t.TempDir()
	propsFile := writePropertiesFile(t, tempDir, `server-name=Dedicated Server
server-port=19132
`)
	t.Setenv("PORT", "19140")

	logger := slog.New(slog.DiscardHandler)
	if err := configpkg.UpdateServerProperties(tempDir, logger); err != nil {
		t.Fatalf("UpdateServerProperties failed: %v", err)
	}

	content, err := os.ReadFile(propsFile)
	if err != nil {
		t.Fatalf("Failed to read updated properties file: %v", err)
	}

	updated := string(content)
	assertContainsAll(t, updated,
		"server-port=19140",
		"server-udp-ports=19140:19140",
	)
}

//nolint:paralleltest // t.Setenv cannot be used with t.Parallel().
func TestUpdateServerProperties_UncommentsServerUdpPorts(t *testing.T) {
	tempDir := t.TempDir()
	propsFile := writePropertiesFile(t, tempDir, `server-name=Dedicated Server
server-port=19132
# server-udp-ports=
#   server-udp-ports=49152-49200 (example comment)
`)

	logger := slog.New(slog.DiscardHandler)
	if err := configpkg.UpdateServerProperties(tempDir, logger); err != nil {
		t.Fatalf("UpdateServerProperties failed: %v", err)
	}

	content, err := os.ReadFile(propsFile)
	if err != nil {
		t.Fatalf("Failed to read updated properties file: %v", err)
	}

	updated := string(content)
	assertContainsAll(t, updated,
		"server-udp-ports=19132:19132",
		"#   server-udp-ports=49152-49200 (example comment)",
	)
	if strings.Contains(updated, "# server-udp-ports=") {
		t.Error("Placeholder # server-udp-ports= was not uncommented")
	}
}

func TestUpdateServerProperties_CustomUdpPortsOverridesDefault(t *testing.T) {
	tempDir := t.TempDir()
	propsFile := writePropertiesFile(t, tempDir, `server-port=19132
# server-udp-ports=
`)
	t.Setenv("PORT", "19135")
	t.Setenv("CFG_SERVER_UDP_PORTS", "19135:32000")

	logger := slog.New(slog.DiscardHandler)
	if err := configpkg.UpdateServerProperties(tempDir, logger); err != nil {
		t.Fatalf("UpdateServerProperties failed: %v", err)
	}

	content, err := os.ReadFile(propsFile)
	if err != nil {
		t.Fatalf("Failed to read updated properties file: %v", err)
	}

	updated := string(content)
	assertContainsAll(t, updated,
		"server-port=19135",
		"server-udp-ports=19135:32000",
	)
}

func setEnvMultiple(t *testing.T, pairs ...string) {
	t.Helper()
	for i := 0; i < len(pairs); i += 2 {
		key, value := pairs[i], pairs[i+1]
		t.Setenv(key, value)
	}
}

func writePropertiesFile(t *testing.T, dir string, content string) string {
	t.Helper()
	propsFile := filepath.Join(dir, "server.properties")
	if err := os.WriteFile(propsFile, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to create test properties file: %v", err)
	}
	return propsFile
}

func assertContainsAll(t *testing.T, content string, expected ...string) {
	t.Helper()
	for _, item := range expected {
		if !strings.Contains(content, item) {
			t.Errorf("Expected to find '%s' in properties file", item)
		}
	}
}
