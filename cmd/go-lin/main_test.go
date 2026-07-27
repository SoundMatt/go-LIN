// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY"
	lin "github.com/SoundMatt/go-LIN"
)

// TestConvert_goldenVector verifies the §11.2 interop driver produces the
// canonical relay.Message for the published LIN golden vector value, with a
// zeroed timestamp so cross-implementation comparison is deterministic.
func TestConvert_goldenVector(t *testing.T) {
	in := `{"id":16,"data":"qrs=","checksum":73,"checksum_type":1}`
	var out, errb bytes.Buffer
	code := cmdConvert([]string{"--protocol", "LIN", "--format", "json"},
		strings.NewReader(in), &out, &errb)
	if code != 0 {
		t.Fatalf("convert exit = %d, stderr = %q", code, errb.String())
	}
	var msg relay.Message
	if err := json.Unmarshal(out.Bytes(), &msg); err != nil {
		t.Fatalf("output is not valid relay.Message JSON: %v", err)
	}
	if msg.Protocol != relay.LIN {
		t.Errorf("Protocol = %v, want LIN", msg.Protocol)
	}
	if msg.ID != "16" {
		t.Errorf("ID = %q, want \"16\"", msg.ID)
	}
	if !msg.Timestamp.IsZero() {
		t.Errorf("Timestamp = %v, want zero (deterministic interop output)", msg.Timestamp)
	}
	if msg.Meta["lin.checksum_type"] != "enhanced" {
		t.Errorf("meta checksum_type = %q, want enhanced", msg.Meta["lin.checksum_type"])
	}
}

// TestConvert_invalidInput rejects a structurally invalid frame with exit 1 and
// the RELAY §5 sentinel name on stderr.
//
//fusa:test REQ-SEC-006
func TestConvert_invalidInput(t *testing.T) {
	in := `{"id":255,"data":"qrs=","checksum":0,"checksum_type":0}` // ID > 0x3F
	var out, errb bytes.Buffer
	code := cmdConvert([]string{"--protocol", "LIN"}, strings.NewReader(in), &out, &errb)
	if code != 1 {
		t.Fatalf("convert exit = %d, want 1", code)
	}
	if got := strings.TrimSpace(errb.String()); got != "ErrInvalidFrame" {
		t.Errorf("stderr = %q, want ErrInvalidFrame", got)
	}
}

// TestConvert_wrongProtocol returns exit 2 (invalid args) for a non-LIN protocol.
func TestConvert_wrongProtocol(t *testing.T) {
	var out, errb bytes.Buffer
	code := cmdConvert([]string{"--protocol", "CAN"}, strings.NewReader(`{}`), &out, &errb)
	if code != 2 {
		t.Errorf("convert exit = %d, want 2", code)
	}
}

// TestSendStream_publishesNDJSON feeds two relay.Message lines (one valid, one
// malformed) and confirms the valid one is published and the malformed one
// skipped without aborting the stream.
func TestSendStream_publishesNDJSON(t *testing.T) {
	in := "{\"protocol\":3,\"id\":\"16\",\"payload\":\"qrs=\",\"meta\":{\"lin.checksum\":\"73\",\"lin.checksum_type\":\"enhanced\"}}\nnot-json\n"
	var out, errb bytes.Buffer
	code := cmdSendStream(strings.NewReader(in), &out, &errb)
	if code != 0 {
		t.Fatalf("send stream exit = %d, stderr = %q", code, errb.String())
	}
	if !strings.Contains(out.String(), "published 1 message") {
		t.Errorf("stdout = %q, want \"published 1 message\"", out.String())
	}
}

// TestVersion_json verifies the §12.1 version document carries the tracked spec
// version and required fields.
func TestVersion_json(t *testing.T) {
	var out bytes.Buffer
	cmdVersion([]string{"--format", "json"}, &out)
	var doc map[string]any
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("version --format json is not valid JSON: %v", err)
	}
	if doc["protocol"] != "LIN" {
		t.Errorf("protocol = %v, want LIN", doc["protocol"])
	}
	if doc["spec_version"] != relay.SpecVersion {
		t.Errorf("spec_version = %v, want %q", doc["spec_version"], relay.SpecVersion)
	}
}

