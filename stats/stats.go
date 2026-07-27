// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package stats provides observability counters for a LIN bus: frames
// received per second, per-ID frame counts, and an estimated bus load
// percentage. It is a passive observer — feed it frames via Observe (e.g.
// from a lin.Bus Subscribe channel) — and has no effect on bus behaviour.
//
// Usage:
//
//	c := stats.New(19200) // bus speed in bps
//	ch, _ := bus.Subscribe(nil)
//	for f := range ch {
//	    c.Observe(f)
//	}
//	fmt.Println(c.FrameRate(), c.BusLoad(), c.PerID())
package stats

import (
	"sync"
	"time"

	lin "github.com/SoundMatt/go-LIN"
)

// perFrameOverheadBits is the fixed LIN 2.x per-frame wire overhead used by
// BusLoad's estimate: break field (>=13 bit-times) + break delimiter (1) +
// sync byte (10, including start/stop bits) + PID byte (10). Each data byte
// and the checksum byte add a further 10 bit-times each (8 data bits + 1
// start + 1 stop), accounted for separately in bitsOnWire.
const perFrameOverheadBits = 13 + 1 + 10 + 10

// bitsPerByteOnWire is the UART framing cost of one byte on the LIN wire
// (1 start bit + 8 data bits + 1 stop bit).
const bitsPerByteOnWire = 10

// Collector accumulates frame statistics. The zero value is not usable;
// call New. A Collector is safe for concurrent use.
type Collector struct {
	mu         sync.Mutex
	baudBPS    float64
	start      time.Time
	totalCount uint64
	totalBits  uint64
	perID      map[uint8]uint64
}

// New creates a Collector. baudKbps is the LIN bus speed in kbit/s, used only
// by BusLoad; pass 0 if unknown (BusLoad then always returns 0).
//
//fusa:req REQ-STATS-001
func New(baudKbps float64) *Collector {
	return &Collector{
		baudBPS: baudKbps * 1000,
		start:   time.Now(),
		perID:   make(map[uint8]uint64),
	}
}

// Observe records one received frame. It is cheap enough to call from a hot
// receive loop.
//
//fusa:req REQ-STATS-002
func (c *Collector) Observe(f lin.Frame) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.totalCount++
	c.perID[f.ID]++
	c.totalBits += bitsOnWire(len(f.Data))
}

// bitsOnWire estimates the number of bit-times a LIN frame with the given
// data length occupies on the wire: fixed break+sync+PID overhead, plus one
// UART byte-time per data byte, plus one for the checksum byte. This is an
// estimate (LIN 2.x allows some flexibility in break length and inter-byte
// spacing); it is intended for observability, not timing-accurate
// simulation.
func bitsOnWire(dataLen int) uint64 {
	return uint64(perFrameOverheadBits + (dataLen+1)*bitsPerByteOnWire)
}

// FrameRate returns the mean number of frames observed per second since New
// or the last Reset.
//
//fusa:req REQ-STATS-003
func (c *Collector) FrameRate() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	elapsed := time.Since(c.start).Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(c.totalCount) / elapsed
}

// BusLoad returns the estimated percentage (0-100+) of theoretical bus
// bandwidth consumed by observed frames since New or the last Reset. It
// returns 0 if the Collector was created with baudKbps <= 0 or no time has
// elapsed.
//
//fusa:req REQ-STATS-004
func (c *Collector) BusLoad() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.baudBPS <= 0 {
		return 0
	}
	elapsed := time.Since(c.start).Seconds()
	if elapsed <= 0 {
		return 0
	}
	usedBPS := float64(c.totalBits) / elapsed
	return usedBPS / c.baudBPS * 100
}

// PerID returns a defensive copy of the per-frame-ID observation counters.
//
//fusa:req REQ-STATS-005
func (c *Collector) PerID() map[uint8]uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[uint8]uint64, len(c.perID))
	for k, v := range c.perID {
		out[k] = v
	}
	return out
}

// TotalCount returns the total number of frames observed since New or the
// last Reset.
//
//fusa:req REQ-STATS-006
func (c *Collector) TotalCount() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.totalCount
}

// Reset clears all counters and restarts the rate-measurement window.
//
//fusa:req REQ-STATS-007
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.start = time.Now()
	c.totalCount = 0
	c.totalBits = 0
	c.perID = make(map[uint8]uint64)
}

// Watch subscribes ch's frames into c until ch closes. It is a convenience
// wrapper around Observe for callers that just want to point a Collector at
// a lin.Bus subscription; it runs until ch closes and does not return until
// then, so callers typically invoke it in its own goroutine:
//
//	ch, _ := bus.Subscribe(nil)
//	go c.Watch(ch)
//
//fusa:req REQ-STATS-008
func (c *Collector) Watch(ch <-chan lin.Frame) {
	for f := range ch {
		c.Observe(f)
	}
}
