package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var legacyCacheName = regexp.MustCompile(`^[0-9a-f]{64}\.json$`)

// CleanupLegacyData removes private data from the old persistent locations.
// The managed Chrome binary lives below the cache directory and is retained.
func CleanupLegacyData() error {
	profile, err := LegacyProfileDir()
	if err != nil {
		return err
	}
	if override := os.Getenv("NOTION_GETPAGE_PROFILE_DIR"); override != "" && filepath.Clean(override) == filepath.Clean(profile) {
		return fmt.Errorf("o perfil configurado ainda usa o diretório persistente antigo: %s", profile)
	}
	cache, err := LegacyCacheDir()
	if err != nil {
		return err
	}
	if override := os.Getenv("NOTION_GETPAGE_CACHE_DIR"); override != "" && filepath.Clean(override) == filepath.Clean(cache) {
		return fmt.Errorf("o cache configurado ainda usa o diretório persistente antigo: %s", cache)
	}
	if err := os.RemoveAll(profile); err != nil {
		return fmt.Errorf("apagar sessão antiga: %w", err)
	}
	entries, err := os.ReadDir(cache)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("listar cache antigo: %w", err)
	}
	for _, entry := range entries {
		if legacyCacheName.MatchString(entry.Name()) {
			if err := os.Remove(filepath.Join(cache, entry.Name())); err != nil {
				return fmt.Errorf("apagar cache antigo: %w", err)
			}
		}
	}
	return nil
}