func TestVersion_text(t *testing.T) {
	var out bytes.Buffer
	cmdVersion(nil, &out)
	if !strings.Contains(out.String(), "go-lin") || !strings.Contains(out.String(), "RELAY spec") {
		t.Errorf("text version output unexpected: %q", out.String())
	}
}

// TestCapabilities_advertisesDriverCommands ensures convert/send/subscribe are
// advertised so relay interop and crossbar discover the driver.
func TestCapabilities_advertisesDriverCommands(t *testing.T) {
	var out bytes.Buffer
	cmdCapabilities(&out)
	var doc struct {
		Kind     string   `json:"kind"`
		Commands []string `json:"commands"`
		Adapt    bool     `json:"adapt"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatalf("capabilities is not valid JSON: %v", err)
	}
	if doc.Kind != "capabilities" {
		t.Errorf("kind = %q, want capabilities", doc.Kind)
	}
	want := map[string]bool{"convert": false, "send": false, "subscribe": false}
	for _, c := range doc.Commands {
		if _, ok := want[c]; ok {
			want[c] = true
		}
	}
	for c, seen := range want {
		if !seen {
			t.Errorf("capabilities.commands missing %q", c)
		}
	}
	if !doc.Adapt {
		t.Error("capabilities.adapt = false, want true")
	}
}

func TestStatus_jsonAndText(t *testing.T) {
	var j bytes.Buffer
	cmdStatus([]string{"--format", "json"}, &j)
	var doc map[string]any
	if err := json.Unmarshal(j.Bytes(), &doc); err != nil {
		t.Fatalf("status --format json invalid: %v", err)
	}
	if doc["healthy"] != true {
		t.Errorf("healthy = %v, want true", doc["healthy"])
	}
	var txt bytes.Buffer
	cmdStatus(nil, &txt)
	if !strings.Contains(txt.String(), "healthy") {
		t.Errorf("status text = %q, want it to mention health", txt.String())
	}
}

// TestPID matches the protected identifier for a known frame ID.
func TestPID(t *testing.T) {
	var out bytes.Buffer
	cmdPID([]string{"0x10"}, &out)
	// PID for ID 0x10 is 0x50 (parity P0=1, P1=0).
	if !strings.Contains(out.String(), "ID=0x10") || !strings.Contains(out.String(), "PID=0x50") {
		t.Errorf("pid output = %q, want ID=0x10 PID=0x50", out.String())
	}
}

func TestCS(t *testing.T) {
	var out bytes.Buffer
	cmdCS([]string{"0x10", "AABB"}, &out)
	if !strings.Contains(out.String(), "checksum(enhanced)=0x") {
		t.Errorf("cs output = %q, want an enhanced checksum", out.String())
	}
}

func TestFlagFormat(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--format", "json"}, "json"},
		{[]string{"--format=json"}, "json"},
		{[]string{"--format", "text"}, "text"},
		{[]string{}, ""},
		{[]string{"dump"}, ""},
	}
	for _, tc := range cases {
		if got := flagFormat(tc.args); got != tc.want {
			t.Errorf("flagFormat(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

// TestConvert_wrongFormat rejects a non-json output format with exit 2.
func TestConvert_wrongFormat(t *testing.T) {
	var out, errb bytes.Buffer
	code := cmdConvert([]string{"--protocol", "LIN", "--format", "yaml"},
		strings.NewReader(`{}`), &out, &errb)
	if code != 2 {
		t.Errorf("convert --format yaml exit = %d, want 2", code)
	}
}

// TestFlagValue and TestHasFlag cover the generic --name/--name=value flag
// helpers added for the spec §11.2 send/subscribe flag forms (issue #41).
func TestFlagValue(t *testing.T) {
	cases := []struct {
		args    []string
		name    string
		wantVal string
		wantOK  bool
	}{
		{[]string{"--id", "16"}, "--id", "16", true},
		{[]string{"--id=16"}, "--id", "16", true},
		{[]string{"--data", "01020304"}, "--data", "01020304", true},
		{[]string{"--id"}, "--id", "", false}, // dangling flag, no value follows
		{[]string{}, "--id", "", false},
		{[]string{"--other", "x"}, "--id", "", false},
	}
	for _, tc := range cases {
		got, ok := flagValue(tc.args, tc.name)
		if got != tc.wantVal || ok != tc.wantOK {
			t.Errorf("flagValue(%v, %q) = (%q, %v), want (%q, %v)", tc.args, tc.name, got, ok, tc.wantVal, tc.wantOK)
		}
	}
}

func TestHasFlag(t *testing.T) {
	cases := []struct {
		args []string
		name string
		want bool
	}{
		{[]string{"--id", "16"}, "--id", true},
		{[]string{"--id=16"}, "--id", true},
		{[]string{"--data", "01"}, "--id", false},
		{[]string{}, "--id", false},
	}
	for _, tc := range cases {
		if got := hasFlag(tc.args, tc.name); got != tc.want {
			t.Errorf("hasFlag(%v, %q) = %v, want %v", tc.args, tc.name, got, tc.want)
		}
	}
}

// TestParseSendArgs_flagForm covers the spec §11.2 `send --id <uint> --data
// <hex>` protocol-flag form (issue #41), alongside the pre-existing
// positional form and the invalid-args case.
func TestParseSendArgs_flagForm(t *testing.T) {
	id, data, ok := parseSendArgs([]string{"--id", "16", "--data", "01020304"})
	if !ok {
		t.Fatal("parseSendArgs(--id/--data): ok = false, want true")
	}
	if id != 0x10 {
		t.Errorf("id = 0x%02X, want 0x10", id)
	}
	if string(data) != "\x01\x02\x03\x04" {
		t.Errorf("data = % X, want 01 02 03 04", data)
	}
}

func TestParseSendArgs_positionalForm(t *testing.T) {
	id, data, ok := parseSendArgs([]string{"0x10", "01020304"})
	if !ok {
		t.Fatal("parseSendArgs(positional): ok = false, want true")
	}
	if id != 0x10 || string(data) != "\x01\x02\x03\x04" {
		t.Errorf("parseSendArgs(positional) = (0x%02X, % X)", id, data)
	}
}

func TestParseSendArgs_invalid(t *testing.T) {
	cases := [][]string{
		{},
		{"only-one-arg"},
		{"--id", "16"},   // --data missing
		{"--data", "01"}, // --id missing
	}
	for _, args := range cases {
		if _, _, ok := parseSendArgs(args); ok {
			t.Errorf("parseSendArgs(%v): ok = true, want false", args)
		}
	}
}

// TestRunSubscribe_count verifies `subscribe --format json --count N` (issue
// #41) exits after N messages rather than running until the channel closes
// or the context is cancelled.
func TestRunSubscribe_count(t *testing.T) {
	ch := make(chan lin.Frame, 3)
	ch <- lin.Frame{ID: 0x10, Data: []byte{0x01}}
	ch <- lin.Frame{ID: 0x11, Data: []byte{0x02}}
	ch <- lin.Frame{ID: 0x12, Data: []byte{0x03}}

	var out bytes.Buffer
	runSubscribe(context.Background(), ch, 2, &out)

	lines := strings.Count(strings.TrimSpace(out.String()), "\n") + 1
	if lines != 2 {
		t.Errorf("runSubscribe(count=2) wrote %d line(s), want 2:\n%s", lines, out.String())
	}
}

// TestRunSubscribe_unboundedStopsOnClose verifies count < 0 (no --count
// flag) runs until the channel closes.
func TestRunSubscribe_unboundedStopsOnClose(t *testing.T) {
	ch := make(chan lin.Frame, 2)
	ch <- lin.Frame{ID: 0x10, Data: []byte{0x01}}
	ch <- lin.Frame{ID: 0x11, Data: []byte{0x02}}
	close(ch)

	var out bytes.Buffer
	runSubscribe(context.Background(), ch, -1, &out)

	lines := strings.Count(strings.TrimSpace(out.String()), "\n") + 1
	if lines != 2 {
		t.Errorf("runSubscribe(count=-1) wrote %d line(s), want 2:\n%s", lines, out.String())
	}
}
