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
//	send   --format json     Streaming relay.Message NDJSON sink (crossbar spoke, §11.2).
//	subscribe --format json  Streaming relay.Message NDJSON source (crossbar spoke, §11.2).
//	dump                     Subscribe to all frames and print them to stdout.
//	pid    <id>              Compute and display the Protected Identifier.
//	cs     <id> <hex-data>   Compute and display the enhanced checksum.
//
// RELAY interop driver (§11.2):
//
//	convert --protocol LIN [--format json]
//	    Read a lin.Frame as JSON on stdin, run it through Frame.ToMessage() (the
//	    §15.7.3 canonical conversion) and write the relay.Message as JSON on
//	    stdout. Exit 0 converted, 1 invalid input, 2 invalid args.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	relay "github.com/SoundMatt/RELAY"
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
		cmdVersion(os.Args[2:], os.Stdout)
	case "capabilities":
		cmdCapabilities(os.Stdout)
	case "status":
		cmdStatus(os.Args[2:], os.Stdout)
	case "send":
		cmdSend(os.Args[2:])
	case "subscribe":
		cmdSubscribe(os.Args[2:])
	case "convert":
		os.Exit(cmdConvert(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
	case "dump":
		cmdDump()
	case "pid":
		cmdPID(os.Args[2:], os.Stdout)
	case "cs":
		cmdCS(os.Args[2:], os.Stdout)
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
  send --format json     streaming relay.Message NDJSON sink (crossbar spoke)
  subscribe --format json  streaming relay.Message NDJSON source (crossbar spoke)
  dump                   print all frames on the virtual bus until SIGINT
  pid  <id>              compute the Protected Identifier for a 6-bit frame ID
  cs   <id> <hex-data>   compute the enhanced LIN checksum for a frame

RELAY interop driver:
  convert --protocol LIN [--format json]   lin.Frame JSON (stdin) -> relay.Message JSON (stdout)
`, toolName)
}

// ── RELAY mandatory commands ──────────────────────────────────────────────────

func cmdVersion(args []string, w io.Writer) {
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
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Fprintf(w, "%s %s (RELAY spec %s, %s)\n", toolName, ver, lin.SpecVersion, runtime.Version())
	}
}

func cmdCapabilities(w io.Writer) {
	ver := toolVersion()
	out := map[string]any{
		"kind":                "capabilities",
		"tool":                toolName,
		"protocol":            protocolName,
		"protocol_int":        protocolInt,
		"version":             ver,
		"spec_version":        lin.SpecVersion,
		"commands":            []string{"version", "capabilities", "status", "send", "subscribe", "convert", "dump", "pid", "cs"},
		"transports":          []string{"virtual"},
		"features":            []string{},
		"interfaces":          []string{"Bus", "MasterBus"},
		"optional_interfaces": []string{"HealthProvider", "MetricsProvider", "Drainer"},
		"adapt":               true,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func cmdStatus(args []string, w io.Writer) {
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
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Fprintf(w, "%s %s: healthy\n", toolName, ver)
	}
}

// ── Protocol commands ─────────────────────────────────────────────────────────

func cmdSend(args []string) {
	// `send --format json` is the streaming NDJSON sink / crossbar spoke (§11.2);
	// `send <id> <hex-data>` is the ad-hoc single-exchange form.
	if flagFormat(args) == "json" {
		os.Exit(cmdSendStream(os.Stdin, os.Stdout, os.Stderr))
	}
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s send <id> <hex-data> | send --format json\n", toolName)
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

// cmdSendStream is the streaming JSON sink (RELAY §11.2 / crossbar spoke). It
// reads relay.Message values as NDJSON on stdin (one per line) and publishes
// each — via FromMessage → Bus.Publish — until EOF. It is the egress dual of
// `subscribe --format json`. Malformed or undeliverable lines are reported to
// stderr and skipped so a single bad message does not tear down the route; only
// a stdin read error is fatal. Exit: 0 clean EOF, 1 stdin read error.
func cmdSendStream(stdin io.Reader, w, errw io.Writer) int {
	bus, err := virtual.New()
	if err != nil {
		fmt.Fprintf(errw, "%s: virtual.New: %v\n", toolName, err)
		return 1
	}
	defer bus.Close() //nolint:errcheck

	sc := bufio.NewScanner(stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024) // tolerate large messages
	sent := 0
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var msg relay.Message
		if err := json.Unmarshal(line, &msg); err != nil {
			fmt.Fprintf(errw, "send: skipping malformed message: %v\n", err)
			continue
		}
		f, err := lin.FromMessage(msg)
		if err != nil {
			fmt.Fprintf(errw, "send: skipping message %q: %v\n", msg.ID, err)
			continue
		}
		if err := bus.Publish(f.ID, f.Data); err != nil {
			fmt.Fprintf(errw, "send: id %d: %v\n", f.ID, err)
			continue
		}
		sent++
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(errw, "send: read error: %v\n", err)
		return 1
	}
	fmt.Fprintf(w, "published %d message(s)\n", sent)
	return 0
}

// cmdSubscribe is the streaming JSON source (RELAY §11.2 / crossbar spoke). With
// `--format json` it subscribes to every frame on the virtual bus and writes each
// as a one-line relay.Message (NDJSON) to stdout until SIGINT. It is the ingress
// dual of `send --format json`.
func cmdSubscribe(args []string) {
	if flagFormat(args) != "json" {
		fmt.Fprintf(os.Stderr, "usage: %s subscribe --format json\n", toolName)
		os.Exit(1)
	}
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

	enc := json.NewEncoder(os.Stdout)
	var seq uint64
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-ch:
			if !ok {
				return
			}
			msg := f.ToMessage()
			msg.Timestamp = time.Now().UTC()
			msg.Seq = seq
			seq++
			_ = enc.Encode(msg) // one compact JSON object per line (NDJSON)
		}
	}
}

// ── RELAY interop driver (spec §11.2) ─────────────────────────────────────────

// cmdConvert implements `convert --protocol LIN [--format json]` (spec §11.2).
// It reads one lin.Frame as JSON on stdin, validates it with ValidateFrame, and
// converts it via Frame.ToMessage() — the same §15.7.3 code path used at runtime
// — writing the relay.Message as JSON on stdout. The timestamp is left zero so
// interop comparisons are deterministic. On invalid input it writes the RELAY
// §5 sentinel name to stderr. Exit: 0 converted, 1 invalid input, 2 invalid args.
//
//fusa:req REQ-SEC-006
func cmdConvert(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	fs.SetOutput(stderr)
	protocol := fs.String("protocol", "", "canonical protocol (must be LIN)")
	format := fs.String("format", "json", "output format (json)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *protocol != "LIN" {
		fmt.Fprintln(stderr, "convert: --protocol LIN is required")
		return 2
	}
	if *format != "json" {
		fmt.Fprintln(stderr, "convert: only --format json is supported")
		return 2
	}

	raw, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(stderr, lin.ErrInvalidFrame.Error())
		return 1
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var f lin.Frame
	if err := dec.Decode(&f); err != nil {
		fmt.Fprintln(stderr, "ErrInvalidFrame") // §5 sentinel name
		return 1
	}
	if err := lin.ValidateFrame(f); err != nil {
		if errors.Is(err, lin.ErrInvalidFrame) {
			fmt.Fprintln(stderr, "ErrInvalidFrame")
		} else {
			fmt.Fprintln(stderr, err.Error())
		}
		return 1
	}
	msg := f.ToMessage() // deterministic: Timestamp left zero
	out, err := json.Marshal(msg)
	if err != nil {
		fmt.Fprintln(stderr, "ErrInvalidFrame")
		return 1
	}
	fmt.Fprintln(stdout, string(out))
	return 0
}

// flagFormat returns the value of a --format flag (e.g. "json"), or "" if absent.
func flagFormat(args []string) string {
	for i, a := range args {
		switch {
		case a == "--format" && i+1 < len(args):
			return args[i+1]
		case strings.HasPrefix(a, "--format="):
			return strings.TrimPrefix(a, "--format=")
		}
	}
	return ""
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

func cmdPID(args []string, w io.Writer) {
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: %s pid <id>\n", toolName)
		os.Exit(1)
	}
	id := parseID(args[0])
	pid := lin.ProtectID(id)
	fmt.Fprintf(w, "ID=0x%02X  PID=0x%02X  binary=%08b\n", id, pid, pid)
}

func cmdCS(args []string, w io.Writer) {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s cs <id> <hex-data>\n", toolName)
		os.Exit(1)
	}
	id := parseID(args[0])
	data := parseHex(args[1])
	pid := lin.ProtectID(id)
	cs := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	fmt.Fprintf(w, "ID=0x%02X  PID=0x%02X  data=%s  checksum(enhanced)=0x%02X\n",
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
