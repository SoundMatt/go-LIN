// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package safety_test

import (
	"encoding/binary"
	"errors"
	"sync"
	"testing"

	"github.com/SoundMatt/go-LIN/safety"
)

var cfg = safety.Config{DataID: 0x0001, SourceID: 0x0010}

// ── REQ-SAFETY-001: DataID in bytes 0-1 ──────────────────────────────────────

//fusa:test REQ-SAFETY-001

func TestProtect_dataIDInHeader(t *testing.T) {
	p := safety.NewProtector(safety.Config{DataID: 0xABCD, SourceID: 0x0000})
	out := p.Protect([]byte{0x01})
	got := binary.LittleEndian.Uint16(out[0:2])
	if got != 0xABCD {
		t.Errorf("DataID in bytes 0-1 = 0x%04X, want 0xABCD", got)
	}
}

// ── REQ-SAFETY-002: SourceID in bytes 2-3 ────────────────────────────────────

//fusa:test REQ-SAFETY-002

func TestProtect_sourceIDInHeader(t *testing.T) {
	p := safety.NewProtector(safety.Config{DataID: 0x0000, SourceID: 0x1234})
	out := p.Protect([]byte{0x01})
	got := binary.LittleEndian.Uint16(out[2:4])
	if got != 0x1234 {
		t.Errorf("SourceID in bytes 2-3 = 0x%04X, want 0x1234", got)
	}
}

// ── REQ-SAFETY-003: counter starts at 0 and increments ───────────────────────

//fusa:test REQ-SAFETY-003

func TestProtect_counterStartsAtZero(t *testing.T) {
	p := safety.NewProtector(cfg)
	out := p.Protect([]byte{0x01})
	seq := binary.LittleEndian.Uint32(out[4:8])
	if seq != 0 {
		t.Errorf("first SequenceCounter = %d, want 0", seq)
	}
}

func TestProtect_counterIncrements(t *testing.T) {
	p := safety.NewProtector(cfg)
	for i := uint32(0); i < 5; i++ {
		out := p.Protect([]byte{byte(i)})
		seq := binary.LittleEndian.Uint32(out[4:8])
		if seq != i {
			t.Errorf("call %d: SequenceCounter = %d, want %d", i, seq, i)
		}
	}
}

// ── REQ-SAFETY-004: counter in bytes 4-7 ─────────────────────────────────────

//fusa:test REQ-SAFETY-004

func TestProtect_counterInBytes4To7(t *testing.T) {
	p := safety.NewProtector(cfg)
	out := p.Protect([]byte{0x01})
	seq := binary.LittleEndian.Uint32(out[4:8])
	if seq != 0 {
		t.Errorf("SequenceCounter in bytes 4-7 = %d, want 0", seq)
	}
}

// ── REQ-SAFETY-005: CRC-16/CCITT-FALSE ───────────────────────────────────────

//fusa:test REQ-SAFETY-005

func TestProtect_crcCoversHeaderAndPayload(t *testing.T) {
	p1 := safety.NewProtector(cfg)
	p2 := safety.NewProtector(cfg)
	out1 := p1.Protect([]byte{0xAA})
	out2 := p2.Protect([]byte{0xBB})
	crc1 := binary.LittleEndian.Uint16(out1[8:10])
	crc2 := binary.LittleEndian.Uint16(out2[8:10])
	if crc1 == crc2 {
		t.Error("CRC should differ for different payloads")
	}
}

// ── REQ-SAFETY-006: CRC in bytes 8-9 ─────────────────────────────────────────

//fusa:test REQ-SAFETY-006

func TestProtect_crcInBytes8And9(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)
	out := p.Protect([]byte{0xDE, 0xAD})
	// corrupt bytes 8-9 and confirm Unwrap detects it
	out[8] ^= 0xFF
	out[9] ^= 0xFF
	_, err := r.Unwrap(out)
	if err == nil {
		t.Error("expected CRC error after corrupting bytes 8-9")
	}
}

