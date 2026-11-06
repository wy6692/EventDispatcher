package config

import "testing"

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
