package fonts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestReturnsIndependentCopy(t *testing.T) {
	first := Manifest()
	if len(first) != 3 || !first[0].Recommended {
		t.Fatalf("Manifest() = %#v", first)
	}
	first[0].Name = "changed"
	if Manifest()[0].Name == "changed" {
		t.Fatal("Manifest() returned aliased storage")
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	archive := testFontArchive(t, "Demo.ttf", []byte("font-data"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) }))
	defer server.Close()

	font := Font{ID: "demo", Name: "Demo", AssetURL: server.URL, SHA256: "00", ArchivePath: "Demo.ttf"}
	err := NewManager(t.TempDir(), true).Install(context.Background(), font, nil)
	if err == nil {
		t.Fatal("Install() checksum mismatch error = nil")
	}
}

func TestTermuxInstallBacksUpAndRestoresFont(t *testing.T) {
	home := t.TempDir()
	termuxDir := filepath.Join(home, ".termux")
	if err := os.MkdirAll(termuxDir, 0o700); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(termuxDir, "font.ttf")
	if err := os.WriteFile(destination, []byte("old-font"), 0o600); err != nil {
		t.Fatal(err)
	}
	archive := testFontArchive(t, "Demo.ttf", []byte("new-font"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) }))
	defer server.Close()

	font := testFont(server.URL, archive, "Demo.ttf")
	manager := NewManager(home, true)
	if err := manager.Install(context.Background(), font, nil); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	if got := readFile(t, destination); got != "new-font" {
		t.Fatalf("installed font = %q", got)
	}
	backup := destination + ".ozsh.bak"
	if got := readFile(t, backup); got != "old-font" {
		t.Fatalf("backup font = %q", got)
	}
	if err := manager.RestoreTermux(context.Background()); err != nil {
		t.Fatalf("RestoreTermux() error = %v", err)
	}
	if got := readFile(t, destination); got != "old-font" {
		t.Fatalf("restored font = %q", got)
	}
	if _, err := os.Stat(backup); !os.IsNotExist(err) {
		t.Fatalf("backup still present after restore: %v", err)
	}
}

func TestLinuxInstallUsesPrivateOzshFontDirectory(t *testing.T) {
	home := t.TempDir()
	archive := testFontArchive(t, "Demo.ttf", []byte("linux-font"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(archive) }))
	defer server.Close()
	font := testFont(server.URL, archive, "Demo.ttf")

	if err := NewManager(home, false).Install(context.Background(), font, nil); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
	destination := filepath.Join(home, ".local", "share", "fonts", "ozsh", "Demo.ttf")
	if got := readFile(t, destination); got != "linux-font" {
		t.Fatalf("installed Linux font = %q", got)
	}
}

func testFontArchive(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func testFont(url string, archive []byte, path string) Font {
	sum := sha256.Sum256(archive)
	return Font{ID: "demo", Name: "Demo", AssetURL: url, SHA256: hex.EncodeToString(sum[:]), ArchivePath: path}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(data)
}
