// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package master_test

import (
	"context"
	"errors"
	"testing"
	"time"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/master"
	"github.com/SoundMatt/go-LIN/virtual"
)

//fusa:test REQ-MASTER-001
//fusa:test REQ-MASTER-002
//fusa:test REQ-MASTER-003

func TestNew(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	n := master.New(bus)
	if n == nil {
		t.Fatal("New returned nil")
	}
}

func TestSendHeader(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	_ = bus.Publish(0x10, []byte{0x01, 0x02})
	n := master.New(bus)

	f, err := n.SendHeader(context.Background(), 0x10)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.ID != 0x10 {
		t.Errorf("frame ID = 0x%02X, want 0x10", f.ID)
	}
}

func TestSendHeader_noResponse(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	_, err := n.SendHeader(context.Background(), 0x10)
	if !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("expected ErrNoResponse, got %v", err)
	}
}

func TestSetSchedule_valid(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	err := n.SetSchedule([]lin.ScheduleEntry{
		{ID: 0x10, DelayMs: 0},
		{ID: 0x20, DelayMs: 0},
	})
	if err != nil {
		t.Fatalf("SetSchedule: %v", err)
	}
}

func TestSetSchedule_empty(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	if err := n.SetSchedule(nil); err == nil {
		t.Error("expected error for empty schedule")
	}
}

func TestSetSchedule_invalidID(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	err := n.SetSchedule([]lin.ScheduleEntry{{ID: 0x40, DelayMs: 0}})
	if err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

func TestRun_executesSchedule(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	_ = bus.Publish(0x10, []byte{0x01})
	_ = bus.Publish(0x20, []byte{0x02})

	n := master.New(bus)
	_ = n.SetSchedule([]lin.ScheduleEntry{
		{ID: 0x10, DelayMs: 0},
		{ID: 0x20, DelayMs: 0},
	})

	// Use subscriber channel (drop-on-full) to avoid callback deadlock
	ch, _ := bus.Subscribe(lin.Filter{All: true})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- n.Run(ctx) }()

	got := 0
	for got < 2 {
		select {
		case <-ch:
			got++
		case <-time.After(5 * time.Second):
			cancel()
			t.Fatal("timed out waiting for frames from master schedule")
		}
	}
	cancel()
	<-done
}

func TestRun_emptySchedule(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	err := n.Run(context.Background())
	if err == nil {
		t.Error("expected error running empty schedule")
	}
}

func TestRun_onError(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	// no slave response registered — master should invoke onError
	n := master.New(bus)
	_ = n.SetSchedule([]lin.ScheduleEntry{{ID: 0x10, DelayMs: 0}})

	errs := make(chan error, 64)
	n.OnError(func(err error) {
		select {
		case errs <- err:
		default:
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- n.Run(ctx) }()

	select {
	case <-errs:
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for onError callback")
	}
	cancel()
	<-done
}

func TestRun_cancelledContext(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	_ = bus.Publish(0x10, []byte{0x01})
	n := master.New(bus)
	_ = n.SetSchedule([]lin.ScheduleEntry{{ID: 0x10, DelayMs: 0}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := n.Run(ctx)
	if err == nil {
		t.Error("expected context error from Run")
	}
}
