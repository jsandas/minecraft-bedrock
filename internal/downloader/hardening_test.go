//nolint:testpackage // direct access to unexported extraction helpers is required for focused regression tests.
package downloader

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeExtractPath(t *testing.T) {
	t.Parallel()

	baseDir := filepath.Join(t.TempDir(), "app")

	t.Run("allows nested relative paths", func(t *testing.T) {
		t.Parallel()

		result, err := sanitizeExtractPath(baseDir, "nested/ok.txt")
		if err != nil {
			t.Fatalf("sanitizeExtractPath returned unexpected error: %v", err)
		}
		want := filepath.Join(baseDir, "nested", "ok.txt")
		if result != want {
			t.Fatalf("sanitizeExtractPath() = %q; want %q", result, want)
		}
	})

	t.Run("rejects traversal attempts", func(t *testing.T) {
		t.Parallel()

		for _, entry := range []string{"../escape.txt", "nested/../../escape.txt"} {
			if _, err := sanitizeExtractPath(baseDir, entry); err == nil {
				t.Fatalf("sanitizeExtractPath(%q) = nil; want error", entry)
			}
		}
	})

	t.Run("rejects absolute paths", func(t *testing.T) {
		t.Parallel()

		absolutePath := filepath.Join(string(os.PathSeparator), "tmp", "escape.txt")
		if _, err := sanitizeExtractPath(baseDir, absolutePath); err == nil {
			t.Fatal("sanitizeExtractPath accepted an absolute path")
		}
	})
}

func TestCopyZipEntryRejectsOversizeContent(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.zip")

	zipFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("os.Create() failed: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	writer, err := zipWriter.Create("big.txt")
	if err != nil {
		t.Fatalf("zipWriter.Create() failed: %v", err)
	}
	if _, err = writer.Write(bytes.Repeat([]byte("a"), maxExtractedFileSize+1)); err != nil {
		t.Fatalf("writer.Write() failed: %v", err)
	}
	if err = zipWriter.Close(); err != nil {
		t.Fatalf("zipWriter.Close() failed: %v", err)
	}
	if err = zipFile.Close(); err != nil {
		t.Fatalf("zipFile.Close() failed: %v", err)
	}

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("zip.OpenReader() failed: %v", err)
	}
	defer archive.Close()

	if len(archive.File) == 0 {
		t.Fatal("zip archive did not contain any files")
	}

	destPath := filepath.Join(tempDir, "extracted.txt")
	err = copyZipEntry(archive.File[0], destPath)
	if err == nil {
		t.Fatal("copyZipEntry() = nil; want size-limit error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
		t.Fatalf("copyZipEntry() error = %q; want size-limit error", err)
	}
}
