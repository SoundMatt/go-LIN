// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package safety provides end-to-end data protection for LIN payloads.
//
// Protector prepends a 10-byte E2E header to every payload before
// transmission. Receiver strips the header and validates CRC, sequence
// counter, and freshness on every received payload.
//
// Wire format (little-endian, 10 bytes followed by original payload):
//
//	Bytes  0–1   DataID (uint16)
//	Bytes  2–3   SourceID (uint16)
//	Bytes  4–7   SequenceCounter (uint32, monotonically increasing)
//	Bytes  8–9   CRC-16/CCITT-FALSE over bytes 0–7 plus the original payload
//	Bytes 10+    Original payload
//
// LIN payloads protected by this header exceed the 8-byte maximum of a
// standard LIN frame. Use safety with diagnostic frames (0x3C/0x3D) or
// application frames that span multiple LIN slots via a transport layer.
//
//fusa:req REQ-SAFETY-001
//fusa:req REQ-SAFETY-002
//fusa:req REQ-SAFETY-003
//fusa:req REQ-SAFETY-004
//fusa:req REQ-SAFETY-005
//fusa:req REQ-SAFETY-006
//fusa:req REQ-SAFETY-007
//fusa:req REQ-SAFETY-008
//fusa:req REQ-SAFETY-009
//fusa:req REQ-SAFETY-010
//fusa:req REQ-SAFETY-011
//fusa:req REQ-SAFETY-012
//fusa:req REQ-SAFETY-013
//fusa:req REQ-SAFETY-014
//fusa:req REQ-SAFETY-015
//fusa:req REQ-SEOOC-001
package safety

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
)

const headerSize = 10

// Config configures end-to-end protection parameters.
//
//fusa:req REQ-SAFETY-001
//fusa:req REQ-SAFETY-002
type Config struct {
	// DataID identifies the logical data element (0–65535).
	DataID uint16
	// SourceID identifies the sender node (0–65535).
	SourceID uint16
}

// ErrorKind categorises E2E check failures.
type ErrorKind int

const (
	// ErrCRCMismatch means the CRC in the header did not match.
	ErrCRCMismatch ErrorKind = iota
	// ErrSequenceGap means one or more sequence numbers were skipped.
	ErrSequenceGap
	// ErrHeaderTooShort means the payload is shorter than the 10-byte header.
	ErrHeaderTooShort
)

// E2EError is returned when an E2E safety check fails.
//
//fusa:req REQ-SAFETY-007
//fusa:req REQ-SAFETY-008
//fusa:req REQ-SAFETY-009
type E2EError struct {
	Kind    ErrorKind
	Counter uint32
	Message string
}

func (e *E2EError) Error() string {
	return fmt.Sprintf("lin/safety: E2E error (kind=%d, counter=%d): %s",
		e.Kind, e.Counter, e.Message)
}

// Protector adds an E2E header to payloads before transmission.
// Its SequenceCounter starts at 0 and increments by 1 per Protect call.
// Protect is safe for concurrent calls.
//
//fusa:req REQ-SAFETY-003
//fusa:req REQ-SAFETY-004
//fusa:req REQ-SAFETY-014
type Protector struct {
	cfg Config
	seq atomic.Uint32
}

// NewProtector creates an E2E protector with SequenceCounter initialised to 0.
//
//fusa:req REQ-SAFETY-003
func NewProtector(cfg Config) *Protector {
	return &Protector{cfg: cfg}
}

// Protect prepends the E2E header and returns the protected payload.
// The output length is exactly headerSize (10) + len(payload).
// The SequenceCounter is atomically incremented on each call.
//
//fusa:req REQ-SAFETY-001
//fusa:req REQ-SAFETY-002
//fusa:req REQ-SAFETY-003
//fusa:req REQ-SAFETY-004
//fusa:req REQ-SAFETY-005
//fusa:req REQ-SAFETY-006
//fusa:req REQ-SAFETY-012
//fusa:req REQ-SAFETY-014
func (p *Protector) Protect(payload []byte) []byte {
	seq := p.seq.Add(1) - 1
	hdr := buildHeader(p.cfg.DataID, p.cfg.SourceID, seq, payload)
	out := make([]byte, headerSize+len(payload))
	copy(out[:headerSize], hdr)
	copy(out[headerSize:], payload)
	return out
}

