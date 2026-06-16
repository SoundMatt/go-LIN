// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command lintool provides a CLI for interacting with the go-LIN virtual bus.
//
// Subcommands:
//
//	send   <id> <hex-data>   Publish a response for a frame ID and trigger it once.
//	dump                     Subscribe to all frames and print them to stdout.
//	pid    <id>              Compute and display the Protected Identifier for a 6-bit ID.
//	cs     <id> <hex-data>   Compute and display the enhanced checksum.
package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "send":
		cmdSend(os.Args[2:])
	case "dump":
		cmdDump()
	case "pid":
		cmdPID(os.Args[2:])
	case "cs":
		cmdCS(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "lintool: unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage: lintool <subcommand> [args]

Subcommands:
  send <id> <hex-data>   publish response for <id> and trigger one frame exchange
  dump                   print all frames on the virtual bus until SIGINT
  pid  <id>              compute the Protected Identifier for a 6-bit frame ID
  cs   <id> <hex-data>   compute the enhanced LIN checksum for a frame
`)
}

func cmdSend(args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: lintool send <id> <hex-data>")
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

	fmt.Fprintln(os.Stderr, "lintool dump: waiting for frames (CTRL+C to stop)")
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
		fmt.Fprintln(os.Stderr, "usage: lintool pid <id>")
		os.Exit(1)
	}
	id := parseID(args[0])
	pid := lin.ProtectID(id)
	fmt.Printf("ID=0x%02X  PID=0x%02X  binary=%08b\n", id, pid, pid)
}

func cmdCS(args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: lintool cs <id> <hex-data>")
		os.Exit(1)
	}
	id := parseID(args[0])
	data := parseHex(args[1])
	pid := lin.ProtectID(id)
	cs := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	fmt.Printf("ID=0x%02X  PID=0x%02X  data=%s  checksum(enhanced)=0x%02X\n",
		id, pid, strings.ToUpper(hex.EncodeToString(data)), cs)
}

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
	if err != nil || v > lin.MaxID {
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
	if len(b) == 0 || len(b) > lin.MaxDataLen {
		fatal("data length %d is not in range 1–%d", len(b), lin.MaxDataLen)
	}
	return b
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "lintool: "+format+"\n", args...)
	os.Exit(1)
}
