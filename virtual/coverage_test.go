// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package virtual_test

import (
	"context"
	"errors"
	"testing"
	"time"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

func TestSetSchedule(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := bus.SetSchedule([]lin.ScheduleEntry{{ID: 0x10, DelayMs: 5}}); err != nil {
		t.Fatalf("SetSchedule: %v", err)
	}
	if err := bus.SetSchedule(nil); err != nil { // empty disables
		t.Fatalf("SetSchedule(nil): %v", err)
	}
	bus.Close()
	if err := bus.SetSchedule([]lin.ScheduleEntry{{ID: 1}}); !errors.Is(err, lin.ErrClosed) {
		t.Errorf("SetSchedule after Close = %v, want ErrClosed", err)
	}
}

func TestPublishClassic(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	if err := bus.PublishClassic(0x10, []byte{0x01, 0x02}); err != nil {
		t.Fatalf("PublishClassic: %v", err)
	}
	f, err := bus.SendHeader(context.Background(), 0x10)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.ChecksumType != lin.ClassicChecksum {
		t.Errorf("ChecksumType = %v, want ClassicChecksum", f.ChecksumType)
	}
	// nil removes the registration.
	if err := bus.PublishClassic(0x10, nil); err != nil {
		t.Fatalf("PublishClassic(nil): %v", err)
	}
	if _, err := bus.SendHeader(context.Background(), 0x10); !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("after removal SendHeader = %v, want ErrNoResponse", err)
	}
	// Out-of-range ID is rejected.
	if err := bus.PublishClassic(0x40, []byte{1}); err == nil {
		t.Error("PublishClassic(0x40) = nil, want range error")
	}
}

func TestPublishClassic_afterClose(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	bus.Close()
	if err := bus.PublishClassic(0x10, []byte{1}); !errors.Is(err, lin.ErrClosed) {
		t.Errorf("PublishClassic after Close = %v, want ErrClosed", err)
	}
}

func TestSubscribe_afterClose(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	bus.Close()
	if _, err := bus.Subscribe(nil); !errors.Is(err, lin.ErrClosed) {
		t.Errorf("Subscribe after Close = %v, want ErrClosed", err)
	}
}

func TestSubscribe_filterMatching(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	ch, err := bus.Subscribe([]lin.Filter{{ID: 0x10}}) // only 0x10
	if err != nil {
		t.Fatal(err)
	}
	bus.Publish(0x10, []byte{0xAA})
	bus.Publish(0x20, []byte{0xBB})
	if _, err := bus.SendHeader(context.Background(), 0x20); err != nil {
		t.Fatal(err)
	}
	if _, err := bus.SendHeader(context.Background(), 0x10); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-ch:
		if f.ID != 0x10 {
			t.Errorf("filtered subscriber received ID 0x%02X, want only 0x10", f.ID)
		}
	case <-time.After(time.Second):
		t.Fatal("expected a frame for 0x10")
	}
}

func TestCloseWithDrain(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	ch, err := bus.Subscribe(nil)
	if err != nil {
		t.Fatal(err)
	}
	bus.Publish(0x10, []byte{0x01})
	if _, err := bus.SendHeader(context.Background(), 0x10); err != nil {
		t.Fatal(err)
	}
	// Drain: consume the pending frame concurrently so the channel empties.
	go func() {
		<-ch
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := bus.CloseWithDrain(ctx); err != nil {
		t.Fatalf("CloseWithDrain: %v", err)
	}
}

func TestCloseWithDrain_ctxCancel(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	ch, err := bus.Subscribe(nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = ch // never drained → forces the ctx.Done() branch
	bus.Publish(0x10, []byte{0x01})
	if _, err := bus.SendHeader(context.Background(), 0x10); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := bus.CloseWithDrain(ctx); err != nil {
		t.Fatalf("CloseWithDrain (ctx cancel path): %v", err)
	}
}

func TestHealthAndMetrics(t *testing.T) {
	bus, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	if h := bus.Health(); h.Status != lin.HealthOK {
		t.Errorf("Health = %v, want HealthOK", h.Status)
	}
	bus.Publish(0x10, []byte{0x01, 0x02})
	if _, err := bus.SendHeader(context.Background(), 0x10); err != nil {
		t.Fatal(err)
	}
	if m := bus.Metrics(); m.WriteCount == 0 {
		t.Error("Metrics.WriteCount = 0 after a SendHeader")
	}
	bus.Close()
	if h := bus.Health(); h.Status != lin.HealthDown {
		t.Errorf("Health after Close = %v, want HealthDown", h.Status)
	}
}
