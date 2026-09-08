package downloader

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxExtractedFileSize = 512 * 1024 * 1024
	maxDownloadSize      = 512 * 1024 * 1024
)

// DownloadMinecraftServer downloads and extracts the Minecraft Bedrock server.
// minecraftVer is the version of the server to download (e.g. "1.20.0.01").
// appDir is the directory where the server should be extracted.
// baseURL is an optional URL to download from (used for testing).
func DownloadMinecraftServer(minecraftVer string, appDir string, baseURL string) error {
	tmpFile, err := os.CreateTemp("", "bedrock-server-*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	zipPath, err := downloadServerArchive(tmpFile, minecraftVer, baseURL)
	if err != nil {
		return err
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err = os.MkdirAll(appDir, 0o750); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	return extractArchive(zipPath, appDir)
}

func downloadServerArchive(tmpFile *os.File, minecraftVer string, baseURL string) (string, error) {
	url := resolveDownloadURL(minecraftVer, baseURL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download server, status code: %d", resp.StatusCode)
	}

	if resp.ContentLength > maxDownloadSize {
		return "", fmt.Errorf("download exceeds maximum allowed size: %d bytes", resp.ContentLength)
	}

	limitedReader := io.LimitReader(resp.Body, maxDownloadSize+1)
	written, err := io.Copy(tmpFile, limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to save download: %w", err)
	}
	if written > maxDownloadSize {
		return "", fmt.Errorf("download exceeds maximum allowed size: %d bytes", written)
	}

	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to rewind temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

func resolveDownloadURL(minecraftVer string, baseURL string) string {
	if baseURL == "" {
		baseURL = "https://www.minecraft.net/bedrockdedicatedserver/bin-linux"
	}
	return fmt.Sprintf("%s/bedrock-server-%s.zip", baseURL, minecraftVer)
}

func extractArchive(archivePath string, appDir string) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if err = extractFile(file, appDir); err != nil {
			return fmt.Errorf("failed to extract file %s: %w", file.Name, err)
		}
	}

	return nil
}

func extractFile(file *zip.File, destDir string) error {
	cleanDestPath, err := sanitizeExtractPath(destDir, file.Name)
	if err != nil {
		return err
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(cleanDestPath, file.Mode())
	}

	if err = os.MkdirAll(filepath.Dir(cleanDestPath), 0o750); err != nil {
		return err
	}

	return copyZipEntry(file, cleanDestPath)
}

func sanitizeExtractPath(destDir string, entryName string) (string, error) {
	cleanDestDir := filepath.Clean(destDir)
	cleanDestPath := filepath.Clean(filepath.Join(cleanDestDir, entryName))
	if cleanDestPath == cleanDestDir || !strings.HasPrefix(cleanDestPath, cleanDestDir+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid zip entry path: %s", entryName)
	}
	return cleanDestPath, nil
}

func copyZipEntry(file *zip.File, destPath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return err
	}
	defer dest.Close()

	limitedReader := io.LimitReader(src, maxExtractedFileSize+1)
	written, err := io.Copy(dest, limitedReader)
	if err != nil {
		return err
	}
	if written > maxExtractedFileSize {
		return fmt.Errorf("extracted file exceeds maximum allowed size: %s", file.Name)
	}

	return nil
}
