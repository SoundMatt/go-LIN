// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ldf_test

import (
	"fmt"
	"strings"

	"github.com/SoundMatt/go-LIN/ldf"
)

const exampleLDF = `
LIN_description_file;
LIN_protocol_version = "2.1";
LIN_speed = 19.2 kbps;

Nodes {
  Master: ECU, 1 ms, 0.1 ms;
  Slaves: DoorModule;
}

Signals {
  WindowPos: 8, 0x00, DoorModule, ECU;
}

Frames {
  WindowStatus: 0x10, DoorModule, 1 {
    WindowPos, 0;
  }
}
`

// Example parses an LDF, encodes a signal into a frame payload, then decodes
// it back — the round trip a write-then-read integration typically performs.
func Example() {
	db, err := ldf.Parse(strings.NewReader(exampleLDF))
	if err != nil {
		panic(err)
	}

	payload := db.Encode(0x10, map[string]uint64{"WindowPos": 0x7F})
	fmt.Printf("payload=% X\n", payload)

	decoded := db.Decode(0x10, payload)
	fmt.Printf("WindowPos=0x%02X\n", decoded["WindowPos"])
	// Output:
	// payload=7F
	// WindowPos=0x7F
}
