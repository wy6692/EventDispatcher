package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFromEnvRequiresAdminToken(t *testing.T) {
	t.Setenv("ADMIN_TOKEN", "")
	t.Setenv("ADDR", "")
	t.Setenv("DB_PATH", "")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected missing token error")
	}
	t.Setenv("ADMIN_TOKEN", "secret-token")
	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" || cfg.AdminToken != "secret-token" {
		t.Fatalf("%+v", cfg)
	}
}

func TestFromEnvLoadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	content := []byte("# comment\nADMIN_TOKEN=from-file\nADDR=:9090\nexport DB_PATH=\"data/from-file.db\"\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	unsetEnv(t, "ADMIN_TOKEN")
	unsetEnv(t, "ADDR")
	unsetEnv(t, "DB_PATH")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminToken != "from-file" || cfg.Addr != ":9090" || cfg.DBPath != "data/from-file.db" {
		t.Fatalf("%+v", cfg)
	}
}

func TestFromEnvDoesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	content := []byte("ADMIN_TOKEN=from-file\nADDR=:9090\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	chdir(t, dir)
	t.Setenv("ADMIN_TOKEN", "from-env")
	t.Setenv("ADDR", ":7070")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AdminToken != "from-env" || cfg.Addr != ":7070" {
		t.Fatalf("%+v", cfg)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}
