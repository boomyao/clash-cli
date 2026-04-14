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
		fmt.Fprintf(out, "\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  clashc\n")
		fmt.Fprintf(out, "  clashc https://example.com/sub.yaml\n")
		fmt.Fprintf(out, "  clashc --api-url 127.0.0.1:9090 --secret xxx\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("clashc %s\n", version)
		os.Exit(0)
	}

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
	// no matter how we leave (TUI crash, panic, signal).
	exitCode := 0
	defer func() {
		if res.Manager != nil && res.Manager.IsRunning() {
			_ = res.Manager.Stop()
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

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "✗ run TUI: %v\n", err)
		exitCode = 1
	}
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	os.Exit(1)
}
