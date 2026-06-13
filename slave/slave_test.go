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

//fusa:test REQ-SLAVE-001
//fusa:test REQ-SLAVE-002
//fusa:test REQ-SLAVE-003

func TestNew(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	s := slave.New(bus)
	if s == nil {
		t.Fatal("New returned nil")
	}
}

func TestSetResponse(t *testing.T) {
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

func TestSetResponse_invalidID(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	if err := s.SetResponse(0x40, []byte{1}); err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

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

func TestSubscribe_receivesFrame(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	s := slave.New(bus)
	_ = s.SetResponse(0x10, []byte{0xAB})

	ch, err := s.Subscribe(lin.Filter{ID: 0x10})
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

	ch, _ := s.Subscribe(lin.Filter{All: true})

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
