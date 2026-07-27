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

// ── REQ-MASTER-001: New returns non-nil Node ─────────────────────────────────

//fusa:test REQ-MASTER-001

func TestNew(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	n := master.New(bus)
	if n == nil {
		t.Fatal("New returned nil")
	}
}

// ── REQ-MASTER-002: SendHeader delegates to bus ───────────────────────────────

//fusa:test REQ-MASTER-002

func TestSendHeader_delegates(t *testing.T) {
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

// ── REQ-MASTER-003 to REQ-MASTER-005: schedule ordering and timing ───────────

//fusa:test REQ-MASTER-003
//fusa:test REQ-MASTER-004
//fusa:test REQ-MASTER-005

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
	ch, _ := bus.Subscribe([]lin.Filter{{All: true}})

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

// ── REQ-MASTER-006: OnFrame callback ─────────────────────────────────────────

//fusa:test REQ-MASTER-006

func TestRun_onFrame(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	_ = bus.Publish(0x10, []byte{0xAB})
	n := master.New(bus)
	_ = n.SetSchedule([]lin.ScheduleEntry{{ID: 0x10, DelayMs: 0}})

	frames := make(chan lin.Frame, 4)
	n.OnFrame(func(f lin.Frame) {
		select {
		case frames <- f:
		default:
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- n.Run(ctx) }()

	select {
	case f := <-frames:
		if f.ID != 0x10 {
			t.Errorf("OnFrame: got ID 0x%02X, want 0x10", f.ID)
		}
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out waiting for OnFrame callback")
	}
	cancel()
	<-done
}

// ── REQ-MASTER-007: OnError callback ─────────────────────────────────────────

//fusa:test REQ-MASTER-007

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

// ── REQ-MASTER-008: context cancellation ─────────────────────────────────────

//fusa:test REQ-MASTER-008

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

// ── REQ-MASTER-009: empty schedule ───────────────────────────────────────────

//fusa:test REQ-MASTER-009

func TestRun_emptySchedule(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	err := n.Run(context.Background())
	if err == nil {
		t.Error("expected error running empty schedule")
	}
}

// ── REQ-MASTER-010: SetSchedule accepts an empty schedule ────────────────────
// An empty slice is valid and disables scheduled transmission, matching the
// lin.MasterBus.SetSchedule contract. Running an empty schedule is still
// independently rejected by Run (REQ-MASTER-009).

//fusa:test REQ-MASTER-010

func TestSetSchedule_empty(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	if err := n.SetSchedule(nil); err != nil {
		t.Errorf("SetSchedule(nil): %v, want nil (empty schedule disables scheduled transmission)", err)
	}
	if err := n.SetSchedule([]lin.ScheduleEntry{}); err != nil {
		t.Errorf("SetSchedule([]): %v, want nil (empty schedule disables scheduled transmission)", err)
	}
	// Run still rejects the empty schedule it was just given (REQ-MASTER-009).
	if err := n.Run(context.Background()); err == nil {
		t.Error("Run with an empty schedule: expected error, got nil")
	}
}

// ── REQ-MASTER-011: SetSchedule rejects invalid ID ───────────────────────────

//fusa:test REQ-MASTER-011

func TestSetSchedule_invalidID(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	err := n.SetSchedule([]lin.ScheduleEntry{{ID: 0x40, DelayMs: 0}})
	if err == nil {
		t.Error("expected error for ID > MaxID")
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

// ── REQ-MASTER-012: defensive copy ───────────────────────────────────────────

//fusa:test REQ-MASTER-012

func TestSetSchedule_defensiveCopy(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	entries := []lin.ScheduleEntry{{ID: 0x10, DelayMs: 0}}
	n := master.New(bus)
	_ = n.SetSchedule(entries)

	// mutate the original slice
	entries[0].ID = 0x3F

	// Run should still use the original ID (0x10), not 0x3F
	_ = bus.Publish(0x10, []byte{0xAA})
	ch, _ := bus.Subscribe([]lin.Filter{{ID: 0x10}})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- n.Run(ctx) }()

	select {
	case f := <-ch:
		if f.ID != 0x10 {
			t.Errorf("defensive copy not taken: got ID 0x%02X, want 0x10", f.ID)
		}
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out")
	}
	cancel()
	<-done
}

// ── REQ-MASTER-013: continue after per-slot errors ───────────────────────────

//fusa:test REQ-MASTER-013

func TestRun_continuesAfterError(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	// slot 0 has no response (will error), slot 1 has a response
	_ = bus.Publish(0x20, []byte{0x02})

	n := master.New(bus)
	_ = n.SetSchedule([]lin.ScheduleEntry{
		{ID: 0x10, DelayMs: 0}, // no response
		{ID: 0x20, DelayMs: 0}, // has response
	})

	ch, _ := bus.Subscribe([]lin.Filter{{ID: 0x20}})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- n.Run(ctx) }()

	select {
	case f := <-ch:
		if f.ID != 0x20 {
			t.Errorf("expected frame 0x20, got 0x%02X", f.ID)
		}
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timed out — Run may have aborted on error")
	}
	cancel()
	<-done
}

// ── REQ-MASTER-014: Diagnostics drives a request/response exchange ───────────

//fusa:test REQ-MASTER-014

func TestDiagnostics_requestResponseRoundTrip(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	// The diagnostic target's response, pre-registered on 0x3D as any slave
	// would via slave.Node.SetResponse.
	resp := lin.SlaveResponseFrame{NAD: 0x01, RSID: 0xF2, Data: []byte{0xAA}}
	respFrame, err := resp.ToFrame()
	if err != nil {
		t.Fatalf("resp.ToFrame: %v", err)
	}
	if err := bus.Publish(lin.LINDiagResponseID, respFrame.Data); err != nil {
		t.Fatalf("Publish(0x3D): %v", err)
	}

	ch, _ := bus.Subscribe([]lin.Filter{{ID: lin.LINDiagRequestID}})

	n := master.New(bus)
	req := lin.MasterRequestFrame{NAD: 0x01, SID: 0xB2, Data: []byte{0x01, 0x02}}
	got, err := n.Diagnostics(context.Background(), req)
	if err != nil {
		t.Fatalf("Diagnostics: %v", err)
	}
	if got.NAD != resp.NAD || got.RSID != resp.RSID || string(got.Data) != string(resp.Data) {
		t.Errorf("Diagnostics response = %+v, want %+v", got, resp)
	}

	// The request itself must actually have been transmitted on 0x3C.
	select {
	case sent := <-ch:
		reqFrame, _ := req.ToFrame()
		if string(sent.Data) != string(reqFrame.Data) {
			t.Errorf("transmitted request data = % X, want % X", sent.Data, reqFrame.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the request frame to be broadcast")
	}
}

func TestDiagnostics_noResponseRegistered(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()

	n := master.New(bus)
	req := lin.MasterRequestFrame{NAD: 0x01, SID: 0xB2}
	if _, err := n.Diagnostics(context.Background(), req); err == nil {
		t.Error("Diagnostics with no registered 0x3D response: expected error, got nil")
	}
}

// ── REQ-MASTER-015/016: sporadic frame slots ──────────────────────────────────

//fusa:test REQ-MASTER-015
//fusa:test REQ-MASTER-016

// sporadicSlotID is an arbitrary schedule-table placeholder ID: any valid
// frame ID works, since SetSporadicGroup treats it purely as a key.
const sporadicSlotID = 0x30

func TestSporadic_selectsHighestPriorityPending(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	_ = bus.Publish(0x11, []byte{0x01})
	_ = bus.Publish(0x12, []byte{0x02})

	n := master.New(bus)
	if err := n.SetSporadicGroup(sporadicSlotID, []uint8{0x11, 0x12}); err != nil {
		t.Fatalf("SetSporadicGroup: %v", err)
	}
	n.SetPending(0x12)
	n.SetPending(0x11) // both pending; 0x11 has higher priority (listed first)

	ch, _ := bus.Subscribe([]lin.Filter{{All: true}})
	_ = n.SetSchedule([]lin.ScheduleEntry{{ID: sporadicSlotID, DelayMs: 0}})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- n.Run(ctx) }()

	select {
	case f := <-ch:
		if f.ID != 0x11 {
			t.Errorf("transmitted frame ID = 0x%02X, want 0x11 (higher priority)", f.ID)
		}
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("timed out waiting for the sporadic frame")
	}
	cancel()
	<-done
}

func TestSporadic_skipsSlotWhenNothingPending(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	_ = bus.Publish(0x11, []byte{0x01})

	n := master.New(bus)
	if err := n.SetSporadicGroup(sporadicSlotID, []uint8{0x11}); err != nil {
		t.Fatalf("SetSporadicGroup: %v", err)
	}
	// no SetPending call: the slot must be skipped, not error.

	var errs int
	n.OnError(func(error) { errs++ })
	ch, _ := bus.Subscribe([]lin.Filter{{All: true}})
	_ = n.SetSchedule([]lin.ScheduleEntry{{ID: sporadicSlotID, DelayMs: 0}})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_ = n.Run(ctx)

	select {
	case f := <-ch:
		t.Errorf("expected no frame transmitted for an empty sporadic slot, got 0x%02X", f.ID)
	default:
	}
	if errs != 0 {
		t.Errorf("OnError called %d time(s) for a correctly-skipped sporadic slot, want 0", errs)
	}
}

func TestSporadic_pendingClearedAfterSelection(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	_ = bus.Publish(0x11, []byte{0x01})

	n := master.New(bus)
	_ = n.SetSporadicGroup(sporadicSlotID, []uint8{0x11})
	n.SetPending(0x11)
	_ = n.SetSchedule([]lin.ScheduleEntry{{ID: sporadicSlotID, DelayMs: 0}})

	ch, _ := bus.Subscribe([]lin.Filter{{All: true}})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_ = n.Run(ctx)

	// The first pass must transmit exactly one frame (pending consumed),
	// not one per schedule iteration.
	count := 0
drain:
	for {
		select {
		case <-ch:
			count++
		default:
			break drain
		}
	}
	if count != 1 {
		t.Errorf("frames transmitted = %d, want exactly 1 (pending flag must clear after selection)", count)
	}
}

func TestSetSporadicGroup_invalidID(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	n := master.New(bus)
	if err := n.SetSporadicGroup(0x40, []uint8{0x11}); err == nil {
		t.Error("SetSporadicGroup(slotID > MaxID): expected error, got nil")
	}
	if err := n.SetSporadicGroup(sporadicSlotID, []uint8{0x40}); err == nil {
		t.Error("SetSporadicGroup(candidate > MaxID): expected error, got nil")
	}
}

func TestSetSporadicGroup_emptyRemovesRegistration(t *testing.T) {
	bus, _ := virtual.New()
	defer bus.Close()
	n := master.New(bus)
	_ = n.SetSporadicGroup(sporadicSlotID, []uint8{0x11})
	if err := n.SetSporadicGroup(sporadicSlotID, nil); err != nil {
		t.Fatalf("SetSporadicGroup(nil): %v", err)
	}

	// sporadicSlotID must now behave as an ordinary slot: SendHeader is
	// attempted directly on it and fails with ErrNoResponse since nothing
	// registered a response for that literal ID.
	_ = bus.Publish(sporadicSlotID, nil) // ensure no stray response is registered
	if _, err := n.SendHeader(context.Background(), sporadicSlotID); !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("SendHeader after clearing sporadic group: err = %v, want ErrNoResponse", err)
	}
}
