// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package master implements the LIN master node.
//
// The master drives the schedule table: it transmits break+sync+PID for
// each slot in turn, collects slave responses, and enforces inter-frame
// slot timing. In a real system the master owns the bit-rate clock and
// is the only node that may transmit headers.
//
// Usage:
//
//	bus, _ := virtual.New()
//	n := master.New(bus)
//	n.SetSchedule([]lin.ScheduleEntry{
//	    {ID: 0x10, DelayMs: 10},
//	    {ID: 0x20, DelayMs: 20},
//	})
//	n.Run(ctx)
//
//fusa:req REQ-MASTER-001
//fusa:req REQ-MASTER-002
//fusa:req REQ-MASTER-003
//fusa:req REQ-MASTER-004
//fusa:req REQ-MASTER-005
//fusa:req REQ-MASTER-006
//fusa:req REQ-MASTER-007
//fusa:req REQ-MASTER-008
//fusa:req REQ-MASTER-009
//fusa:req REQ-MASTER-010
//fusa:req REQ-MASTER-011
//fusa:req REQ-MASTER-012
//fusa:req REQ-MASTER-013
package master

import (
	"context"
	"errors"
	"fmt"
	"time"

	lin "github.com/SoundMatt/go-LIN"
)

// Node is a LIN master node. It drives a schedule table over a MasterBus.
//
//fusa:req REQ-MASTER-001
type Node struct {
	bus      lin.MasterBus
	schedule []lin.ScheduleEntry
	onFrame  func(lin.Frame)
	onError  func(error)
}

// New creates a LIN master node backed by bus.
//
//fusa:req REQ-MASTER-001
func New(bus lin.MasterBus) *Node {
	return &Node{bus: bus}
}

// SetSchedule replaces the active schedule table. It validates all entries
// before storing a defensive copy. It is safe to call between Run
// invocations but must not be called concurrently with Run.
//
//fusa:req REQ-MASTER-010
//fusa:req REQ-MASTER-011
//fusa:req REQ-MASTER-012
func (n *Node) SetSchedule(entries []lin.ScheduleEntry) error {
	if err := validateSchedule(entries); err != nil {
		return err
	}
	cp := make([]lin.ScheduleEntry, len(entries))
	copy(cp, entries)
	n.schedule = cp
	return nil
}

// OnFrame registers a callback invoked for every successfully received frame.
// The callback is called synchronously from Run; it must not block.
//
//fusa:req REQ-MASTER-006
func (n *Node) OnFrame(fn func(lin.Frame)) {
	n.onFrame = fn
}

// OnError registers a callback invoked when a slot produces an error
// (e.g., no slave response). The callback is called synchronously from Run.
//
//fusa:req REQ-MASTER-007
func (n *Node) OnError(fn func(error)) {
	n.onError = fn
}

// SendHeader triggers a single frame exchange for id outside of the normal
// schedule. The frame is broadcast to all subscribers on the bus.
//
//fusa:req REQ-MASTER-002
func (n *Node) SendHeader(ctx context.Context, id uint8) (lin.Frame, error) {
	return n.bus.SendHeader(ctx, id)
}

// Run executes the schedule table repeatedly until ctx is cancelled.
// Each slot transmits a header, waits for a slave response, then sleeps
// for the slot's configured delay. Per-slot errors invoke OnError but do
// not abort the schedule.
//
//fusa:req REQ-MASTER-003
//fusa:req REQ-MASTER-004
//fusa:req REQ-MASTER-005
//fusa:req REQ-MASTER-006
//fusa:req REQ-MASTER-007
//fusa:req REQ-MASTER-008
//fusa:req REQ-MASTER-009
//fusa:req REQ-MASTER-013
func (n *Node) Run(ctx context.Context) error {
	if len(n.schedule) == 0 {
		return errors.New("master: schedule is empty")
	}
	for {
		for _, slot := range n.schedule {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			f, err := n.bus.SendHeader(ctx, slot.ID)
			if err != nil {
				if n.onError != nil {
					n.onError(fmt.Errorf("master: slot 0x%02X: %w", slot.ID, err))
				}
			} else {
				if n.onFrame != nil {
					n.onFrame(f)
				}
			}

			if slot.DelayMs > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(slot.DelayMs) * time.Millisecond):
				}
			}
		}
	}
}

// validateSchedule checks that every schedule entry has a valid frame ID.
//
//fusa:req REQ-MASTER-010
//fusa:req REQ-MASTER-011
func validateSchedule(entries []lin.ScheduleEntry) error {
	if len(entries) == 0 {
		return errors.New("master: schedule must have at least one entry")
	}
	for i, e := range entries {
		if e.ID > lin.MaxID {
			return fmt.Errorf("master: schedule entry %d: ID 0x%02X exceeds maximum 0x%02X", i, e.ID, lin.MaxID)
		}
	}
	return nil
}
