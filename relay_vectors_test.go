// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Conformance tests against the RELAY spec v0.3 golden reference vectors
// (spec/vectors/). The fixtures under testdata/relay-vectors/ are verbatim
// copies of the published vectors; they verify ToMessage()/FromMessage() and
// ValidateFrame() against the canonical expected output.

package lin_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	relay "github.com/SoundMatt/RELAY"
	lin "github.com/SoundMatt/go-LIN"
)

type vector struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Kind        string          `json:"kind"`
	Value       json.RawMessage `json:"value"`
	Message     json.RawMessage `json:"message"`
	Error       string          `json:"error"`
}

func loadVector(t *testing.T, path string) vector {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "relay-vectors", path))
	if err != nil {
		t.Fatalf("read vector %s: %v", path, err)
	}
	var v vector
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("unmarshal vector %s: %v", path, err)
	}
	return v
}

// ── SpecVersion tracks the linked RELAY module ────────────────────────────────

func TestSpecVersion_tracksRelay(t *testing.T) {
	if lin.SpecVersion != relay.SpecVersion {
		t.Errorf("lin.SpecVersion = %q; must track relay.SpecVersion = %q",
			lin.SpecVersion, relay.SpecVersion)
	}
}

// ── Golden frame vector: canonical JSON, ToMessage, FromMessage round-trip ────

func TestVector_linFrame_canonicalJSON(t *testing.T) {
	v := loadVector(t, "lin-frame.json")

	var f lin.Frame
	if err := json.Unmarshal(v.Value, &f); err != nil {
		t.Fatalf("unmarshal value into Frame: %v", err)
	}

	// Re-marshal the Frame and confirm it matches the canonical value byte
	// content (schema conformance for the lin.Frame type, spec §15.3).
	got, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("marshal Frame: %v", err)
	}
	var gotObj, wantObj map[string]any
	_ = json.Unmarshal(got, &gotObj)
	_ = json.Unmarshal(v.Value, &wantObj)
	if !reflect.DeepEqual(gotObj, wantObj) {
		t.Errorf("canonical JSON mismatch:\n got  %s\n want %s", got, v.Value)
	}
}

func TestVector_linFrame_toMessage(t *testing.T) {
	v := loadVector(t, "lin-frame.json")

	var f lin.Frame
	if err := json.Unmarshal(v.Value, &f); err != nil {
		t.Fatalf("unmarshal value into Frame: %v", err)
	}

	var want relay.Message
	if err := json.Unmarshal(v.Message, &want); err != nil {
		t.Fatalf("unmarshal expected message: %v", err)
	}

	got := f.ToMessage()
	if !reflect.DeepEqual(got, want) {
		gj, _ := json.Marshal(got)
		t.Errorf("ToMessage mismatch:\n got  %s\n want %s", gj, v.Message)
	}
}

func TestVector_linFrame_fromMessageRoundTrip(t *testing.T) {
	v := loadVector(t, "lin-frame.json")

	var want lin.Frame
	if err := json.Unmarshal(v.Value, &want); err != nil {
		t.Fatalf("unmarshal value into Frame: %v", err)
	}

	got, err := lin.FromMessage(want.ToMessage())
	if err != nil {
		t.Fatalf("FromMessage: %v", err)
	}
	if got.ID != want.ID || got.ChecksumType != want.ChecksumType ||
		got.Checksum != want.Checksum || !bytes.Equal(got.Data, want.Data) {
		t.Errorf("FromMessage round-trip mismatch: got %+v, want %+v", got, want)
	}
}

// ── Error vectors: ValidateFrame must reject with ErrInvalidFrame ─────────────

func TestVector_errorFrames_validateFrame(t *testing.T) {
	for _, path := range []string{
		"errors/lin-id-overflow.json",
		"errors/lin-diagnostic-wrong-checksum.json",
	} {
		v := loadVector(t, path)
		t.Run(v.Name, func(t *testing.T) {
			if v.Error != "ErrInvalidFrame" {
				t.Fatalf("vector declares error %q, expected ErrInvalidFrame", v.Error)
			}
			var f lin.Frame
			if err := json.Unmarshal(v.Value, &f); err != nil {
				t.Fatalf("unmarshal value into Frame: %v", err)
			}
			err := lin.ValidateFrame(f)
			if err == nil {
				t.Fatalf("ValidateFrame accepted invalid frame %+v", f)
			}
			if !errors.Is(err, lin.ErrInvalidFrame) {
				t.Errorf("ValidateFrame error = %v; want errors.Is ErrInvalidFrame", err)
			}
		})
	}
}
