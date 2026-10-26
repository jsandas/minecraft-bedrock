package config

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	DefaultPort = "19132"
	envKeyParts = 2
)

// UpdateServerProperties reads environment variables prefixed with CFG_ and updates
// the server.properties file accordingly.
func UpdateServerProperties(appDir string, logger *slog.Logger) error {
	root, err := os.OpenRoot(filepath.Clean(appDir))
	if err != nil {
		return fmt.Errorf("error opening app root: %w", err)
	}
	defer root.Close()

	envVars := collectEnvironmentConfigVars()

	lines, err := readPropertiesFile(root, "server.properties")
	if err != nil {
		return fmt.Errorf("error reading properties file: %w", err)
	}

	updatedLines, changed := updatePropertyLines(lines, envVars, logger)
	if !changed {
		return nil
	}

	if writeErr := writePropertiesFile(root, "server.properties", updatedLines); writeErr != nil {
		return fmt.Errorf("error writing properties file: %w", writeErr)
	}

	return nil
}

func collectEnvironmentConfigVars() map[string]string {
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "CFG_") {
			continue
		}

		parts := strings.SplitN(env, "=", envKeyParts)
		if len(parts) != envKeyParts {
			continue
		}

		key := strings.ToLower(strings.ReplaceAll(strings.TrimPrefix(parts[0], "CFG_"), "_", "-"))
		value := parts[1]
		envVars[key] = value
	}

	port := DefaultPort
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}
	if p, ok := envVars["server-port"]; ok && p != "" {
		port = p
	}

	if _, ok := envVars["server-port"]; !ok {
		envVars["server-port"] = port
	}

	if _, ok := envVars["server-udp-ports"]; !ok {
		envVars["server-udp-ports"] = fmt.Sprintf("%s:%s", port, port)
	}

	return envVars
}

func updatePropertyLines(lines []string, envVars map[string]string, logger *slog.Logger) ([]string, bool) {
	updatedLines := append([]string(nil), lines...)
	handled := make(map[string]bool)
	updated := updateActiveProperties(updatedLines, envVars, handled, logger)

	if uncommentTemplateProperties(updatedLines, envVars, handled, logger) {
		updated = true
	}
	if appendMissingProperties(&updatedLines, envVars, handled, logger) {
		updated = true
	}

	return updatedLines, updated
}

func updateActiveProperties(
	lines []string,
	envVars map[string]string,
	handled map[string]bool,
	logger *slog.Logger,
) bool {
	updated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		key, currentValue, ok := parsePropertyLine(trimmed)
		if !ok {
			continue
		}

		newValue, exists := envVars[key]
		if !exists {
			continue
		}

		handled[key] = true
		if currentValue == newValue {
			continue
		}

		lines[i] = fmt.Sprintf("%s=%s", key, newValue)
		updated = true
		logPropertyUpdate(logger, key)
	}
	return updated
}

func uncommentTemplateProperties(
	lines []string,
	envVars map[string]string,
	handled map[string]bool,
	logger *slog.Logger,
) bool {
	updated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}

		withoutHash := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		key, _, ok := parsePropertyLine(withoutHash)
		if !ok || handled[key] {
			continue
		}

		newValue, exists := envVars[key]
		if !exists {
			continue
		}

		lines[i] = fmt.Sprintf("%s=%s", key, newValue)
		handled[key] = true
		updated = true
		logPropertyUpdate(logger, key)
	}
	return updated
}

func appendMissingProperties(
	lines *[]string,
	envVars map[string]string,
	handled map[string]bool,
	logger *slog.Logger,
) bool {
	var remainingKeys []string
	for key := range envVars {
		if !handled[key] {
			remainingKeys = append(remainingKeys, key)
		}
	}
	if len(remainingKeys) == 0 {
		return false
	}

	slices.Sort(remainingKeys)
	for _, key := range remainingKeys {
		*lines = append(*lines, fmt.Sprintf("%s=%s", key, envVars[key]))
		handled[key] = true
		logPropertyUpdate(logger, key)
	}
	return true
}

func parsePropertyLine(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", envKeyParts)
	if len(parts) != envKeyParts {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func logPropertyUpdate(logger *slog.Logger, key string) {
	if logger == nil {
		return
	}

	// Property values are intentionally never logged. Only the property key is
	// emitted to avoid leaking secrets such as passwords or tokens.
	logger.Info("Updating property", "key", key)
}

func readPropertiesFile(root *os.Root, filePath string) ([]string, error) {
	file, err := root.Open(filePath)
	if err != nil {
		return nil, err
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if scanErr := scanner.Err(); scanErr != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return nil, fmt.Errorf("scan error: %w", errors.Join(scanErr, closeErr))
		}
		return nil, scanErr
	}

	if closeErr := file.Close(); closeErr != nil {
		return nil, closeErr
	}

	return lines, nil
}

func writePropertiesFile(root *os.Root, filePath string, lines []string) error {
	file, err := root.Create(filePath)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err = writer.WriteString(line + "\n"); err != nil {
			closeErr := file.Close()
			if closeErr != nil {
				return fmt.Errorf("write error: %w", errors.Join(err, closeErr))
			}
			return err
		}
	}

	if err = writer.Flush(); err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("flush error: %w", errors.Join(err, closeErr))
		}
		return err
	}

	if closeErr := file.Close(); closeErr != nil {
		return closeErr
	}

	return nil
}
