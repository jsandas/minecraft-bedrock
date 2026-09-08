package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const envKeyParts = 2

// UpdateServerProperties reads environment variables prefixed with CFG_ and updates
// the server.properties file accordingly.
func UpdateServerProperties(appDir string) error {
	propsFile := filepath.Join(appDir, "server.properties")
	envVars := collectEnvironmentConfigVars()
	if len(envVars) == 0 {
		return nil
	}

	lines, err := readPropertiesFile(propsFile)
	if err != nil {
		return fmt.Errorf("error reading properties file: %w", err)
	}

	updatedLines, changed := updatePropertyLines(lines, envVars)
	if !changed {
		return nil
	}

	if writeErr := writePropertiesFile(propsFile, updatedLines); writeErr != nil {
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

	return envVars
}

func updatePropertyLines(lines []string, envVars map[string]string) ([]string, bool) {
	updatedLines := append([]string(nil), lines...)
	updated := false

	for i, line := range updatedLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", envKeyParts)
		if len(parts) != envKeyParts {
			continue
		}

		key := strings.TrimSpace(parts[0])
		newValue, exists := envVars[key]
		if !exists {
			continue
		}

		currentValue := strings.TrimSpace(parts[1])
		if currentValue == newValue {
			continue
		}

		updatedLines[i] = fmt.Sprintf("%s=%s", key, newValue)
		updated = true
		slog.Default().Info("Updating property", "key", key, "from", currentValue, "to", newValue)
	}

	return updatedLines, updated
}

func readPropertiesFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	var lines []string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return lines, nil
}

func writePropertiesFile(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err = writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}
