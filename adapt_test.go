// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin_test

import (
	"context"
	"testing"
	"time"

	relay "github.com/SoundMatt/RELAY"
	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

// ── RELAY adapter (spec §10.3 / §13.7) ────────────────────────────────────────

//fusa:test REQ-ADAPT-001
func TestAdapt_protocolIsLIN(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatalf("virtual.New: %v", err)
	}
	defer bus.Close()

	node := lin.Adapt(bus)
	if node == nil {
		t.Fatal("Adapt returned nil")
	}
	if node.Protocol() != relay.LIN {
		t.Errorf("Protocol() = %v, want relay.LIN", node.Protocol())
	}
}

//fusa:test REQ-ADAPT-002
//fusa:test REQ-ADAPT-004
func TestAdapt_sendThenReceive(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatalf("virtual.New: %v", err)
	}
	node := lin.Adapt(bus)
	defer node.Close()

	ch, err := node.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	payload := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	if err := node.Send(context.Background(), relay.Message{ID: "16", Payload: payload}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	// Send registers the slave response; trigger one exchange so the frame is
	// delivered to subscribers.
	if _, err := bus.SendHeader(context.Background(), 16); err != nil {
		t.Fatalf("SendHeader: %v", err)
	}

	select {
	case msg := <-ch:
		if msg.Protocol != relay.LIN {
			t.Errorf("msg.Protocol = %v, want LIN", msg.Protocol)
		}
		if msg.ID != "16" {
			t.Errorf("msg.ID = %q, want \"16\"", msg.ID)
		}
		if string(msg.Payload) != string(payload) {
			t.Errorf("msg.Payload = %x, want %x", msg.Payload, payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for adapted message")
	}
}

//fusa:test REQ-ADAPT-003
func TestAdapt_sendRejectsBadID(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatalf("virtual.New: %v", err)
	}
	node := lin.Adapt(bus)
	defer node.Close()

	for _, id := range []string{"64", "not-a-number", "-1"} {
		if err := node.Send(context.Background(), relay.Message{ID: id, Payload: []byte{1}}); err == nil {
			t.Errorf("Send(ID=%q) = nil error, want rejection", id)
		}
	}
}

//fusa:test REQ-ADAPT-005
func TestAdapt_closeClosesBus(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatalf("virtual.New: %v", err)
	}
	node := lin.Adapt(bus)
	if err := node.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// The underlying bus is closed: a further Publish must fail.
	if err := bus.Publish(1, []byte{1}); err == nil {
		t.Error("Publish after node.Close() = nil error, want closed error")
	}
}
