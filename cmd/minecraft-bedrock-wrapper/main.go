package main

import (
	"errors"
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
	logger := slog.New(slog.DiscardHandler)
	flags, err := parseFlags(args)
	if err != nil {
		return handleFlagError(err)
	}

	if err = validateRuntimeConfig(flags.authKey); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	workDir, err := resolveWorkDir(*flags.appDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
		return 1
	}

	if err = prepareRuntime(flags, workDir, logger); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	return runServer(flags, logger)
}

func parseFlags(args []string) (cliFlags, error) {
	fs := flag.NewFlagSet("bedrock", flag.ContinueOnError)
	flags := cliFlags{}
	flags.command = fs.String("command", "./bedrock_server", "command to execute (used for debugging purposes)")
	flags.listenAddress = fs.String("listen", ":8080", "address for the web server")
	flags.appDir = fs.String("app-dir", "", "directory containing the minecraft server (defaults to current directory)")
	flags.mcVersion = fs.String("mc-version", "", "Minecraft version to download (if not already present)")
	flags.authKey = fs.String(
		"auth-key",
		"",
		"pre-shared key for authentication (recommended to use AUTH_KEY env var instead)",
	)

	loadFlagsFromEnv(flags.listenAddress, flags.appDir, flags.mcVersion, flags.authKey)
	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	return flags, nil
}

type cliFlags struct {
	command       *string
	listenAddress *string
	appDir        *string
	mcVersion     *string
	authKey       *string
}

func handleFlagError(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
	return 1
}

func validateRuntimeConfig(authKey *string) error {
	if *authKey == "" {
		return fmt.Errorf(
			"error: authentication key is required. Set it using the AUTH_KEY environment variable or --auth-key flag",
		)
	}
	if err := os.Setenv("LD_LIBRARY_PATH", "."); err != nil {
		return fmt.Errorf("error setting LD_LIBRARY_PATH: %w", err)
	}
	if !eulaAccepted() {
		return fmt.Errorf(
			"you must accept the EULA by setting EULA_ACCEPT to 'true'\nLinks:\n   https://minecraft.net/eula\n   https://go.microsoft.com/fwlink/?LinkId=521839",
		)
	}
	return nil
}

func prepareRuntime(flags cliFlags, workDir string, logger *slog.Logger) error {
	if *flags.mcVersion != "" {
		logger.Info("Downloading Minecraft server version", "version", *flags.mcVersion)
		if downloadErr := downloader.DownloadMinecraftServer(*flags.mcVersion, workDir, ""); downloadErr != nil {
			return fmt.Errorf("error downloading server: %w", downloadErr)
		}
	}
	if updateErr := config.UpdateServerProperties(workDir, logger); updateErr != nil {
		return fmt.Errorf("error updating server properties: %w", updateErr)
	}
	return nil
}

func runServer(flags cliFlags, logger *slog.Logger) int {
	cmdRunner := runner.New(*flags.command)
	if startErr := cmdRunner.Start(); startErr != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", startErr)
		return 1
	}

	srv := server.New(server.Config{Runner: cmdRunner, AuthKey: *flags.authKey, Logger: logger})
	serverErrCh := make(chan error, 1)
	go func() {
		if startErr := srv.Start(*flags.listenAddress); startErr != nil {
			serverErrCh <- fmt.Errorf("error starting web server: %w", startErr)
		}
	}()

	runnerDoneCh := make(chan error, 1)
	go func() {
		runnerDoneCh <- cmdRunner.Wait()
	}()

	select {
	case serverErr := <-serverErrCh:
		fmt.Fprintf(os.Stderr, "%v\n", serverErr)
		return 1
	case waitErr := <-runnerDoneCh:
		if waitErr != nil {
			fmt.Fprintf(os.Stderr, "Error running command: %v\n", waitErr)
			return 1
		}
		return 0
	}
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
