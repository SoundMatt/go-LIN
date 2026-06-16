// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package lin provides the core types and Bus interface for LIN bus
// (Local Interconnect Network) communication. Implementations are in
// sub-packages and are swappable without changing application code.
//
// LIN is a serial, single-wire, master-slave bus defined by the LIN
// Consortium (LIN 2.x) for low-bandwidth automotive subsystems (seat
// position, mirror control, sun roof, HVAC, etc.).
//
// Quickstart:
//
//	import (
//	    lin "github.com/SoundMatt/go-LIN"
//	    "github.com/SoundMatt/go-LIN/virtual"
//	)
//
//	bus, _ := virtual.New()
//	defer bus.Close()
//
//	ch, _ := bus.Subscribe([]lin.Filter{{ID: 0x10}})
//	bus.Publish(0x10, []byte{0x01, 0x02, 0x03, 0x04})
//	frame := <-ch
package lin

import (
	"context"
	"errors"
	"fmt"

	relay "github.com/SoundMatt/RELAY"
)

// LINMaxDataLen is the maximum number of data bytes in a LIN frame payload.
const LINMaxDataLen = 8

// LINMaxID is the maximum raw LIN frame identifier (6 bits).
const LINMaxID = 0x3F

// LINDiagRequestID is the master request diagnostic frame ID (0x3C).
const LINDiagRequestID = 0x3C

// LINDiagResponseID is the slave response diagnostic frame ID (0x3D).
const LINDiagResponseID = 0x3D

// Deprecated: use LINMaxDataLen.
const MaxDataLen = LINMaxDataLen

// Deprecated: use LINMaxID.
const MaxID = LINMaxID

// Deprecated: use LINDiagRequestID.
const DiagRequestID = LINDiagRequestID

// Deprecated: use LINDiagResponseID.
const DiagResponseID = LINDiagResponseID

// ChecksumType selects the checksum algorithm applied to a frame.
type ChecksumType uint8

const (
	// ClassicChecksum (LIN 1.x) covers the data bytes only.
	ClassicChecksum ChecksumType = iota
	// EnhancedChecksum (LIN 2.x) covers PID + data bytes.
	EnhancedChecksum
)

// SubscriberOption configures a subscription channel.
// Re-exported from relay for convenience; callers may use relay.SubscriberOption directly.
type SubscriberOption = relay.SubscriberOption

// Frame is a LIN bus frame.
//
// A LIN frame is identified by a 6-bit ID (0x00–0x3F). The two
// most-significant bits of the Protected Identifier (PID) are parity bits
// computed by ProtectID. Data is 1–8 bytes. Checksum covers the payload
// (classic) or PID + payload (enhanced).
//
//fusa:req REQ-LIN-001
//fusa:req REQ-LIN-002
//fusa:req REQ-LIN-003
type Frame struct {
	// ID is the 6-bit frame identifier (0x00–0x3F).
	ID uint8

	// Data is the frame payload (1–8 bytes).
	Data []byte

	// Checksum is the wire checksum byte appended after the data bytes.
	Checksum uint8

	// ChecksumType indicates whether the checksum is classic or enhanced.
	ChecksumType ChecksumType
}

// Filter selects frames by ID.
//
// A frame passes when frame.ID == ID (exact match).
// The zero value matches no frames; use Filter{All: true} to receive all frames.
type Filter struct {
	// ID is the exact LIN frame identifier to match (0x00–0x3F).
	ID uint8

	// All overrides ID and matches every frame when true.
	All bool
}

// Matches reports whether f passes the filter.
func (flt Filter) Matches(f Frame) bool {
	if flt.All {
		return true
	}
	return f.ID == flt.ID
}

// ScheduleEntry is one slot in a LIN schedule table.
type ScheduleEntry struct {
	// ID is the frame identifier transmitted by the master in this slot.
	ID uint8

	// DelayMs is the slot duration in milliseconds.
	DelayMs uint32
}

// Bus is the interface implemented by all LIN bus transports.
//
//fusa:req REQ-LIN-011
//fusa:req REQ-LIN-012
type Bus interface {
	// Publish registers a response payload for the given frame ID.
	// When the master requests that ID, the supplied data is sent.
	// Passing nil data removes a previously registered response.
	//
	//fusa:req REQ-LIN-011
	//fusa:req REQ-LIN-019
	Publish(id uint8, data []byte) error

	// Subscribe returns a channel that delivers frames matching any of the
	// supplied filters. Pass nil to receive all frames.
	// opts configures channel delivery (depth, back-pressure per relay §14).
	//
	//fusa:req REQ-LIN-012
	//fusa:req REQ-LIN-020
	Subscribe(filters []Filter, opts ...SubscriberOption) (<-chan Frame, error)

	// Close releases all resources and closes all subscriber channels.
	Close() error
}

