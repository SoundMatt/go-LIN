// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	relay "github.com/SoundMatt/RELAY"
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
