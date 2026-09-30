package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var durationPattern = regexp.MustCompile(`^([1-9][0-9]*)([smhd])$`)

const cacheFormatVersion = 4

func ParseDuration(input string) (time.Duration, error) {
	match := durationPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(input)))
	if match == nil {
		return 0, errors.New("duração inválida: use 30m, 12h ou 7d")
	}
	n, err := strconv.ParseInt(match[1], 10, 64)
	if err != nil {
		return 0, errors.New("duração fora do limite")
	}
	unit := map[string]time.Duration{"s": time.Second, "m": time.Minute, "h": time.Hour, "d": 24 * time.Hour}[match[2]]
	if n > int64((365*24*time.Hour)/unit) {
		return 0, errors.New("o prazo máximo do cache é 365d")
	}
	return time.Duration(n) * unit, nil
}

func ValidateURL(input string) error {
	u, err := url.Parse(input)
	if err != nil || u == nil || u.Scheme != "https" || u.User != nil || u.Path == "" || u.Path == "/" {
		return errors.New("use uma URL HTTPS de página do Notion")
	}
	host := strings.ToLower(u.Hostname())
	allowed := host == "notion.so" || host == "notion.site" || host == "notion.com" ||
		strings.HasSuffix(host, ".notion.so") || strings.HasSuffix(host, ".notion.site") || strings.HasSuffix(host, ".notion.com")
	if !allowed || (u.Port() != "" && u.Port() != "443") {
		return errors.New("use uma URL HTTPS de página do Notion")
	}
	return nil
}

func CacheDir() (string, error) {
	if value := os.Getenv("NOTION_GETPAGE_CACHE_DIR"); value != "" {
		return value, nil
	}
	dir, err := RuntimeDir()
	return filepath.Join(dir, "cache"), err
}

func ProfileDir() (string, error) {
	if value := os.Getenv("NOTION_GETPAGE_PROFILE_DIR"); value != "" {
		return value, nil
	}
	dir, err := RuntimeDir()
	return filepath.Join(dir, "profile"), err
}

func RuntimeDir() (string, error) {
	if runtime.GOOS != "linux" {
		return "", errors.New("sessão temporária com limpeza no reinício disponível apenas no Linux")
	}
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		return "", errors.New("XDG_RUNTIME_DIR ausente; é necessário um diretório temporário do usuário")
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return "", fmt.Errorf("verificar XDG_RUNTIME_DIR: %w", err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 {
		return "", errors.New("XDG_RUNTIME_DIR deve ser um diretório privado (0700)")
	}
	return filepath.Join(dir, "notion-getpage"), nil
}

func BrowserDir() (string, error) {
	if value := os.Getenv("NOTION_GETPAGE_BROWSER_DIR"); value != "" {
		return value, nil
	}
	if value := os.Getenv("XDG_CACHE_HOME"); value != "" {
		return filepath.Join(value, "notion-getpage", "browser"), nil
	}
	dir, err := os.UserCacheDir()
	return filepath.Join(dir, "notion-getpage", "browser"), err
}

func LegacyProfileDir() (string, error) {
	if value := os.Getenv("XDG_DATA_HOME"); value != "" {
		return filepath.Join(value, "notion-getpage", "browser"), nil
	}
	dir, err := os.UserHomeDir()
	return filepath.Join(dir, ".local", "share", "notion-getpage", "browser"), err
}

func LegacyCacheDir() (string, error) {
	if value := os.Getenv("XDG_CACHE_HOME"); value != "" {
		return filepath.Join(value, "notion-getpage"), nil
	}
	dir, err := os.UserCacheDir()
	return filepath.Join(dir, "notion-getpage"), err
}

func EnsurePrivateDir(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	return os.Chmod(path, 0700)
}

type cacheEntry struct {
	FormatVersion int       `json:"format_version"`
	URL           string    `json:"url"`
	Markdown      string    `json:"markdown"`
	FetchedAt     time.Time `json:"fetched_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type Cache struct{ Dir string }

func (c Cache) path(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(c.Dir, hex.EncodeToString(sum[:])+".json")
}

func (c Cache) Get(rawURL string, now time.Time) (string, bool) {
	content, err := os.ReadFile(c.path(rawURL))
	if err != nil {
		return "", false
	}
	var entry cacheEntry
	if json.Unmarshal(content, &entry) != nil || entry.FormatVersion != cacheFormatVersion || entry.URL != rawURL || entry.ExpiresAt.IsZero() || !now.Before(entry.ExpiresAt) {
		return "", false
	}
	return entry.Markdown, true
}

func (c Cache) Put(rawURL, markdown string, ttl time.Duration, now time.Time) error {
	if err := EnsurePrivateDir(c.Dir); err != nil {
		return err
	}
	data, err := json.Marshal(cacheEntry{FormatVersion: cacheFormatVersion, URL: rawURL, Markdown: markdown, FetchedAt: now, ExpiresAt: now.Add(ttl)})
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(c.Dir, ".entry-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := file.Chmod(0600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), c.path(rawURL)); err != nil {
		return fmt.Errorf("gravar cache: %w", err)
	}
	return nil
}
