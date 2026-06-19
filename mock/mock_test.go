// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package mock_test

import (
	"testing"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/mock"
)

//fusa:test REQ-MOCK-001
func TestNew_satisfiesMasterBus(t *testing.T) {
	bus, err := mock.New()
	if err != nil {
		t.Fatalf("mock.New: %v", err)
	}
	if bus == nil {
		t.Fatal("mock.New returned nil bus")
	}
	defer bus.Close()

	// Compile-time + runtime assertion that the mock satisfies the full
	// lin.MasterBus contract (RELAY §7 rule 4).
	var _ lin.MasterBus = bus
}
