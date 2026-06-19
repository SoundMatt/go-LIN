// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package slave_test

import (
	"testing"

	"github.com/SoundMatt/go-LIN/slave"
	"github.com/SoundMatt/go-LIN/virtual"
)

func TestSetResponse_rejectsHighID(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	s := slave.New(bus)
	if err := s.SetResponse(0x40, []byte{0x01}); err == nil {
		t.Error("SetResponse(0x40) = nil, want range error")
	}
}

func TestSetResponse_publishErrorAfterClose(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	s := slave.New(bus)
	bus.Close()
	if err := s.SetResponse(0x10, []byte{0x01}); err == nil {
		t.Error("SetResponse on closed bus = nil, want Publish error")
	}
}

// TestSetResponse_removeKeepsOthers exercises removeID's keep-branch: removing
// one ID must leave the others registered.
func TestSetResponse_removeKeepsOthers(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()
	s := slave.New(bus)

	for _, id := range []uint8{0x10, 0x11, 0x12} {
		if err := s.SetResponse(id, []byte{id}); err != nil {
			t.Fatalf("SetResponse(0x%02X): %v", id, err)
		}
	}
	// Remove the middle one (nil data removes registration).
	if err := s.SetResponse(0x11, nil); err != nil {
		t.Fatalf("SetResponse remove: %v", err)
	}
	got := s.RegisteredIDs()
	want := map[uint8]bool{0x10: true, 0x12: true}
	if len(got) != 2 {
		t.Fatalf("RegisteredIDs = %v, want 2 entries", got)
	}
	for _, id := range got {
		if !want[id] {
			t.Errorf("unexpected registered ID 0x%02X", id)
		}
	}
}
