package browser

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"notion-getpage/internal/core"
)

const versionsURL = "https://googlechromelabs.github.io/chrome-for-testing/last-known-good-versions-with-downloads.json"
const chromeCheckInterval = 24 * time.Hour

var chromeVersionPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]+){3}$`)

type chromeAsset struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

type chromeVersions struct {
	Channels map[string]struct {
		Version   string                   `json:"version"`
		Downloads map[string][]chromeAsset `json:"downloads"`
	} `json:"channels"`
}

func managedChrome() (string, error) {
	platform, executable, err := chromePlatform()
	if err != nil {
		return "", err
	}
	base, err := core.BrowserDir()
	if err != nil {
		return "", err
	}
	if err := core.EnsurePrivateDir(base); err != nil {
		return "", err
	}
	var existing string
	currentPath := filepath.Join(base, "current")
	if saved, err := os.ReadFile(currentPath); err == nil {
		version := strings.TrimSpace(string(saved))
		if chromeVersionPattern.MatchString(version) {
			path := filepath.Join(base, version, executable)
			if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
				existing = path
				if checked, err := os.Stat(currentPath); err == nil && time.Since(checked.ModTime()) < chromeCheckInterval {
					return existing, nil
				}
			}
		}
	}
	updated, err := updateManagedChrome(base, platform, executable)
	if err != nil && existing != "" {
		fmt.Fprintln(os.Stderr, "Aviso: não foi possível verificar ou atualizar o Chrome; usando a versão instalada:", err)
		return existing, nil
	}
	return updated, err
}

func updateManagedChrome(base, platform, executable string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(versionsURL)
	if err != nil {
		return "", fmt.Errorf("consultar Chrome for Testing: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("consultar Chrome for Testing: HTTP %d", resp.StatusCode)
	}
	var versions chromeVersions
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&versions); err != nil {
		return "", err
	}
	stable := versions.Channels["Stable"]
	var assetURL string
	for _, asset := range stable.Downloads["chrome"] {
		if asset.Platform == platform {
			assetURL = asset.URL
			break
		}
	}
	if !chromeVersionPattern.MatchString(stable.Version) || assetURL == "" || !strings.HasPrefix(assetURL, "https://storage.googleapis.com/chrome-for-testing-public/") {
		return "", errors.New("Chrome for Testing não forneceu um download para esta plataforma")
	}
	versionDir := filepath.Join(base, stable.Version)
	bin := filepath.Join(versionDir, executable)
	if _, err := os.Stat(bin); err == nil {
		_ = os.WriteFile(filepath.Join(base, "current"), []byte(stable.Version), 0600)
		return bin, nil
	}
	zipPath := filepath.Join(base, stable.Version+".zip")
	defer os.Remove(zipPath)
	fmt.Fprintln(os.Stderr, "Baixando Chrome for Testing para o primeiro uso...")
	file, err := os.OpenFile(zipPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Get(assetURL)
	if err != nil {
		file.Close()
		return "", fmt.Errorf("baixar Chrome for Testing: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		file.Close()
		return "", fmt.Errorf("baixar Chrome for Testing: HTTP %d", response.StatusCode)
	}
	_, copyErr := io.Copy(file, io.LimitReader(response.Body, 500<<20))
	response.Body.Close()
	closeErr := file.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if err := extractChromeZip(zipPath, versionDir); err != nil {
		return "", err
	}
	if err := os.Chmod(bin, 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(base, "current"), []byte(stable.Version), 0600); err != nil {
		return "", err
	}
	return bin, nil
}

func chromePlatform() (platform, executable string, err error) {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "linux64", "chrome-linux64/chrome", nil
	case "darwin/amd64":
		return "mac-x64", "chrome-mac-x64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing", nil
	case "darwin/arm64":
		return "mac-arm64", "chrome-mac-arm64/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing", nil
	case "windows/amd64":
		return "win64", "chrome-win64/chrome.exe", nil
	default:
		return "", "", fmt.Errorf("plataforma sem navegador gerenciado: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
}

func extractChromeZip(zipPath, destination string) error {
	archive, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer archive.Close()
	if err := core.EnsurePrivateDir(destination); err != nil {
		return err
	}
	for _, entry := range archive.File {
		name := filepath.Clean(entry.Name)
		if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return fmt.Errorf("caminho inválido no arquivo do navegador: %q", entry.Name)
		}
		path := filepath.Join(destination, name)
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0700); err != nil {
				return err
			}
			continue
		}
		if !entry.FileInfo().Mode().IsRegular() {
			return fmt.Errorf("tipo de arquivo inválido no navegador: %q", entry.Name)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		reader, err := entry.Open()
		if err != nil {
			return err
		}
		writer, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode().Perm())
		if err != nil {
			reader.Close()
			return err
		}
		_, copyErr := io.Copy(writer, reader)
		reader.Close()
		closeErr := writer.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