// MasterBus extends Bus with the ability to drive frame exchanges.
// It is implemented by transports that support master-node operation.
//
//fusa:req REQ-LIN-013
//fusa:req REQ-LIN-014
type MasterBus interface {
	Bus

	// SendHeader drives a frame exchange: transmit break+sync+PID for id,
	// collect the slave response (if any), verify checksum, and broadcast
	// the resulting Frame to all subscribers.
	// Returns ErrNoResponse when no slave response was registered.
	//
	//fusa:req REQ-LIN-013
	//fusa:req REQ-LIN-014
	SendHeader(ctx context.Context, id uint8) (Frame, error)
}

// ErrNoResponse is returned by MasterBus.SendHeader when no slave has
// registered a response for the requested frame ID.
//
//fusa:req REQ-LIN-014
//fusa:req REQ-LIN-021
var ErrNoResponse = errors.New("lin: no slave response registered for frame ID")

// ProtectID computes the Protected Identifier for a 6-bit LIN frame ID.
//
// The two parity bits are appended in bits 6 and 7 of the returned byte:
//
//	P0 = ID0 ^ ID1 ^ ID2 ^ ID4  (bit 6)
//	P1 = !(ID1 ^ ID3 ^ ID4 ^ ID5) (bit 7)
//
//fusa:req REQ-LIN-004
//fusa:req REQ-LIN-005
//fusa:req REQ-LIN-018
func ProtectID(id uint8) uint8 {
	if id > LINMaxID {
		id &= LINMaxID
	}
	p0 := ((id >> 0) ^ (id >> 1) ^ (id >> 2) ^ (id >> 4)) & 0x01
	p1 := (^((id >> 1) ^ (id >> 3) ^ (id >> 4) ^ (id >> 5))) & 0x01
	return id | (p0 << 6) | (p1 << 7)
}

// VerifyPID checks that the parity bits in a Protected Identifier are correct.
// It returns the raw 6-bit ID and nil on success, or an error on parity failure.
//
//fusa:req REQ-LIN-006
//fusa:req REQ-LIN-007
func VerifyPID(pid uint8) (uint8, error) {
	id := pid & LINMaxID
	if ProtectID(id) != pid {
		return 0, fmt.Errorf("lin: PID 0x%02X parity mismatch", pid)
	}
	return id, nil
}

// CalcChecksum computes the LIN checksum for the given PID and data.
//
// Classic checksum (LIN 1.x) sums data bytes only (pid ignored).
// Enhanced checksum (LIN 2.x) includes the PID byte in the sum.
// Both use inverted carry-around 8-bit addition.
//
//fusa:req REQ-LIN-008
//fusa:req REQ-LIN-009
//fusa:req REQ-LIN-010
func CalcChecksum(pid uint8, data []byte, ct ChecksumType) uint8 {
	var sum uint16
	if ct == EnhancedChecksum {
		sum = uint16(pid)
	}
	for _, b := range data {
		sum += uint16(b)
		if sum > 0xFF {
			sum -= 0xFF // carry-around (not 0x100)
		}
	}
	return uint8(0xFF - uint8(sum))
}

// ValidateFrame checks that f is a well-formed LIN frame.
//
//fusa:req REQ-LIN-001
//fusa:req REQ-LIN-002
//fusa:req REQ-LIN-003
//fusa:req REQ-LIN-015
//fusa:req REQ-LIN-016
//fusa:req REQ-LIN-017
func ValidateFrame(f Frame) error {
	if f.ID > LINMaxID {
		return fmt.Errorf("lin: frame ID 0x%02X exceeds maximum 0x%02X", f.ID, LINMaxID)
	}
	if len(f.Data) == 0 {
		return errors.New("lin: frame data must not be empty")
	}
	if len(f.Data) > LINMaxDataLen {
		return fmt.Errorf("lin: frame data length %d exceeds maximum %d", len(f.Data), LINMaxDataLen)
	}
	return nil
}