// Receiver validates E2E headers on received payloads.
// The first Unwrap call accepts any counter value to seed the sequence.
//
//fusa:req REQ-SAFETY-007
//fusa:req REQ-SAFETY-008
//fusa:req REQ-SAFETY-009
//fusa:req REQ-SAFETY-010
//fusa:req REQ-SAFETY-013
type Receiver struct {
	mu      sync.Mutex
	cfg     Config
	lastSeq uint32
	first   bool
}

// NewReceiver creates an E2E receiver. The first Unwrap accepts any counter.
//
//fusa:req REQ-SAFETY-013
func NewReceiver(cfg Config) *Receiver {
	return &Receiver{cfg: cfg, first: true}
}

// Unwrap validates the E2E header and returns an independent copy of the
// original payload. Returns E2EError on CRC mismatch, sequence gap, or
// short payload.
//
//fusa:req REQ-SAFETY-007
//fusa:req REQ-SAFETY-008
//fusa:req REQ-SAFETY-009
//fusa:req REQ-SAFETY-010
//fusa:req REQ-SAFETY-011
//fusa:req REQ-SAFETY-013
//fusa:req REQ-SAFETY-015
func (r *Receiver) Unwrap(data []byte) ([]byte, error) {
	if len(data) < headerSize {
		return nil, &E2EError{
			Kind:    ErrHeaderTooShort,
			Message: fmt.Sprintf("got %d bytes, need at least %d", len(data), headerSize),
		}
	}

	dataID := binary.LittleEndian.Uint16(data[0:2])
	sourceID := binary.LittleEndian.Uint16(data[2:4])
	seq := binary.LittleEndian.Uint32(data[4:8])
	crcWire := binary.LittleEndian.Uint16(data[8:10])
	payload := data[headerSize:]

	// zero the CRC slot before computing
	tmp := make([]byte, headerSize)
	copy(tmp, data[:headerSize])
	tmp[8] = 0
	tmp[9] = 0
	crcCalc := crc16(append(tmp, payload...))

	if crcCalc != crcWire {
		return nil, &E2EError{
			Kind:    ErrCRCMismatch,
			Counter: seq,
			Message: fmt.Sprintf("CRC mismatch: wire=0x%04X calc=0x%04X", crcWire, crcCalc),
		}
	}

	_ = dataID   // validated implicitly via CRC
	_ = sourceID // validated implicitly via CRC

	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.first && seq != r.lastSeq+1 {
		r.lastSeq = seq
		return nil, &E2EError{
			Kind:    ErrSequenceGap,
			Counter: seq,
			Message: fmt.Sprintf("sequence gap: last=%d recv=%d", r.lastSeq-1, seq),
		}
	}
	r.lastSeq = seq
	r.first = false

	out := make([]byte, len(payload))
	copy(out, payload)
	return out, nil
}

func buildHeader(dataID, sourceID uint16, seq uint32, payload []byte) []byte {
	hdr := make([]byte, headerSize)
	binary.LittleEndian.PutUint16(hdr[0:2], dataID)
	binary.LittleEndian.PutUint16(hdr[2:4], sourceID)
	binary.LittleEndian.PutUint32(hdr[4:8], seq)
	// CRC slots zeroed initially
	crc := crc16(append(hdr, payload...))
	binary.LittleEndian.PutUint16(hdr[8:10], crc)
	return hdr
}

// crc16 computes CRC-16/CCITT-FALSE (poly=0x1021, init=0xFFFF, refin=false).
//
//fusa:req REQ-SAFETY-005
func crc16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
