// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package stats_test

import (
	"fmt"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/stats"
)

// Example collects per-ID frame counts from a handful of observed frames.
// (FrameRate and BusLoad are omitted here since they depend on wall-clock
// time and are not deterministic across test runs.)
func Example() {
	c := stats.New(19.2) // 19.2 kbit/s bus speed
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01}})
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x02}})
	c.Observe(lin.Frame{ID: 0x20, Data: []byte{0x03}})

	fmt.Println(c.TotalCount())
	perID := c.PerID()
	fmt.Println(perID[0x10], perID[0x20])
	// Output:
	// 3
	// 2 1
}
