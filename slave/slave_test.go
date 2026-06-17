// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package slave_test

import (
	"context"
	"testing"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/slave"
	"github.com/SoundMatt/go-LIN/virtual"
)

// ── REQ-SLAVE-001: New returns non-nil Node ───────────────────────────────────

//fusa:test REQ-SLAVE-001

func TestNew(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	s := slave.New(bus)
	if s == nil {
		t.Fatal("New returned nil")
	}
}

// ── REQ-SLAVE-002: SetResponse registers response via Publish ─────────────────

//fusa:test REQ-SLAVE-002

func TestSetResponse_registers(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	if err := s.SetResponse(0x10, []byte{0x01, 0x02}); err != nil {
		t.Fatalf("SetResponse: %v", err)
	}

	ids := s.RegisteredIDs()
	if len(ids) != 1 || ids[0] != 0x10 {
		t.Errorf("RegisteredIDs = %v, want [0x10]", ids)
	}
}

// ── REQ-SLAVE-003: SetResponse(nil) removes registration ─────────────────────

//fusa:test REQ-SLAVE-003

func TestSetResponse_remove(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	_ = s.SetResponse(0x10, []byte{0x01})
	_ = s.SetResponse(0x10, nil)

	ids := s.RegisteredIDs()
	if len(ids) != 0 {
		t.Errorf("RegisteredIDs after remove = %v, want []", ids)
	}
}

// ── REQ-SLAVE-004: SetResponse rejects ID > MaxID ────────────────────────────

//fusa:test REQ-SLAVE-004

func TestSetResponse_invalidID(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	if err := s.SetResponse(0x40, []byte{1}); err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

// ── REQ-SLAVE-005: RegisteredIDs reflects current state ──────────────────────

//fusa:test REQ-SLAVE-005

func TestRegisterredIDs_multiple(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	_ = s.SetResponse(0x01, []byte{1})
	_ = s.SetResponse(0x02, []byte{2})
	_ = s.SetResponse(0x03, []byte{3})

	ids := s.RegisteredIDs()
	if len(ids) != 3 {
		t.Errorf("RegisteredIDs count = %d, want 3", len(ids))
	}
}

// ── REQ-SLAVE-006: Subscribe delivers frames ──────────────────────────────────

//fusa:test REQ-SLAVE-006

func TestSubscribe_receivesFrame(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	_ = s.SetResponse(0x10, []byte{0xAB})

	ch, err := s.Subscribe([]lin.Filter{{ID: 0x10}})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// master triggers frame exchange
	_, _ = bus.SendHeader(context.Background(), 0x10)

	select {
	case f := <-ch:
		if f.ID != 0x10 {
			t.Errorf("received frame ID 0x%02X, want 0x10", f.ID)
		}
	default:
		t.Error("slave subscriber did not receive frame")
	}
}

func TestSubscribe_allFrames(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	_ = s.SetResponse(0x01, []byte{0x01})
	_ = s.SetResponse(0x02, []byte{0x02})

	ch, _ := s.Subscribe([]lin.Filter{{All: true}})

	_, _ = bus.SendHeader(context.Background(), 0x01)
	_, _ = bus.SendHeader(context.Background(), 0x02)

	count := 0
	for i := 0; i < 2; i++ {
		select {
		case <-ch:
			count++
		default:
		}
	}
	if count != 2 {
		t.Errorf("received %d frames, want 2", count)
	}
}

// ── REQ-SLAVE-007: RegisteredIDs returns empty slice when none registered ─────

//fusa:test REQ-SLAVE-007

func TestRegisteredIDs_emptyWhenNone(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	ids := s.RegisteredIDs()
	if ids == nil {
		t.Error("RegisteredIDs must return non-nil slice when empty")
	}
	if len(ids) != 0 {
		t.Errorf("RegisteredIDs = %v, want empty", ids)
	}
}

// ── REQ-SLAVE-008: SetResponse overwrites previous registration ───────────────

//fusa:test REQ-SLAVE-008

func TestSetResponse_overwrites(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	_ = s.SetResponse(0x10, []byte{0x01})
	_ = s.SetResponse(0x10, []byte{0x02}) // overwrite

	// Should still be registered only once
	ids := s.RegisteredIDs()
	if len(ids) != 1 {
		t.Errorf("RegisteredIDs count = %d, want 1 after overwrite", len(ids))
	}

	// The new data should be returned by SendHeader
	f, err := bus.SendHeader(context.Background(), 0x10)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.Data[0] != 0x02 {
		t.Errorf("data[0] = 0x%02X, want 0x02 (overwritten value)", f.Data[0])
	}
}
