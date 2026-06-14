// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package ldf_test

import (
	"strings"
	"testing"

	"github.com/SoundMatt/go-LIN/ldf"
)

// ── sample LDF ───────────────────────────────────────────────────────────────

const sampleLDF = `
LIN_description_file;
LIN_protocol_version = "2.1";
LIN_language_version = "2.1";
LIN_speed = 19.2 kbps;

Nodes {
  Master: MasterNode, 1 ms, 0.1 ms;
  Slaves: SlaveA, SlaveB;
}

Signals {
  EngineSpeed:    16, 0x0000, MasterNode, SlaveA;
  MotorTemp:       8, 0x00,   SlaveA, MasterNode;
  WindowPos:       8, 0x00,   SlaveB, MasterNode;
}

Frames {
  EngineFrame: 0x10, MasterNode, 4 {
    EngineSpeed, 0;
    MotorTemp,  16;
  }
  WindowFrame: 0x20, SlaveB, 2 {
    WindowPos, 0;
  }
}

Schedule_tables {
  NormalSchedule {
    EngineFrame delay 10 ms;
    WindowFrame delay 15 ms;
  }
}
`

// ── REQ-LDF-001: protocol version ────────────────────────────────────────────

//fusa:test REQ-LDF-001

func TestParse_header(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.ProtocolVersion != "2.1" {
		t.Errorf("ProtocolVersion = %q, want %q", db.ProtocolVersion, "2.1")
	}
	if db.LanguageVersion != "2.1" {
		t.Errorf("LanguageVersion = %q, want %q", db.LanguageVersion, "2.1")
	}
}

// ── REQ-LDF-002: baud rate ────────────────────────────────────────────────────

//fusa:test REQ-LDF-002

func TestParse_baudRate(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.Speed != 19.2 {
		t.Errorf("Speed = %.1f, want 19.2", db.Speed)
	}
}

// ── REQ-LDF-003: master node ─────────────────────────────────────────────────

//fusa:test REQ-LDF-003

func TestParse_masterNode(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.MasterNode != "MasterNode" {
		t.Errorf("MasterNode = %q, want %q", db.MasterNode, "MasterNode")
	}
}

// ── REQ-LDF-004: slave node list ─────────────────────────────────────────────

//fusa:test REQ-LDF-004

func TestParse_slaveNodes(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(db.SlaveNodes) != 2 {
		t.Errorf("SlaveNodes count = %d, want 2", len(db.SlaveNodes))
	}
}

// ── REQ-LDF-005: frame descriptors ───────────────────────────────────────────

//fusa:test REQ-LDF-005

func TestParse_frames(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	frames := db.Frames()
	if len(frames) != 2 {
		t.Errorf("Frames count = %d, want 2", len(frames))
	}
	f := db.Frame(0x10)
	if f == nil {
		t.Fatal("Frame(0x10) returned nil")
	}
	if f.Name != "EngineFrame" {
		t.Errorf("frame Name = %q, want EngineFrame", f.Name)
	}
	if f.Length != 4 {
		t.Errorf("frame Length = %d, want 4", f.Length)
	}
	if f.Publisher != "MasterNode" {
		t.Errorf("frame Publisher = %q, want MasterNode", f.Publisher)
	}
}

// ── REQ-LDF-006: signal-to-bit-offset mappings ───────────────────────────────

//fusa:test REQ-LDF-006

func TestParse_signalRefs(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	f := db.Frame(0x10)
	if f == nil {
		t.Fatal("Frame(0x10) returned nil")
	}
	if len(f.Signals) != 2 {
		t.Fatalf("frame Signals count = %d, want 2", len(f.Signals))
	}
	// EngineSpeed at offset 0, MotorTemp at offset 16
	offsets := map[string]int{"EngineSpeed": 0, "MotorTemp": 16}
	for _, ref := range f.Signals {
		want, ok := offsets[ref.Name]
		if !ok {
			t.Errorf("unexpected signal ref %q", ref.Name)
			continue
		}
		if ref.BitOffset != want {
			t.Errorf("signal %q BitOffset = %d, want %d", ref.Name, ref.BitOffset, want)
		}
	}
}

// ── REQ-LDF-007: signal bit width ─────────────────────────────────────────────

//fusa:test REQ-LDF-007

