package main

import "testing"

func TestRun_HelpReturnsZero(t *testing.T) {
	t.Parallel()

	if got := run([]string{"--help"}); got != 0 {
		t.Fatalf("run(--help) = %d; want 0", got)
	}
}

func TestLoadFlagsFromEnvUsesMinecraftVersionEnv(t *testing.T) {
	t.Setenv("MINECRAFT_VER", "1.2.3.4")

	listenAddress := ""
	appDir := ""
	mcVersion := ""
	authKey := ""

	loadFlagsFromEnv(&listenAddress, &appDir, &mcVersion, &authKey)
	if mcVersion != "1.2.3.4" {
		t.Fatalf("expected MINECRAFT_VER to be loaded, got %q", mcVersion)
	}
}
