package fonts

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxArchiveSize = 128 << 20
	maxFontSize    = 32 << 20
)

type Manager struct {
	home   string
	termux bool
	client *http.Client
}

func NewManager(home string, termux bool) *Manager {
	return &Manager{
		home:   filepath.Clean(home),
		termux: termux,
		client: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (m *Manager) Install(ctx context.Context, font Font, progress func(downloaded, total int64)) error {
	if err := validateFont(font); err != nil {
		return err
	}
	if m.home == "" || m.home == "." {
		return fmt.Errorf("cannot determine HOME")
	}
	archive, err := m.download(ctx, font, progress)
	if err != nil {
		return err
	}
	fontData, err := extractFont(archive, font.ArchivePath)
	if err != nil {
		return err
	}
	if m.termux {
		return m.installTermux(ctx, fontData)
	}
	return m.installLinux(ctx, font, fontData)
}

func (m *Manager) RestoreTermux(ctx context.Context) error {
	if !m.termux {
		return fmt.Errorf("Termux font restore is only available in Termux mode")
	}
	destination := filepath.Join(m.home, ".termux", "font.ttf")
	backup := destination + ".ozsh.bak"
	data, err := os.ReadFile(backup)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no ozsh Termux font backup found")
		}
		return fmt.Errorf("read Termux font backup: %w", err)
	}
	if err := atomicWrite(destination, data, 0o600); err != nil {
		return fmt.Errorf("restore Termux font: %w", err)
	}
	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("remove restored font backup: %w", err)
	}
	return reloadTermux(ctx)
}

func validateFont(font Font) error {
	if strings.TrimSpace(font.ID) == "" || strings.TrimSpace(font.AssetURL) == "" || strings.TrimSpace(font.SHA256) == "" || strings.TrimSpace(font.ArchivePath) == "" {
		return fmt.Errorf("font manifest entry is incomplete")
	}
	parsed, err := url.Parse(font.AssetURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("invalid font asset URL")
	}
	if len(font.SHA256) != sha256.Size*2 {
		return fmt.Errorf("invalid font SHA-256")
	}
	if _, err := hex.DecodeString(font.SHA256); err != nil {
		return fmt.Errorf("invalid font SHA-256: %w", err)
	}
	clean := filepath.ToSlash(filepath.Clean(font.ArchivePath))
	if clean == "." || clean != filepath.ToSlash(font.ArchivePath) || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return fmt.Errorf("unsafe font archive path %q", font.ArchivePath)
	}
	return nil
}

func (m *Manager) download(ctx context.Context, font Font, progress func(downloaded, total int64)) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, font.AssetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create font request: %w", err)
	}
	response, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download font archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download font archive: HTTP %s", response.Status)
	}
	if response.ContentLength > maxArchiveSize {
		return nil, fmt.Errorf("font archive exceeds %d bytes", maxArchiveSize)
	}
	var buffer bytes.Buffer
	reader := io.LimitReader(response.Body, maxArchiveSize+1)
	chunk := make([]byte, 64*1024)
	var downloaded int64
	for {
		n, readErr := reader.Read(chunk)
		if n > 0 {
			downloaded += int64(n)
			if downloaded > maxArchiveSize {
				return nil, fmt.Errorf("font archive exceeds %d bytes", maxArchiveSize)
			}
			if _, err := buffer.Write(chunk[:n]); err != nil {
				return nil, fmt.Errorf("buffer font archive: %w", err)
			}
			if progress != nil {
				progress(downloaded, response.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read font archive: %w", readErr)
		}
	}
	archive := buffer.Bytes()
	sum := sha256.Sum256(archive)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), font.SHA256) {
		return nil, fmt.Errorf("font archive SHA-256 mismatch")
	}
	return archive, nil
}

func extractFont(archive []byte, archivePath string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("open font archive: %w", err)
	}
	for _, entry := range reader.File {
		if filepath.ToSlash(entry.Name) != filepath.ToSlash(archivePath) {
			continue
		}
		if entry.FileInfo().Mode()&os.ModeSymlink != 0 || !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("font archive asset must be a regular file")
		}
		if entry.UncompressedSize64 > maxFontSize {
			return nil, fmt.Errorf("font file exceeds %d bytes", maxFontSize)
		}
		handle, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open font asset: %w", err)
		}
		data, readErr := io.ReadAll(io.LimitReader(handle, maxFontSize+1))
		closeErr := handle.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read font asset: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close font asset: %w", closeErr)
		}
		if len(data) > maxFontSize {
			return nil, fmt.Errorf("font file exceeds %d bytes", maxFontSize)
		}
		return data, nil
	}
	return nil, fmt.Errorf("font asset %q not found in archive", archivePath)
}

func (m *Manager) installTermux(ctx context.Context, data []byte) error {
	dir := filepath.Join(m.home, ".termux")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create Termux config directory: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure Termux config directory: %w", err)
	}
	destination := filepath.Join(dir, "font.ttf")
	backup := destination + ".ozsh.bak"
	if _, err := os.Stat(backup); os.IsNotExist(err) {
		if current, readErr := os.ReadFile(destination); readErr == nil {
			if writeErr := atomicWrite(backup, current, 0o600); writeErr != nil {
				return fmt.Errorf("back up Termux font: %w", writeErr)
			}
		} else if !os.IsNotExist(readErr) {
			return fmt.Errorf("read existing Termux font: %w", readErr)
		}
	} else if err != nil {
		return fmt.Errorf("inspect Termux font backup: %w", err)
	}
	if err := atomicWrite(destination, data, 0o600); err != nil {
		return fmt.Errorf("install Termux font: %w", err)
	}
	return reloadTermux(ctx)
}

func (m *Manager) installLinux(ctx context.Context, font Font, data []byte) error {
	dir := filepath.Join(m.home, ".local", "share", "fonts", "ozsh")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create Linux font directory: %w", err)
	}
	name := filepath.Base(font.ArchivePath)
	if filepath.Ext(name) == "" {
		name += ".ttf"
	}
	if err := atomicWrite(filepath.Join(dir, name), data, 0o644); err != nil {
		return fmt.Errorf("install Linux font: %w", err)
	}
	if path, err := exec.LookPath("fc-cache"); err == nil {
		if output, runErr := exec.CommandContext(ctx, path, "-f", dir).CombinedOutput(); runErr != nil {
			return fmt.Errorf("refresh Linux font cache: %s", strings.TrimSpace(string(output)))
		}
	}
	return nil
}

func reloadTermux(ctx context.Context) error {
	path, err := exec.LookPath("termux-reload-settings")
	if err != nil {
		return nil
	}
	if output, runErr := exec.CommandContext(ctx, path).CombinedOutput(); runErr != nil {
		return fmt.Errorf("reload Termux settings: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".ozsh-font-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}
