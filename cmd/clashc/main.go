package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/boomyao/clash-cli/internal/bootstrap"
	"github.com/boomyao/clash-cli/internal/config"
	"github.com/boomyao/clash-cli/internal/tui"
	"github.com/boomyao/clash-cli/internal/updater"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
)

func main() {
	apiURL := flag.String("api-url", "", "mihomo external controller URL (e.g., 127.0.0.1:9090)")
	secret := flag.String("secret", "", "mihomo API secret")
	mihomobin := flag.String("mihomo-bin", "", "path to mihomo binary")
	configDir := flag.String("config-dir", "", "mihomo config directory")
	noAutoStart := flag.Bool("no-autostart", false, "do not auto-launch mihomo if it is not running")
	showVersion := flag.Bool("version", false, "show version and exit")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "clashc %s — TUI client for mihomo (Clash Meta) — github.com/boomyao/clash-cli\n\n", version)
		fmt.Fprintf(out, "Usage:\n")
		fmt.Fprintf(out, "  clashc                  Connect to mihomo (auto-start if not running)\n")
		fmt.Fprintf(out, "  clashc <subscription>   Import subscription URL, then start TUI\n")
		fmt.Fprintf(out, "  clashc update           Download and install the latest release\n")
		fmt.Fprintf(out, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  clashc\n")
		fmt.Fprintf(out, "  clashc 'https://example.com/sub.yaml'\n")
		fmt.Fprintf(out, "  clashc update\n")
		fmt.Fprintf(out, "  clashc --api-url 127.0.0.1:9090 --secret xxx\n")
	}
	flag.Parse()

	// 'update' subcommand: short-circuit before doing anything else
	if args := flag.Args(); len(args) > 0 && args[0] == "update" {
		if _, err := updater.Run(version, os.Stderr); err != nil {
			fail("update: %v", err)
		}
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("clashc %s\n", version)
		os.Exit(0)
	}

	// Tell the TUI which version we are so the update-check can compare.
	tui.SetVersion(version)

	if err := config.EnsureDirs(); err != nil {
		fail("create directories: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		fail("load config: %v", err)
	}

	// Apply flag overrides
	if *apiURL != "" {
		cfg.API.ExternalController = *apiURL
	}
	if *secret != "" {
		cfg.API.Secret = *secret
	}
	if *mihomobin != "" {
		cfg.Core.BinaryPath = *mihomobin
	}
	if *configDir != "" {
		cfg.Core.ConfigDir = *configDir
	}
	if *noAutoStart {
		cfg.Core.AutoStart = false
	}

	// Positional argument: subscription URL or local yaml path
	var urlArg string
	if args := flag.Args(); len(args) > 0 {
		urlArg = args[0]
		if !bootstrap.LooksLikeURL(urlArg) {
			fail("expected a subscription URL (http:// or https://), got: %s", urlArg)
		}
	}

	// Persist any flag-driven changes
	_ = cfg.Save()

	// Bootstrap: download subscription if needed, ensure mihomo is running
	fmt.Fprintln(os.Stderr, "▶ Preparing clashc...")
	res, err := bootstrap.Run(cfg, urlArg)
	if err != nil {
		fail("%v", err)
	}
	if res.Manager != nil {
		fmt.Fprintln(os.Stderr, "✓ Started mihomo")
	} else {
		fmt.Fprintln(os.Stderr, "✓ Connected to existing mihomo")
	}

	// Cleanup-on-exit: if we launched mihomo, make sure it dies with us
	// — UNLESS the user toggled "background mode" inside the TUI, in which
	// case we leave it running so it survives clashc exit.
	exitCode := 0
	keepMihomo := false
	defer func() {
		if res.Manager != nil && res.Manager.IsRunning() && !keepMihomo {
			_ = res.Manager.Stop()
		}
		if keepMihomo && res.Manager != nil {
			fmt.Fprintln(os.Stderr, "✓ mihomo left running in background (PID ownership released)")
		}
		os.Exit(exitCode)
	}()

	// Launch TUI
	model := tui.NewAppModel(cfg, res.Manager)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		p.Quit()
	}()

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ run TUI: %v\n", err)
		exitCode = 1
	}
	if fm, ok := finalModel.(tui.AppModel); ok {
		keepMihomo = fm.BackgroundMode()
	}
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	os.Exit(1)
}
