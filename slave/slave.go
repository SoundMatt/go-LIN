// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package slave implements a LIN slave node.
//
// A slave node publishes response frames and subscribes to observe the bus.
// When the master transmits a header for a registered ID, the slave's
// response is sent by the underlying transport.
//
// Usage:
//
//	bus, _ := virtual.New()
//	s := slave.New(bus)
//	s.SetResponse(0x10, []byte{0x01, 0x02, 0x03, 0x04})
//	ch, _ := s.Subscribe([]lin.Filter{{ID: 0x10}})
//
//fusa:req REQ-SLAVE-001
//fusa:req REQ-SLAVE-002
//fusa:req REQ-SLAVE-003
//fusa:req REQ-SLAVE-004
//fusa:req REQ-SLAVE-005
//fusa:req REQ-SLAVE-006
//fusa:req REQ-SLAVE-007
//fusa:req REQ-SLAVE-008
package slave

import (
	"fmt"
	"sync"

	lin "github.com/SoundMatt/go-LIN"
)

// Node is a LIN slave node. It registers response payloads on the bus and
// provides a subscription interface for observing frames.
//
//fusa:req REQ-SLAVE-001
type Node struct {
	mu  sync.RWMutex
	bus lin.Bus
	ids []uint8
}

// New creates a LIN slave node backed by bus.
//
//fusa:req REQ-SLAVE-001
func New(bus lin.Bus) *Node {
	return &Node{bus: bus}
}

// SetResponse registers the response payload for the given frame ID.
// When the master requests this ID, bus.Publish delivers the payload.
// Passing nil removes the response.
// Calling with an existing ID replaces the previous registration.
//
//fusa:req REQ-SLAVE-002
//fusa:req REQ-SLAVE-003
//fusa:req REQ-SLAVE-004
//fusa:req REQ-SLAVE-008
func (n *Node) SetResponse(id uint8, data []byte) error {
	if id > lin.MaxID {
		return fmt.Errorf("slave: frame ID 0x%02X exceeds maximum 0x%02X", id, lin.MaxID)
	}
	if err := n.bus.Publish(id, data); err != nil {
		return fmt.Errorf("slave: Publish: %w", err)
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if data != nil {
		if !containsID(n.ids, id) {
			n.ids = append(n.ids, id)
		}
	} else {
		n.ids = removeID(n.ids, id)
	}
	return nil
}

// Subscribe returns a channel delivering frames that match any of the
// supplied filters. Pass nil to receive all frames.
//
//fusa:req REQ-SLAVE-006
func (n *Node) Subscribe(filters []lin.Filter, opts ...lin.SubscriberOption) (<-chan lin.Frame, error) {
	return n.bus.Subscribe(filters, opts...)
}

// RegisteredIDs returns the frame IDs for which this slave has a registered response.
// Returns an empty (non-nil) slice when no responses are registered.
//
//fusa:req REQ-SLAVE-005
//fusa:req REQ-SLAVE-007
func (n *Node) RegisteredIDs() []uint8 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	out := make([]uint8, len(n.ids))
	copy(out, n.ids)
	return out
}

func containsID(ids []uint8, id uint8) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

func removeID(ids []uint8, id uint8) []uint8 {
	out := ids[:0]
	for _, v := range ids {
		if v != id {
			out = append(out, v)
		}
	}
	return out
}
