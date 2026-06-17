// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command go-lin is the RELAY-conformant CLI for go-LIN.
//
// Mandatory RELAY commands (§11.1):
//
//	version [--format text|json]   Print tool and spec version.
//	capabilities                   Print capabilities as JSON.
//	status [--format text|json]    Print self-assessed health.
//
// Protocol commands:
//
//	send   <id> <hex-data>   Publish a response for a frame ID and trigger one exchange.
//	dump                     Subscribe to all frames and print them to stdout.
//	pid    <id>              Compute and display the Protected Identifier.
//	cs     <id> <hex-data>   Compute and display the enhanced checksum.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

const toolName = "go-lin"
const protocolName = "LIN"
const protocolInt = 3

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "version":
		cmdVersion(os.Args[2:])
	case "capabilities":
		cmdCapabilities()
	case "status":
		cmdStatus(os.Args[2:])
	case "send":
		cmdSend(os.Args[2:])
	case "dump":
		cmdDump()
	case "pid":
		cmdPID(os.Args[2:])
	case "cs":
		cmdCS(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "%s: unknown subcommand %q\n", toolName, os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s <subcommand> [args]

RELAY mandatory commands:
  version [--format text|json]   print tool and spec version
  capabilities                   print capabilities document as JSON
  status [--format text|json]    print self-assessed health

Protocol commands:
  send <id> <hex-data>   publish response for <id> and trigger one frame exchange
  dump                   print all frames on the virtual bus until SIGINT
  pid  <id>              compute the Protected Identifier for a 6-bit frame ID
  cs   <id> <hex-data>   compute the enhanced LIN checksum for a frame
`, toolName)
}

// ── RELAY mandatory commands ──────────────────────────────────────────────────

func cmdVersion(args []string) {
	format := "text"
	for _, a := range args {
		if a == "--format=json" || a == "json" {
			format = "json"
		} else if a == "--format=text" || a == "text" {
			format = "text"
		}
	}

	ver := toolVersion()
	if format == "json" {
		out := map[string]any{
			"tool":         toolName,
			"protocol":     protocolName,
			"protocol_int": protocolInt,
			"version":      ver,
			"spec_version": lin.SpecVersion,
			"language":     "go",
			"runtime":      runtime.Version(),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Printf("%s %s (RELAY spec %s, %s)\n", toolName, ver, lin.SpecVersion, runtime.Version())
	}
}

func cmdCapabilities() {
	ver := toolVersion()
	out := map[string]any{
		"kind":                "capabilities",
		"tool":                toolName,
		"protocol":            protocolName,
		"protocol_int":        protocolInt,
		"version":             ver,
		"spec_version":        lin.SpecVersion,
		"commands":            []string{"version", "capabilities", "status", "send", "dump", "pid", "cs"},
		"transports":          []string{"virtual"},
		"features":            []string{},
		"interfaces":          []string{"Bus", "MasterBus"},
		"optional_interfaces": []string{"HealthProvider", "MetricsProvider", "Drainer"},
		"adapt":               true,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func cmdStatus(args []string) {
	format := "text"
	for _, a := range args {
		if a == "--format=json" || a == "json" {
			format = "json"
		}
	}
	ver := toolVersion()
	if format == "json" {
		out := map[string]any{
			"protocol":  protocolName,
			"tool":      toolName,
			"version":   ver,
			"healthy":   true,
			"connected": false,
			"endpoint":  "",
			"details":   map[string]any{},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Printf("%s %s: healthy\n", toolName, ver)
	}
}

// ── Protocol commands ─────────────────────────────────────────────────────────

func cmdSend(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s send <id> <hex-data>\n", toolName)
		os.Exit(1)
	}
	id := parseID(args[0])
	data := parseHex(args[1])

	bus, err := virtual.New()
	if err != nil {
		fatal("virtual.New: %v", err)
	}
	defer bus.Close()

	if err := bus.Publish(id, data); err != nil {
		fatal("Publish: %v", err)
	}

	f, err := bus.SendHeader(context.Background(), id)
	if err != nil {
		fatal("SendHeader: %v", err)
	}
	printFrame(f)
}

func cmdDump() {
	bus, err := virtual.New()
	if err != nil {
		fatal("virtual.New: %v", err)
	}
	defer bus.Close()

	ch, err := bus.Subscribe([]lin.Filter{{All: true}})
	if err != nil {
		fatal("Subscribe: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fmt.Fprintf(os.Stderr, "%s dump: waiting for frames (CTRL+C to stop)\n", toolName)
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-ch:
			if !ok {
				return
			}
			printFrame(f)
		}
	}
}

func cmdPID(args []string) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s pid <id>\n", toolName)
		os.Exit(1)
	}
	id := parseID(args[0])
	pid := lin.ProtectID(id)
	fmt.Printf("ID=0x%02X  PID=0x%02X  binary=%08b\n", id, pid, pid)
}

func cmdCS(args []string) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s cs <id> <hex-data>\n", toolName)
		os.Exit(1)
	}
	id := parseID(args[0])
	data := parseHex(args[1])
	pid := lin.ProtectID(id)
	cs := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	fmt.Printf("ID=0x%02X  PID=0x%02X  data=%s  checksum(enhanced)=0x%02X\n",
		id, pid, strings.ToUpper(hex.EncodeToString(data)), cs)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printFrame(f lin.Frame) {
	fmt.Printf("%02X#%s  cs=0x%02X\n",
		f.ID, strings.ToUpper(hex.EncodeToString(f.Data)), f.Checksum)
}

func parseID(s string) uint8 {
	var v uint64
	var err error
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err = strconv.ParseUint(s[2:], 16, 8)
	} else {
		v, err = strconv.ParseUint(s, 10, 8)
	}
	if err != nil || v > lin.LINMaxID {
		fatal("invalid LIN frame ID %q (must be 0x00–0x3F)", s)
	}
	return uint8(v)
}

func parseHex(s string) []byte {
	s = strings.ReplaceAll(s, " ", "")
	b, err := hex.DecodeString(s)
	if err != nil {
		fatal("invalid hex data %q: %v", s, err)
	}
	if len(b) == 0 || len(b) > lin.LINMaxDataLen {
		fatal("data length %d is not in range 1–%d", len(b), lin.LINMaxDataLen)
	}
	return b
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, toolName+": "+format+"\n", args...)
	os.Exit(1)
}

func toolVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			return v
		}
	}
	return "dev"
}