// ── REQ-SAFETY-007: ErrHeaderTooShort ────────────────────────────────────────

//fusa:test REQ-SAFETY-007

func TestUnwrap_headerTooShort(t *testing.T) {
	r := safety.NewReceiver(cfg)
	_, err := r.Unwrap([]byte{0x01, 0x02, 0x03})
	if err == nil {
		t.Fatal("expected error for short payload")
	}
	var e2e *safety.E2EError
	if !errors.As(err, &e2e) {
		t.Fatalf("expected *E2EError, got %T", err)
	}
	if e2e.Kind != safety.ErrHeaderTooShort {
		t.Errorf("ErrorKind = %v, want ErrHeaderTooShort", e2e.Kind)
	}
}

// ── REQ-SAFETY-008: ErrCRCMismatch ───────────────────────────────────────────

//fusa:test REQ-SAFETY-008
//fusa:test REQ-SEC-002

func TestUnwrap_crcMismatch(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	protected := p.Protect([]byte{0xAA, 0xBB})
	// corrupt the CRC bytes (bytes 8-9)
	protected[8] ^= 0xFF
	protected[9] ^= 0xFF

	_, err := r.Unwrap(protected)
	if err == nil {
		t.Fatal("expected E2E error for CRC mismatch")
	}
	var e2e *safety.E2EError
	if !errors.As(err, &e2e) {
		t.Fatalf("expected *E2EError, got %T", err)
	}
	if e2e.Kind != safety.ErrCRCMismatch {
		t.Errorf("ErrorKind = %v, want ErrCRCMismatch", e2e.Kind)
	}
}

func TestUnwrap_dataCorruption(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	protected := p.Protect([]byte{0x01, 0x02, 0x03})
	// flip a byte in the payload region (after the header)
	protected[10] ^= 0x01

	_, err := r.Unwrap(protected)
	if err == nil {
		t.Error("expected CRC error for corrupted data")
	}
}

// ── REQ-SAFETY-009: ErrSequenceGap ───────────────────────────────────────────

//fusa:test REQ-SAFETY-009
//fusa:test REQ-SEC-003

func TestUnwrap_sequenceGap(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	// first message OK
	_, _ = r.Unwrap(p.Protect([]byte{0x01}))

	// skip seq 1 by creating a new protector (simulates gap)
	p2 := safety.NewProtector(cfg)
	_ = p2.Protect([]byte{0x00}) // seq 0
	_ = p2.Protect([]byte{0x00}) // seq 1 — skip
	third := p2.Protect([]byte{0x03}) // seq 2

	_, err := r.Unwrap(third)
	if err == nil {
		t.Fatal("expected sequence gap error")
	}
	var e2e *safety.E2EError
	if !errors.As(err, &e2e) {
		t.Fatalf("expected *E2EError, got %T", err)
	}
	if e2e.Kind != safety.ErrSequenceGap {
		t.Errorf("ErrorKind = %v, want ErrSequenceGap", e2e.Kind)
	}
}

// ── REQ-SAFETY-010: Unwrap returns original payload ──────────────────────────

//fusa:test REQ-SAFETY-010
//fusa:test REQ-SAFETY-011

func TestProtectUnwrap_roundtrip(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	payload := []byte{0x01, 0x02, 0x03, 0x04}
	protected := p.Protect(payload)
	got, err := r.Unwrap(protected)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("Unwrap = %v, want %v", got, payload)
	}
}

func TestProtect_sequenceMonotonic(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	for i := 0; i < 10; i++ {
		protected := p.Protect([]byte{byte(i)})
		if _, err := r.Unwrap(protected); err != nil {
			t.Fatalf("Unwrap iteration %d: %v", i, err)
		}
	}
}

// ── REQ-SAFETY-012: output length = 10 + len(payload) ───────────────────────

//fusa:test REQ-SAFETY-012

