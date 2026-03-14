package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"uptui/internal/daemon"
	"uptui/internal/ipc"
	"uptui/internal/models"
	"uptui/internal/store"
	"uptui/internal/tui"
)

const ipcAddr = "127.0.0.1:29374"

func main() {
	if len(os.Args) < 2 {
		runTUI()
		return
	}
	switch os.Args[1] {
	case "daemon":
		runDaemon()
	case "stop":
		runStop()
	case "status":
		runStatus()
	case "add":
		runAdd(os.Args[2:])
	case "edit":
		runEdit(os.Args[2:])
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

// ── TUI ────────────────────────────────────────────────────────────────────────

func runTUI() {
	if err := ensureDaemon(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not start daemon: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'uptui daemon' in a separate terminal.")
	}

	client := ipc.NewClient(ipcAddr)
	m := tui.NewModel(client)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "tui error: %v\n", err)
		os.Exit(1)
	}
}

// ensureDaemon starts the daemon in the background if it is not already running.
func ensureDaemon() error {
	client := ipc.NewClient(ipcAddr)
	if client.Ping() {
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}

	dir := dataDir()
	os.MkdirAll(dir, 0755)

	logPath := filepath.Join(dir, "daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logFile, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	}

	cmd := exec.Command(exe, "daemon")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}

	// Wait up to 3 s for daemon to become ready
	for i := 0; i < 10; i++ {
		time.Sleep(300 * time.Millisecond)
		if client.Ping() {
			return nil
		}
	}
	return fmt.Errorf("daemon did not respond after 3 s (check %s)", logPath)
}

// ── daemon ─────────────────────────────────────────────────────────────────────

func runDaemon() {
	dir := dataDir()
	pidFile := filepath.Join(dir, "daemon.pid")
	configFile := filepath.Join(dir, "monitors.toml")

	// Detect if another instance is already running
	client := ipc.NewClient(ipcAddr)
	if client.Ping() {
		fmt.Fprintln(os.Stderr, "daemon is already running")
		os.Exit(1)
	}

	s, err := store.New(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}

	// Write PID file
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d\n", os.Getpid())), 0644)
	defer os.Remove(pidFile)

	ctx, cancel := context.WithCancel(context.Background())

	// Shutdown on interrupt
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		<-sig
		cancel()
	}()

	fmt.Fprintf(os.Stderr, "uptui daemon listening on %s  (data: %s)\n", ipcAddr, dir)

	d := daemon.New(s, configFile)
	if err := d.Run(ctx, ipcAddr); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		os.Exit(1)
	}
}

// ── stop ───────────────────────────────────────────────────────────────────────

func runStop() {
	pidFile := filepath.Join(dataDir(), "daemon.pid")
	b, err := os.ReadFile(pidFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "daemon is not running (no PID file)")
		os.Exit(1)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid PID file: %v\n", err)
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "process %d not found: %v\n", pid, err)
		os.Remove(pidFile)
		os.Exit(1)
	}

	if err := proc.Kill(); err != nil {
		fmt.Fprintf(os.Stderr, "kill %d: %v\n", pid, err)
		os.Exit(1)
	}

	os.Remove(pidFile)
	fmt.Printf("daemon (PID %d) stopped\n", pid)
}

// ── status ─────────────────────────────────────────────────────────────────────

func runStatus() {
	client := ipc.NewClient(ipcAddr)
	monitors, err := client.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(monitors) == 0 {
		fmt.Println("no monitors configured")
		return
	}

	fmt.Printf("%-25s  %-5s  %-8s  %s\n", "NAME", "TYPE", "STATUS", "LATENCY")
	fmt.Println(strings.Repeat("─", 55))
	for _, ms := range monitors {
		lat := "-"
		if ms.Latency > 0 {
			lat = fmt.Sprintf("%d ms", ms.Latency)
		}
		fmt.Printf("%-25s  %-5s  %-8s  %s\n",
			truncateStr(ms.Monitor.Name, 25),
			ms.Monitor.Type,
			ms.Status,
			lat,
		)
	}
}

// ── add ────────────────────────────────────────────────────────────────────────

func runAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: uptui add [--name NAME] [--type http|tcp] [--interval N] TARGET")
		os.Exit(1)
	}

	mon := models.Monitor{
		Type:     models.HTTP,
		Interval: 60,
		Timeout:  30,
		Active:   true,
	}

	remaining := []string{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name", "-n":
			i++
			if i < len(args) {
				mon.Name = args[i]
			}
		case "--type", "-t":
			i++
			if i < len(args) {
				mon.Type = models.MonitorType(args[i])
			}
		case "--interval", "-i":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &mon.Interval)
			}
		default:
			remaining = append(remaining, args[i])
		}
	}

	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "error: TARGET is required")
		os.Exit(1)
	}
	mon.Target = remaining[0]

	if mon.Name == "" {
		mon.Name = mon.Target
		if len(mon.Name) > 40 {
			mon.Name = mon.Name[:40]
		}
	}

	if mon.Interval < 10 {
		mon.Interval = 10
	}

	client := ipc.NewClient(ipcAddr)
	ms, err := client.Add(mon)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("added monitor: %s (%s)\n", ms.Monitor.Name, ms.Monitor.Target)
}

// ── edit ───────────────────────────────────────────────────────────────────────

func runEdit(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: uptui edit NAME [--name NEWNAME] [--target URL] [--type TYPE] [--interval N] [--timeout N]")
		os.Exit(1)
	}

	oldName := args[0]
	args = args[1:]

	// Fetch current monitor to use as defaults
	client := ipc.NewClient(ipcAddr)
	monitors, err := client.List()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var current *models.Monitor
	for _, ms := range monitors {
		if ms.Monitor.Name == oldName {
			m := ms.Monitor
			current = &m
			break
		}
	}
	if current == nil {
		fmt.Fprintf(os.Stderr, "monitor %q not found\n", oldName)
		os.Exit(1)
	}

	m := *current
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--name":
			i++
			if i < len(args) {
				m.Name = args[i]
			}
		case "--target":
			i++
			if i < len(args) {
				m.Target = args[i]
			}
		case "--type":
			i++
			if i < len(args) {
				m.Type = models.MonitorType(args[i])
			}
		case "--interval":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &m.Interval)
			}
		case "--timeout":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &m.Timeout)
			}
		}
	}

	ms, err := client.Edit(oldName, m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("updated monitor: %s (%s)\n", ms.Monitor.Name, ms.Monitor.Target)
}

// ── helpers ────────────────────────────────────────────────────────────────────

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".uptui")
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func printHelp() {
	fmt.Print(`uptui - uptime monitor daemon + TUI

Usage:
  uptui                    open TUI (auto-starts daemon if needed)
  uptui daemon             run daemon in foreground
  uptui stop               stop the background daemon
  uptui status             print monitor status to stdout
  uptui add TARGET         add a monitor  [--name NAME] [--type http|tcp] [--interval N]
  uptui edit NAME          edit a monitor [--name NEWNAME] [--target URL] [--type TYPE] [--interval N] [--timeout N]

Config: ~/.uptui/monitors.toml  (edit by hand; daemon picks up changes within 5 s)
Data:   ~/.uptui/
`)
}
