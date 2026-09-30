package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCacheTTLAndURLKey(t *testing.T) {
	c := Cache{Dir: t.TempDir()}
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	u1 := "https://www.notion.so/Test-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	u2 := u1 + "?pvs=4"
	if err := c.Put(u1, "# Olá\n", time.Hour, now); err != nil {
		t.Fatal(err)
	}
	if md, ok := c.Get(u1, now.Add(59*time.Minute)); !ok || md != "# Olá\n" {
		t.Fatalf("expected cache hit, got %q %v", md, ok)
	}
	if _, ok := c.Get(u1, now.Add(time.Hour)); ok {
		t.Fatal("expired entry must miss")
	}
	if _, ok := c.Get(u2, now); ok {
		t.Fatal("different URLs must have different keys")
	}
	if err := os.WriteFile(c.path(u1), []byte(`{"url":"`+u1+`","markdown":"old","expires_at":"2099-01-01T00:00:00Z"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get(u1, now); ok {
		t.Fatal("legacy cache entry must miss after extraction changes")
	}
	info, err := os.Stat(c.path(u1))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("cache file mode = %v", info.Mode().Perm())
	}
	dir, err := os.Stat(filepath.Dir(c.path(u1)))
	if err != nil || dir.Mode().Perm() != 0700 {
		t.Fatalf("cache directory mode = %v, err = %v", dir.Mode().Perm(), err)
	}
}

func TestInputValidation(t *testing.T) {
	for _, input := range []string{
		"https://www.notion.so/Test-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"https://example.notion.site/Page-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	} {
		if err := ValidateURL(input); err != nil {
			t.Errorf("valid URL %q: %v", input, err)
		}
	}
	for _, input := range []string{
		"http://www.notion.so/Page", "https://notion.so.evil.com/Page", "https://user:pass@notion.so/Page", "https://notion.so/",
	} {
		if err := ValidateURL(input); err == nil {
			t.Errorf("accepted invalid URL %q", input)
		}
	}
	if _, err := ParseDuration("2h"); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"0s", "2w", "366d", "xyz"} {
		if _, err := ParseDuration(input); err == nil {
			t.Errorf("accepted invalid duration %q", input)
		}
	}
}

func TestRuntimeStorageAndLegacyCleanup(t *testing.T) {
	runtimeDir := t.TempDir()
	if err := os.Chmod(runtimeDir, 0700); err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	cacheHome := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	t.Setenv("NOTION_GETPAGE_PROFILE_DIR", "")
	t.Setenv("NOTION_GETPAGE_CACHE_DIR", "")
	t.Setenv("NOTION_GETPAGE_BROWSER_DIR", "")

	profile, err := ProfileDir()
	if err != nil || profile != filepath.Join(runtimeDir, "notion-getpage", "profile") {
		t.Fatalf("profile = %q, err = %v", profile, err)
	}
	cache, err := CacheDir()
	if err != nil || cache != filepath.Join(runtimeDir, "notion-getpage", "cache") {
		t.Fatalf("cache = %q, err = %v", cache, err)
	}
	browser, err := BrowserDir()
	if err != nil || browser != filepath.Join(cacheHome, "notion-getpage", "browser") {
		t.Fatalf("browser = %q, err = %v", browser, err)
	}
	legacyProfile, _ := LegacyProfileDir()
	if err := os.MkdirAll(legacyProfile, 0700); err != nil {
		t.Fatal(err)
	}
	legacyCache, _ := LegacyCacheDir()
	if err := os.MkdirAll(browser, 0700); err != nil {
		t.Fatal(err)
	}
	oldEntry := filepath.Join(legacyCache, strings.Repeat("a", 64)+".json")
	for path, contents := range map[string]string{
		filepath.Join(legacyProfile, "session"): "secret",
		oldEntry:                                "markdown",
		filepath.Join(browser, "current"):       "binary version",
	} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := CleanupLegacyData(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{legacyProfile, oldEntry} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("old private data remains at %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(browser, "current")); err != nil {
		t.Fatalf("browser binary metadata was removed: %v", err)
	}
}
