// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package safety_test

import (
	"errors"
	"testing"

	"github.com/SoundMatt/go-LIN/safety"
)

//fusa:test REQ-SAFETY-001
//fusa:test REQ-SAFETY-002
//fusa:test REQ-SAFETY-003
//fusa:test REQ-SAFETY-004
//fusa:test REQ-SAFETY-005
//fusa:test REQ-SEOOC-001

var cfg = safety.Config{DataID: 0x0001, SourceID: 0x0010}

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
