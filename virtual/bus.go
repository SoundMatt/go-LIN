// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package virtual provides an in-process LIN bus for development and testing.
// It has zero dependencies and works on all platforms.
//
// The virtual bus simulates both master and slave behaviour in a single
// process. Call Publish to register slave responses; call SendHeader to
// trigger frame exchanges as a master node. All observed frames are
// delivered to matching subscribers.
//
//fusa:req REQ-VIRT-001
//fusa:req REQ-VIRT-002
//fusa:req REQ-VIRT-003
//fusa:req REQ-VIRT-004
//fusa:req REQ-VIRT-005
//fusa:req REQ-VIRT-006
//fusa:req REQ-VIRT-007
//fusa:req REQ-VIRT-008
//fusa:req REQ-VIRT-009
//fusa:req REQ-VIRT-010
//fusa:req REQ-VIRT-011
//fusa:req REQ-VIRT-012
//fusa:req REQ-VIRT-013
//fusa:req REQ-VIRT-014
//fusa:req REQ-VIRT-015
//fusa:req REQ-VIRT-016
//fusa:req REQ-VIRT-017
//fusa:req REQ-VIRT-018
//fusa:req REQ-VIRT-019
package virtual

import (
	"context"
	"fmt"
	"sync"

	lin "github.com/SoundMatt/go-LIN"
)

const defaultChanSize = 64

// Bus is an in-process LIN bus. Multiple goroutines may call Publish,
// SendHeader, and Subscribe concurrently. The zero value is not usable;
// call New.
//
// Bus implements lin.MasterBus.
type Bus struct {
	mu        sync.RWMutex
	responses map[uint8]responseEntry
	subs      []*subscription
	closed    bool
}

type responseEntry struct {
	data         []byte
	checksumType lin.ChecksumType
}

type subscription struct {
	filters []lin.Filter
	ch      chan lin.Frame
}

var errClosed = fmt.Errorf("lin/virtual: bus is closed")

// New creates an in-process virtual LIN bus.
//
//fusa:req REQ-VIRT-001
func New() (*Bus, error) {
	return &Bus{
		responses: make(map[uint8]responseEntry),
	}, nil
}

// Publish registers a response payload for the given frame ID.
// Publish(id, nil) removes the registration.
// Publish after Close returns an error.
//
//fusa:req REQ-VIRT-002
//fusa:req REQ-VIRT-003
//fusa:req REQ-VIRT-004
//fusa:req REQ-VIRT-005
//fusa:req REQ-VIRT-019
func (b *Bus) Publish(id uint8, data []byte) error {
	if id > lin.MaxID {
		return fmt.Errorf("lin/virtual: frame ID 0x%02X exceeds maximum 0x%02X", id, lin.MaxID)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errClosed
	}
	if data == nil {
		delete(b.responses, id)
		return nil
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	b.responses[id] = responseEntry{data: cp, checksumType: lin.EnhancedChecksum}
	return nil
}

// PublishClassic is like Publish but applies the classic (LIN 1.x) checksum.
func (b *Bus) PublishClassic(id uint8, data []byte) error {
	if id > lin.MaxID {
		return fmt.Errorf("lin/virtual: frame ID 0x%02X exceeds maximum 0x%02X", id, lin.MaxID)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return errClosed
	}
	if data == nil {
		delete(b.responses, id)
		return nil
	}
	cp := make([]byte, len(data))
	copy(cp, data)
	b.responses[id] = responseEntry{data: cp, checksumType: lin.ClassicChecksum}
	return nil
}

// SendHeader drives a frame exchange for the given ID.
// It looks up any registered slave response, synthesises the Frame with the
// correct PID and checksum, broadcasts it to all matching subscribers, and
// returns the Frame.
// Returns lin.ErrNoResponse when no response is registered.
// Returns an error when called after Close.
//
//fusa:req REQ-VIRT-006
//fusa:req REQ-VIRT-007
//fusa:req REQ-VIRT-008
//fusa:req REQ-VIRT-009
//fusa:req REQ-VIRT-010
//fusa:req REQ-VIRT-017
//fusa:req REQ-VIRT-018
func (b *Bus) SendHeader(ctx context.Context, id uint8) (lin.Frame, error) {
	if id > lin.MaxID {
		return lin.Frame{}, fmt.Errorf("lin/virtual: frame ID 0x%02X exceeds maximum 0x%02X", id, lin.MaxID)
	}

	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return lin.Frame{}, errClosed
	}
	entry, ok := b.responses[id]
	b.mu.RUnlock()

	if !ok {
		return lin.Frame{}, lin.ErrNoResponse
	}

	pid := lin.ProtectID(id)
	data := make([]byte, len(entry.data))
	copy(data, entry.data)
	cs := lin.CalcChecksum(pid, data, entry.checksumType)

	f := lin.Frame{
		ID:           id,
		Data:         data,
		Checksum:     cs,
		ChecksumType: entry.checksumType,
	}

	b.broadcast(f)
	return f, nil
}

// Subscribe returns a channel that delivers frames matching any of the
// supplied filters. Pass nil to receive all frames.
// opts configures channel delivery (depth, back-pressure).
//
//fusa:req REQ-VIRT-011
//fusa:req REQ-VIRT-012
//fusa:req REQ-VIRT-013
//fusa:req REQ-VIRT-014
func (b *Bus) Subscribe(filters []lin.Filter, opts ...lin.SubscriberOption) (<-chan lin.Frame, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, errClosed
	}
	s := &subscription{
		filters: filters,
		ch:      make(chan lin.Frame, defaultChanSize),
	}
	b.subs = append(b.subs, s)
	return s.ch, nil
}

// Close releases all resources and closes all subscriber channels.
//
//fusa:req REQ-VIRT-015
//fusa:req REQ-VIRT-016
func (b *Bus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	for _, s := range b.subs {
		close(s.ch)
	}
	b.subs = nil
	return nil
}

func (b *Bus) broadcast(f lin.Frame) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, s := range b.subs {
		if matchesAny(s.filters, f) {
			select {
			case s.ch <- f:
			default:
				// drop on full channel — REQ-VIRT-013
			}
		}
	}
}

func matchesAny(filters []lin.Filter, f lin.Frame) bool {
	if len(filters) == 0 {
		return true
	}
	for _, flt := range filters {
		if flt.Matches(f) {
			return true
		}
	}
	return false
}
