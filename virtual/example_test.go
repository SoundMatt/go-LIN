// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package virtual_test

import (
	"context"
	"fmt"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

// Example demonstrates the smallest complete virtual-bus round trip: a
// slave response is registered with Publish, a subscriber listens for it,
// and SendHeader drives the exchange as a master would.
func Example() {
	bus, err := virtual.New()
	if err != nil {
		panic(err)
	}
	defer bus.Close()

	ch, err := bus.Subscribe([]lin.Filter{{ID: 0x10}})
	if err != nil {
		panic(err)
	}

	if err := bus.Publish(0x10, []byte{0x42, 0x00, 0x00, 0x00}); err != nil {
		panic(err)
	}
	if _, err := bus.SendHeader(context.Background(), 0x10); err != nil {
		panic(err)
	}

	f := <-ch
	fmt.Printf("%02X#%X\n", f.ID, f.Data)
	// Output: 10#42000000
}
