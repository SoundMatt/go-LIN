// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package master_test

import (
	"context"
	"fmt"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/master"
	"github.com/SoundMatt/go-LIN/virtual"
)

// Example demonstrates a master node triggering a single frame exchange
// with SendHeader — the same call Run makes for each schedule slot, shown
// here directly for a deterministic, single-shot example.
func Example() {
	bus, err := virtual.New()
	if err != nil {
		panic(err)
	}
	defer bus.Close()

	if err := bus.Publish(0x10, []byte{0x42, 0x00, 0x00, 0x00}); err != nil {
		panic(err)
	}

	n := master.New(bus)
	f, err := n.SendHeader(context.Background(), 0x10)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%02X#%X\n", f.ID, f.Data)
	// Output: 10#42000000
}

// ExampleNode_Diagnostics drives one LIN diagnostic request/response
// exchange (LIN 2.x §4.2.3) against a pre-registered slave response.
func ExampleNode_Diagnostics() {
	bus, err := virtual.New()
	if err != nil {
		panic(err)
	}
	defer bus.Close()

	// The diagnostic target's response, as a real slave would register it.
	resp, err := lin.SlaveResponseFrame{NAD: 0x01, RSID: 0xF2, Data: []byte{0xAA}}.ToFrame()
	if err != nil {
		panic(err)
	}
	if err := bus.Publish(lin.LINDiagResponseID, resp.Data); err != nil {
		panic(err)
	}

	n := master.New(bus)
	req := lin.MasterRequestFrame{NAD: 0x01, SID: 0xB2, Data: []byte{0x01}}
	got, err := n.Diagnostics(context.Background(), req)
	if err != nil {
		panic(err)
	}
	fmt.Printf("NAD=0x%02X RSID=0x%02X Data=%X\n", got.NAD, got.RSID, got.Data)
	// Output: NAD=0x01 RSID=0xF2 Data=AA
}