func TestProtect_outputLength(t *testing.T) {
	p := safety.NewProtector(cfg)
	cases := [][]byte{{}, {0x01}, {0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}}
	for _, payload := range cases {
		out := p.Protect(payload)
		want := 10 + len(payload)
		if len(out) != want {
			t.Errorf("Protect output len = %d, want %d for payload len %d", len(out), want, len(payload))
		}
	}
}

// ── REQ-SAFETY-013: first message accepts any counter ────────────────────────

//fusa:test REQ-SAFETY-013

func TestUnwrap_firstMessageAcceptsAnyCounter(t *testing.T) {
	// Build a Protect with a high initial counter by discarding 99 messages
	p := safety.NewProtector(cfg)
	for i := 0; i < 99; i++ {
		_ = p.Protect([]byte{0x00})
	}
	// message 100 has counter=99
	msg := p.Protect([]byte{0xAB})

	r := safety.NewReceiver(cfg)
	got, err := r.Unwrap(msg)
	if err != nil {
		t.Fatalf("first Unwrap with counter=99 should succeed: %v", err)
	}
	if len(got) != 1 || got[0] != 0xAB {
		t.Errorf("payload = %v, want [0xAB]", got)
	}
}

// ── REQ-SAFETY-014: Protect is safe for concurrent calls ─────────────────────

//fusa:test REQ-SAFETY-014

func TestProtect_concurrent(t *testing.T) {
	p := safety.NewProtector(cfg)
	counters := make(chan uint32, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				out := p.Protect([]byte{0x01})
				seq := binary.LittleEndian.Uint32(out[4:8])
				counters <- seq
			}
		}()
	}
	wg.Wait()
	close(counters)

	// verify all 1000 counter values are unique
	seen := make(map[uint32]bool, 1000)
	for c := range counters {
		if seen[c] {
			t.Errorf("duplicate counter value %d", c)
		}
		seen[c] = true
	}
}

// ── REQ-SAFETY-015: Unwrap returns independent payload copy ──────────────────

//fusa:test REQ-SAFETY-015

func TestUnwrap_independentCopy(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	payload := []byte{0xAA, 0xBB}
	protected := p.Protect(payload)
	got, err := r.Unwrap(protected)
	if err != nil {
		t.Fatalf("Unwrap: %v", err)
	}
	// mutate the returned slice
	got[0] = 0xFF
	// a second Unwrap of the same protected message should return the original payload
	// (Receiver tracks state, so use a fresh receiver)
	r2 := safety.NewReceiver(cfg)
	got2, err := r2.Unwrap(protected)
	if err != nil {
		t.Fatalf("second Unwrap: %v", err)
	}
	if got2[0] != 0xAA {
		t.Errorf("second Unwrap got[0] = 0x%02X, want 0xAA — copy not independent", got2[0])
	}
}

func TestProtect_emptyPayload(t *testing.T) {
	p := safety.NewProtector(cfg)
	r := safety.NewReceiver(cfg)

	protected := p.Protect([]byte{})
	got, err := r.Unwrap(protected)
	if err != nil {
		t.Fatalf("Unwrap empty payload: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty payload, got %v", got)
	}
}

func TestE2EError_message(t *testing.T) {
	e := &safety.E2EError{Kind: safety.ErrCRCMismatch, Counter: 42, Message: "test"}
	msg := e.Error()
	if len(msg) == 0 {
		t.Error("E2EError.Error() returned empty string")
	}
}

func FuzzProtectUnwrap(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0x03, 0x04})
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, payload []byte) {
		p := safety.NewProtector(cfg)
		r := safety.NewReceiver(cfg)
		protected := p.Protect(payload)
		got, err := r.Unwrap(protected)
		if err != nil {
			t.Fatalf("round-trip failed for payload %v: %v", payload, err)
		}
		if string(got) != string(payload) {
			t.Errorf("payload mismatch: got %v, want %v", got, payload)
		}
	})
}