func TestParse_signalBitWidth(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	sig := db.Signal("EngineSpeed")
	if sig == nil {
		t.Fatal("Signal(EngineSpeed) returned nil")
	}
	if sig.BitWidth != 16 {
		t.Errorf("EngineSpeed.BitWidth = %d, want 16", sig.BitWidth)
	}
	sig2 := db.Signal("MotorTemp")
	if sig2 == nil {
		t.Fatal("Signal(MotorTemp) returned nil")
	}
	if sig2.BitWidth != 8 {
		t.Errorf("MotorTemp.BitWidth = %d, want 8", sig2.BitWidth)
	}
}

// ── REQ-LDF-008: signal publisher ────────────────────────────────────────────

//fusa:test REQ-LDF-008

func TestParse_signalPublisher(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	sig := db.Signal("EngineSpeed")
	if sig == nil {
		t.Fatal("Signal(EngineSpeed) returned nil")
	}
	if sig.Publisher != "MasterNode" {
		t.Errorf("EngineSpeed.Publisher = %q, want MasterNode", sig.Publisher)
	}
}

// ── REQ-LDF-009: LSB-first Intel byte order ───────────────────────────────────

//fusa:test REQ-LDF-009

func TestParse_decode(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// EngineSpeed (16-bit at offset 0) = 0x1388 = 5000 rpm raw (little-endian: 0x88, 0x13)
	// MotorTemp   (8-bit at offset 16) = 0x50   = 80°C raw
	data := []byte{0x88, 0x13, 0x50, 0x00}
	vals := db.Decode(0x10, data)

	if vals["EngineSpeed"] != 0x1388 {
		t.Errorf("EngineSpeed = 0x%04X, want 0x1388", vals["EngineSpeed"])
	}
	if vals["MotorTemp"] != 0x50 {
		t.Errorf("MotorTemp = 0x%02X, want 0x50", vals["MotorTemp"])
	}
}

// ── REQ-LDF-010: Decode returns nil for unknown frame ID ─────────────────────

//fusa:test REQ-LDF-010

func TestDecode_unknownFrame(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.Decode(0x3F, []byte{0x01}) != nil {
		t.Error("Decode(unknown ID) must return nil")
	}
}

// ── REQ-LDF-011: schedule tables ─────────────────────────────────────────────

//fusa:test REQ-LDF-011

func TestParse_schedules(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	sched := db.Schedule("NormalSchedule")
	if len(sched) != 2 {
		t.Fatalf("NormalSchedule entries = %d, want 2", len(sched))
	}
	if sched[0].DelayMs != 10 {
		t.Errorf("slot 0 delay = %d ms, want 10", sched[0].DelayMs)
	}
	if sched[1].DelayMs != 15 {
		t.Errorf("slot 1 delay = %d ms, want 15", sched[1].DelayMs)
	}
}

// ── REQ-LDF-012: Frame() returns nil for unknown ID ──────────────────────────

//fusa:test REQ-LDF-012

func TestFrame_unknownID(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.Frame(0xFF) != nil {
		t.Error("Frame(0xFF) should return nil")
	}
}

// ── REQ-LDF-013: Signal() returns nil for unknown name ───────────────────────

//fusa:test REQ-LDF-013

func TestSignal_unknownName(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.Signal("DoesNotExist") != nil {
		t.Error("Signal(DoesNotExist) should return nil")
	}
}

// ── REQ-LDF-014: Parse does not panic ────────────────────────────────────────

//fusa:test REQ-LDF-014

func TestParse_emptyInput(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse(empty) unexpected error: %v", err)
	}
	if len(db.Frames()) != 0 {
		t.Error("empty LDF should have no frames")
	}
}

func TestParse_signalsAll(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	sigs := db.Signals()
	if len(sigs) != 3 {
		t.Errorf("Signals count = %d, want 3", len(sigs))
	}
}

// ── REQ-LDF-015: Frames() returns defensive copy ─────────────────────────────

//fusa:test REQ-LDF-015

func TestFrames_defensiveCopy(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	frames1 := db.Frames()
	// delete from returned map
	delete(frames1, 0x10)
	// second call should still have all frames
	frames2 := db.Frames()
	if len(frames2) != 2 {
		t.Errorf("Frames() after external delete: got %d frames, want 2", len(frames2))
	}
}

func FuzzParse(f *testing.F) {
	f.Add(sampleLDF)
	f.Add("")
	f.Add("LIN_protocol_version = \"2.1\";\n")

	f.Fuzz(func(t *testing.T, input string) {
		db, err := ldf.Parse(strings.NewReader(input))
		if err != nil {
			return // parse errors are expected for arbitrary input
		}
		// basic sanity: Frames/Signals/Schedules must not panic
		_ = db.Frames()
		_ = db.Signals()
		for id := uint8(0); id <= 0x3F; id++ {
			_ = db.Frame(id)
		}
	})
}
