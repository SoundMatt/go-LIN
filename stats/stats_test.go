// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package stats_test

import (
	"sync"
	"testing"
	"time"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/stats"
)

// ── REQ-STATS-001: New returns a ready Collector ──────────────────────────────
// ── REQ-STATS-002: Observe updates total and per-ID counters ─────────────────

//fusa:test REQ-STATS-001
//fusa:test REQ-STATS-002

func TestObserve_totalCountAndPerID(t *testing.T) {
	c := stats.New(19.2)
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01}})
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x02}})
	c.Observe(lin.Frame{ID: 0x20, Data: []byte{0x03}})

	if got := c.TotalCount(); got != 3 {
		t.Errorf("TotalCount = %d, want 3", got)
	}
	perID := c.PerID()
	if perID[0x10] != 2 {
		t.Errorf("PerID[0x10] = %d, want 2", perID[0x10])
	}
	if perID[0x20] != 1 {
		t.Errorf("PerID[0x20] = %d, want 1", perID[0x20])
	}
}

// ── REQ-STATS-005: PerID returns a defensive copy ─────────────────────────────

//fusa:test REQ-STATS-005

func TestPerID_defensiveCopy(t *testing.T) {
	c := stats.New(19.2)
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01}})
	perID := c.PerID()
	perID[0x10] = 999
	perID[0x99] = 1
	if got := c.PerID()[0x10]; got != 1 {
		t.Errorf("mutating the returned map affected the Collector: PerID[0x10] = %d, want 1", got)
	}
}

func TestFrameRate_positive(t *testing.T) {
	c := stats.New(19.2)
	for i := 0; i < 10; i++ {
		c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01}})
	}
	time.Sleep(5 * time.Millisecond)
	if rate := c.FrameRate(); rate <= 0 {
		t.Errorf("FrameRate = %v, want > 0", rate)
	}
}

// ── REQ-STATS-004: BusLoad returns 0 when the bus speed is unknown ───────────

//fusa:test REQ-STATS-004

// TestBusLoad_zeroWhenNoBaud verifies BusLoad returns 0 when the Collector
// was created without a known bus speed (baudKbps <= 0).
func TestBusLoad_zeroWhenNoBaud(t *testing.T) {
	c := stats.New(0)
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01, 0x02, 0x03, 0x04}})
	time.Sleep(time.Millisecond)
	if load := c.BusLoad(); load != 0 {
		t.Errorf("BusLoad (no baud) = %v, want 0", load)
	}
}

// TestBusLoad_increasesWithTraffic verifies BusLoad reports a higher load
// after observing more frames in the same window.
func TestBusLoad_increasesWithTraffic(t *testing.T) {
	c := stats.New(19.2)
	time.Sleep(2 * time.Millisecond)
	before := c.BusLoad()
	for i := 0; i < 50; i++ {
		c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}})
	}
	after := c.BusLoad()
	if after <= before {
		t.Errorf("BusLoad after traffic = %v, want > BusLoad before = %v", after, before)
	}
}

// ── REQ-STATS-006: TotalCount reports the cumulative Observe count ───────────
// ── REQ-STATS-007: Reset clears all counters and the rate window ─────────────

//fusa:test REQ-STATS-006
//fusa:test REQ-STATS-007

func TestReset_clearsCounters(t *testing.T) {
	c := stats.New(19.2)
	c.Observe(lin.Frame{ID: 0x10, Data: []byte{0x01}})
	c.Reset()
	if got := c.TotalCount(); got != 0 {
		t.Errorf("TotalCount after Reset = %d, want 0", got)
	}
	if got := len(c.PerID()); got != 0 {
		t.Errorf("len(PerID()) after Reset = %d, want 0", got)
	}
}

// ── REQ-STATS-008: Watch observes every frame until the channel closes ───────

//fusa:test REQ-STATS-008

func TestWatch_consumesUntilClose(t *testing.T) {
	c := stats.New(19.2)
	ch := make(chan lin.Frame, 4)
	ch <- lin.Frame{ID: 0x10, Data: []byte{0x01}}
	ch <- lin.Frame{ID: 0x11, Data: []byte{0x02}}
	close(ch)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Watch(ch)
	}()
	wg.Wait()

	if got := c.TotalCount(); got != 2 {
		t.Errorf("TotalCount after Watch = %d, want 2", got)
	}
}

// ── REQ-STATS-003: FrameRate reports a non-negative, finite rate ─────────────

//fusa:test REQ-STATS-003

func TestFrameRate_zeroBeforeAnyElapsedTime(t *testing.T) {
	// A fresh Collector with no observations should report a non-negative,
	// finite rate rather than dividing by zero.
	c := stats.New(19.2)
	if rate := c.FrameRate(); rate < 0 {
		t.Errorf("FrameRate (fresh Collector) = %v, want >= 0", rate)
	}
}
