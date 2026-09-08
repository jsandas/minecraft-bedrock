package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jsandas/bedrock-server/internal/config"
	"github.com/jsandas/bedrock-server/internal/downloader"
	"github.com/jsandas/bedrock-server/internal/runner"
	"github.com/jsandas/bedrock-server/internal/server"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("bedrock", flag.ContinueOnError)
	command := fs.String(
		"command",
		"./bedrock_server",
		"command to execute (used for debugging purposes)",
	)
	listenAddress := fs.String("listen", ":8080", "address for the web server")
	appDir := fs.String("app-dir", "", "directory containing the minecraft server (defaults to current directory)")
	mcVersion := fs.String("mc-version", "", "Minecraft version to download (if not already present)")
	authKey := fs.String(
		"auth-key",
		"",
		"pre-shared key for authentication (recommended to use AUTH_KEY env var instead)",
	)

	loadFlagsFromEnv(listenAddress, appDir, mcVersion, authKey)

	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		return 1
	}

	if *authKey == "" {
		fmt.Fprintf(
			os.Stderr,
			"Error: Authentication key is required. Set it using the "+
				"AUTH_KEY environment variable or --auth-key flag\n",
		)
		return 1
	}

	if err := os.Setenv("LD_LIBRARY_PATH", "."); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting LD_LIBRARY_PATH: %v\n", err)
		return 1
	}

	if !eulaAccepted() {
		fmt.Fprintf(os.Stderr, "You must accept the EULA by setting EULA_ACCEPT to 'true'\n Links:\n")
		fmt.Fprintf(os.Stderr, "   https://minecraft.net/eula\n")
		fmt.Fprintf(os.Stderr, "   https://go.microsoft.com/fwlink/?LinkId=521839\n")
		return 1
	}

	workDir, err := resolveWorkDir(*appDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		return 1
	}

	if *mcVersion != "" {
		slog.Default().Info("Downloading Minecraft server version", "version", *mcVersion)
		if downloadErr := downloader.DownloadMinecraftServer(*mcVersion, workDir, ""); downloadErr != nil {
			fmt.Fprintf(os.Stderr, "Error downloading server: %v\n", downloadErr)
			return 1
		}
	}

	if updateErr := config.UpdateServerProperties(workDir); updateErr != nil {
		fmt.Fprintf(os.Stderr, "Error updating server properties: %v\n", updateErr)
		return 1
	}

	cmdRunner := runner.New(*command)
	if startErr := cmdRunner.Start(); startErr != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", startErr)
		return 1
	}

	srv := server.New(server.Config{
		Runner:  cmdRunner,
		AuthKey: *authKey,
	})
	go func() {
		if startErr := srv.Start(*listenAddress); startErr != nil {
			fmt.Fprintf(os.Stderr, "Error starting web server: %v\n", startErr)
			os.Exit(1)
		}
	}()

	if waitErr := cmdRunner.Wait(); waitErr != nil {
		fmt.Fprintf(os.Stderr, "Error running command: %v\n", waitErr)
		return 1
	}

	return 0
}

func loadFlagsFromEnv(listenAddress, appDir, mcVersion, authKey *string) {
	if envListenAddress := os.Getenv("LISTEN_ADDRESS"); envListenAddress != "" {
		*listenAddress = envListenAddress
	}
	if envAppDir := os.Getenv("APP_DIR"); envAppDir != "" {
		*appDir = envAppDir
	}
	if envMcVer := os.Getenv("MINECRAFT_VER"); envMcVer != "" {
		*mcVersion = envMcVer
	}
	if envAuthKey := os.Getenv("AUTH_KEY"); envAuthKey != "" {
		*authKey = envAuthKey
	}
}

func eulaAccepted() bool {
	return os.Getenv("EULA_ACCEPT") == "true"
}

func resolveWorkDir(appDir string) (string, error) {
	if appDir != "" {
		return appDir, nil
	}

	return os.Getwd()
}
