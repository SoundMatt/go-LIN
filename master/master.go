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
//fusa:req REQ-MASTER-015
//fusa:req REQ-MASTER-016
package master

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

	mu       sync.Mutex
	pending  map[uint8]bool
	sporadic map[uint8][]uint8
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
// An empty slice is valid and disables scheduled transmission, matching the
// lin.MasterBus.SetSchedule contract (§8.3). Calling Run with an empty
// schedule still fails fast (REQ-MASTER-009); SetSchedule itself does not
// duplicate that check.
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

// Diagnostics drives one LIN diagnostic request/response exchange (LIN 2.x
// §4.2.3): it publishes req's wire encoding on lin.LINDiagRequestID (0x3C)
// and triggers its header, then triggers lin.LINDiagResponseID (0x3D) and
// parses the target's response.
//
// If publishing or sending the request fails, Diagnostics returns without
// sending the response header — a diagnostic client must not solicit a
// response the target was never asked for.
//
//fusa:req REQ-MASTER-014
func (n *Node) Diagnostics(ctx context.Context, req lin.MasterRequestFrame) (lin.SlaveResponseFrame, error) {
	f, err := req.ToFrame()
	if err != nil {
		return lin.SlaveResponseFrame{}, fmt.Errorf("master: diagnostics: %w", err)
	}
	if err := n.bus.Publish(f.ID, f.Data); err != nil {
		return lin.SlaveResponseFrame{}, fmt.Errorf("master: diagnostics: publish request: %w", err)
	}
	if _, err := n.bus.SendHeader(ctx, lin.LINDiagRequestID); err != nil {
		return lin.SlaveResponseFrame{}, fmt.Errorf("master: diagnostics: send request header: %w", err)
	}
	resp, err := n.bus.SendHeader(ctx, lin.LINDiagResponseID)
	if err != nil {
		return lin.SlaveResponseFrame{}, fmt.Errorf("master: diagnostics: send response header: %w", err)
	}
	out, err := lin.ParseSlaveResponseFrame(resp)
	if err != nil {
		return lin.SlaveResponseFrame{}, fmt.Errorf("master: diagnostics: parse response: %w", err)
	}
	return out, nil
}

// SetSporadicGroup declares slotID — a placeholder frame ID used as a
// lin.ScheduleEntry.ID — as a sporadic frame slot per LIN 2.x §2.3.2.4:
// candidates lists the real frame IDs sharing that slot, in priority order
// (index 0 = highest priority). When Run reaches a schedule slot whose ID
// equals slotID, it transmits the header of the highest-priority candidate
// that has a pending update (SetPending) and clears that candidate's
// pending flag; if no candidate is pending, the slot is skipped — no header
// is transmitted and neither OnFrame nor OnError is invoked for that pass.
//
// Passing a nil or empty candidates slice removes slotID's sporadic-group
// registration, so schedule slots using that ID resolve as an ordinary
// (non-sporadic) slot again.
//
//fusa:req REQ-MASTER-016
func (n *Node) SetSporadicGroup(slotID uint8, candidates []uint8) error {
	if slotID > lin.MaxID {
		return fmt.Errorf("master: sporadic slot ID 0x%02X exceeds maximum 0x%02X", slotID, lin.MaxID)
	}
	for i, id := range candidates {
		if id > lin.MaxID {
			return fmt.Errorf("master: sporadic candidate %d: ID 0x%02X exceeds maximum 0x%02X", i, id, lin.MaxID)
		}
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if len(candidates) == 0 {
		delete(n.sporadic, slotID)
		return nil
	}
	if n.sporadic == nil {
		n.sporadic = make(map[uint8][]uint8)
	}
	n.sporadic[slotID] = append([]uint8(nil), candidates...)
	return nil
}

// SetPending marks id's data as changed by the application, making it
// eligible for selection the next time Run reaches a sporadic slot
// (SetSporadicGroup) whose candidates include id. It has no effect on
// ordinary (non-sporadic) schedule slots.
//
//fusa:req REQ-MASTER-015
func (n *Node) SetPending(id uint8) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.pending == nil {
		n.pending = make(map[uint8]bool)
	}
	n.pending[id] = true
}

// selectSporadic reports whether slotID is a registered sporadic group and,
// if so, returns the highest-priority pending candidate — clearing its
// pending flag — or ok=false if the group has no pending candidate.
func (n *Node) selectSporadic(slotID uint8) (id uint8, isSporadic, ok bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	candidates, isSporadic := n.sporadic[slotID]
	if !isSporadic {
		return 0, false, false
	}
	for _, c := range candidates {
		if n.pending[c] {
			delete(n.pending, c)
			return c, true, true
		}
	}
	return 0, true, false
}

// Run executes the schedule table repeatedly until ctx is cancelled.
// Each slot transmits a header, waits for a slave response, then sleeps
// for the slot's configured delay. Per-slot errors invoke OnError but do
// not abort the schedule. A slot registered via SetSporadicGroup with no
// pending candidate is skipped (REQ-MASTER-016).
//
//fusa:req REQ-MASTER-003
//fusa:req REQ-MASTER-004
//fusa:req REQ-MASTER-005
//fusa:req REQ-MASTER-006
//fusa:req REQ-MASTER-007
//fusa:req REQ-MASTER-008
//fusa:req REQ-MASTER-009
//fusa:req REQ-MASTER-013
//fusa:req REQ-MASTER-016
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

			id, isSporadic, pending := n.selectSporadic(slot.ID)
			if !isSporadic {
				id = slot.ID
			}
			if !isSporadic || pending {
				f, err := n.bus.SendHeader(ctx, id)
				if err != nil {
					if n.onError != nil {
						n.onError(fmt.Errorf("master: slot 0x%02X: %w", id, err))
					}
				} else {
					if n.onFrame != nil {
						n.onFrame(f)
					}
				}
			}
			// A sporadic slot with no pending candidate is skipped: no
			// header is transmitted and neither callback fires, per
			// LIN 2.x §2.3.2.4.

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
// An empty slice is valid: it matches lin.MasterBus.SetSchedule's documented
// contract that an empty schedule disables scheduled transmission.
//
//fusa:req REQ-MASTER-010
//fusa:req REQ-MASTER-011
func validateSchedule(entries []lin.ScheduleEntry) error {
	for i, e := range entries {
		if e.ID > lin.MaxID {
			return fmt.Errorf("master: schedule entry %d: ID 0x%02X exceeds maximum 0x%02X", i, e.ID, lin.MaxID)
		}
	}
	return nil
}
