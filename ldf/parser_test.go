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

//fusa:test REQ-LDF-001
//fusa:test REQ-LDF-002
//fusa:test REQ-LDF-003
//fusa:test REQ-LDF-004

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
	if db.Speed != 19.2 {
		t.Errorf("Speed = %.1f, want 19.2", db.Speed)
	}
}

func TestParse_nodes(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.MasterNode != "MasterNode" {
		t.Errorf("MasterNode = %q, want %q", db.MasterNode, "MasterNode")
	}
	if len(db.SlaveNodes) != 2 {
		t.Errorf("SlaveNodes count = %d, want 2", len(db.SlaveNodes))
	}
}

func TestParse_signals(t *testing.T) {
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
	if sig.Publisher != "MasterNode" {
		t.Errorf("EngineSpeed.Publisher = %q, want MasterNode", sig.Publisher)
	}

	sig2 := db.Signal("MotorTemp")
	if sig2 == nil {
		t.Fatal("Signal(MotorTemp) returned nil")
	}
	if sig2.BitWidth != 8 {
		t.Errorf("MotorTemp.BitWidth = %d, want 8", sig2.BitWidth)
	}
}

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
	if len(f.Signals) != 2 {
		t.Errorf("frame Signals count = %d, want 2", len(f.Signals))
	}
}

func TestParse_schedules(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	sched := db.Schedule("NormalSchedule")
	if len(sched) != 2 {
		t.Errorf("NormalSchedule entries = %d, want 2", len(sched))
	}
	if sched[0].DelayMs != 10 {
		t.Errorf("slot 0 delay = %d ms, want 10", sched[0].DelayMs)
	}
	if sched[1].DelayMs != 15 {
		t.Errorf("slot 1 delay = %d ms, want 15", sched[1].DelayMs)
	}
}

func TestParse_decode(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// EngineSpeed (16-bit at offset 0) = 0x1388 = 5000 rpm raw
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

func TestParse_signalNotFound(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.Signal("DoesNotExist") != nil {
		t.Error("Signal(DoesNotExist) should return nil")
	}
}

func TestParse_frameNotFound(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if db.Frame(0xFF) != nil {
		t.Error("Frame(0xFF) should return nil")
	}
}

func TestParse_emptyLDF(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse(empty) unexpected error: %v", err)
	}
	if len(db.Frames()) != 0 {
		t.Error("empty LDF should have no frames")
	}
}

func TestParse_signals_all(t *testing.T) {
	db, err := ldf.Parse(strings.NewReader(sampleLDF))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	sigs := db.Signals()
	if len(sigs) != 3 {
		t.Errorf("Signals count = %d, want 3", len(sigs))
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
