// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package ldf parses LIN Description Files (LDF) as defined by the LIN
// Consortium specification (LIN 2.x). An LDF describes the full topology
// of a LIN cluster: protocol version, baud rate, nodes, frames, signals,
// and schedule tables.
//
// Usage:
//
//	db, err := ldf.Parse(strings.NewReader(ldfContent))
//	frames := db.Frames()
//	sig := db.Signal("EngineSpeed")
//
//fusa:req REQ-LDF-001
//fusa:req REQ-LDF-002
//fusa:req REQ-LDF-003
//fusa:req REQ-LDF-004
//fusa:req REQ-LDF-005
//fusa:req REQ-LDF-006
//fusa:req REQ-LDF-007
//fusa:req REQ-LDF-008
//fusa:req REQ-LDF-009
//fusa:req REQ-LDF-010
//fusa:req REQ-LDF-011
//fusa:req REQ-LDF-012
//fusa:req REQ-LDF-013
//fusa:req REQ-LDF-014
//fusa:req REQ-LDF-015
package ldf

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	lin "github.com/SoundMatt/go-LIN"
)

// DB holds the parsed contents of an LDF file.
//
//fusa:req REQ-LDF-001
//fusa:req REQ-LDF-002
//fusa:req REQ-LDF-003
//fusa:req REQ-LDF-004
type DB struct {
	ProtocolVersion string
	LanguageVersion string
	Speed           float64 // kbps
	MasterNode      string
	SlaveNodes      []string
	frames          map[uint8]*Frame
	signals         map[string]*Signal
	schedules       map[string][]lin.ScheduleEntry
}

// Frame describes a LIN frame as declared in the LDF Frames section.
//
//fusa:req REQ-LDF-005
//fusa:req REQ-LDF-006
type Frame struct {
	Name      string
	ID        uint8
	Publisher string
	Length    int
	Signals   []SignalRef
}

// SignalRef is a signal embedded within a frame at a given bit offset.
//
//fusa:req REQ-LDF-006
type SignalRef struct {
	Name      string
	BitOffset int
}

// Signal describes a signal as declared in the LDF Signals section.
//
//fusa:req REQ-LDF-007
//fusa:req REQ-LDF-008
type Signal struct {
	Name        string
	BitWidth    int
	InitValue   uint64
	Publisher   string
	Subscribers []string
	Scale       float64
	Offset      float64
	Unit        string
	Min         float64
	Max         float64
}

// Parse reads and parses an LDF file from r.
// It never panics; malformed input results in an empty or partial DB with a
// non-nil error.
//
//fusa:req REQ-LDF-001
//fusa:req REQ-LDF-002
//fusa:req REQ-LDF-003
//fusa:req REQ-LDF-004
//fusa:req REQ-LDF-014
//fusa:req REQ-SEC-001
func Parse(r io.Reader) (*DB, error) {
	db := &DB{
		frames:    make(map[uint8]*Frame),
		signals:   make(map[string]*Signal),
		schedules: make(map[string][]lin.ScheduleEntry),
	}
	if err := db.parse(r); err != nil {
		return nil, fmt.Errorf("ldf: parse: %w", err)
	}
	return db, nil
}

// Frames returns all frames declared in the LDF, keyed by frame ID.
// The returned map is a defensive copy; mutations do not affect the DB.
//
//fusa:req REQ-LDF-005
//fusa:req REQ-LDF-015
func (db *DB) Frames() map[uint8]*Frame {
	out := make(map[uint8]*Frame, len(db.frames))
	for k, v := range db.frames {
		cp := *v
		out[k] = &cp
	}
	return out
}

// Frame returns the frame with the given ID, or nil if not found.
//
//fusa:req REQ-LDF-005
//fusa:req REQ-LDF-012
func (db *DB) Frame(id uint8) *Frame {
	f, ok := db.frames[id]
	if !ok {
		return nil
	}
	cp := *f
	return &cp
}

// Signal returns the signal with the given name, or nil if not found.
//
//fusa:req REQ-LDF-007
//fusa:req REQ-LDF-013
func (db *DB) Signal(name string) *Signal {
	s, ok := db.signals[name]
	if !ok {
		return nil
	}
	cp := *s
	return &cp
}

// Signals returns all signals declared in the LDF, keyed by name.
//
//fusa:req REQ-LDF-007
//fusa:req REQ-LDF-008
func (db *DB) Signals() map[string]*Signal {
	out := make(map[string]*Signal, len(db.signals))
	for k, v := range db.signals {
		cp := *v
		out[k] = &cp
	}
	return out
}

// Schedule returns the schedule table with the given name, or nil.
//
//fusa:req REQ-LDF-011
func (db *DB) Schedule(name string) []lin.ScheduleEntry {
	s, ok := db.schedules[name]
	if !ok {
		return nil
	}
	out := make([]lin.ScheduleEntry, len(s))
	copy(out, s)
	return out
}

// Decode extracts signal values from a raw frame payload.
// Returns a map of signal name → raw uint64 value (unscaled).
// Returns nil when the frame ID is not present in the LDF.
//
//fusa:req REQ-LDF-009
//fusa:req REQ-LDF-010
func (db *DB) Decode(id uint8, data []byte) map[string]uint64 {
	f, ok := db.frames[id]
	if !ok {
		return nil
	}
	result := make(map[string]uint64, len(f.Signals))
	for _, ref := range f.Signals {
		sig, ok := db.signals[ref.Name]
		if !ok {
			continue
		}
		result[ref.Name] = extractBits(data, ref.BitOffset, sig.BitWidth)
	}
	return result
}

// extractBits extracts bitWidth bits starting at bitOffset (LSB first, Intel byte order).
//
//fusa:req REQ-LDF-009
func extractBits(data []byte, bitOffset, bitWidth int) uint64 {
	var val uint64
	for i := 0; i < bitWidth; i++ {
		byteIdx := (bitOffset + i) / 8
		bitIdx := uint((bitOffset + i) % 8)
		if byteIdx >= len(data) {
			break
		}
		if data[byteIdx]&(1<<bitIdx) != 0 {
			val |= 1 << uint(i)
		}
	}
	return val
}

// parse does the actual lexing and parsing of the LDF text.
func (db *DB) parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// strip // comments
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	p := &ldfParser{lines: lines}
	return p.parseTop(db)
}

type ldfParser struct {
	lines []string
	pos   int
}

func (p *ldfParser) peek() string {
	if p.pos >= len(p.lines) {
		return ""
	}
	return p.lines[p.pos]
}

func (p *ldfParser) next() string {
	l := p.peek()
	p.pos++
	return l
}

func (p *ldfParser) parseTop(db *DB) error {
	for p.pos < len(p.lines) {
		line := p.peek()
		switch {
		case strings.HasPrefix(line, "LIN_protocol_version"):
			db.ProtocolVersion = extractQuotedValue(line)
			p.next()
		case strings.HasPrefix(line, "LIN_language_version"):
			db.LanguageVersion = extractQuotedValue(line)
			p.next()
		case strings.HasPrefix(line, "LIN_speed"):
			db.Speed = extractFloatValue(line)
			p.next()
		case strings.HasPrefix(line, "Nodes"):
			p.next()
			if err := p.parseNodes(db); err != nil {
				return err
			}
		case strings.HasPrefix(line, "Signals"):
			p.next()
			if err := p.parseSignals(db); err != nil {
				return err
			}
		case strings.HasPrefix(line, "Frames"):
			p.next()
			if err := p.parseFrames(db); err != nil {
				return err
			}
		case strings.HasPrefix(line, "Schedule_tables"):
			p.next()
			if err := p.parseScheduleTables(db); err != nil {
				return err
			}
		default:
			p.next() // skip unknown sections
		}
	}
	return nil
}

func (p *ldfParser) parseNodes(db *DB) error {
	if strings.HasPrefix(p.peek(), "{") {
		p.next()
	}
	for p.pos < len(p.lines) {
		line := p.peek()
		if line == "}" {
			p.next()
			return nil
		}
		p.next()
		if strings.HasPrefix(line, "Master:") {
			parts := strings.SplitN(strings.TrimPrefix(line, "Master:"), ",", 2)
			if len(parts) > 0 {
				db.MasterNode = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[0]), ";"))
			}
		} else if strings.HasPrefix(line, "Slaves:") {
			rest := strings.TrimPrefix(line, "Slaves:")
			rest = strings.TrimSuffix(strings.TrimSpace(rest), ";")
			for _, s := range strings.Split(rest, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					db.SlaveNodes = append(db.SlaveNodes, s)
				}
			}
		}
	}
	return fmt.Errorf("ldf: unterminated Nodes section")
}

func (p *ldfParser) parseSignals(db *DB) error {
	if strings.HasPrefix(p.peek(), "{") {
		p.next()
	}
	for p.pos < len(p.lines) {
		line := p.peek()
		if line == "}" {
			p.next()
			return nil
		}
		p.next()
		// name : bitWidth, initValue, publisher, subscribers ;
		line = strings.TrimSuffix(strings.TrimSpace(line), ";")
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		name := strings.TrimSpace(line[:colonIdx])
		rest := strings.TrimSpace(line[colonIdx+1:])
		parts := strings.SplitN(rest, ",", 4)
		if len(parts) < 3 {
			continue
		}
		sig := &Signal{Name: name}
		if bw, err := parseInt(strings.TrimSpace(parts[0])); err == nil {
			sig.BitWidth = int(bw)
		}
		sig.InitValue, _ = parseUint(strings.TrimSpace(parts[1]))
		sig.Publisher = strings.TrimSpace(parts[2])
		if len(parts) > 3 {
			for _, sub := range strings.Split(parts[3], ",") {
				sub = strings.TrimSpace(sub)
				if sub != "" {
					sig.Subscribers = append(sig.Subscribers, sub)
				}
			}
		}
		db.signals[name] = sig
	}
	return fmt.Errorf("ldf: unterminated Signals section")
}

func (p *ldfParser) parseFrames(db *DB) error {
	if strings.HasPrefix(p.peek(), "{") {
		p.next()
	}
	for p.pos < len(p.lines) {
		line := p.peek()
		if line == "}" {
			p.next()
			return nil
		}
		// frame header: name : id, publisher, length {
		if strings.HasSuffix(line, "{") || strings.Contains(line, "{") {
			p.next()
			fr, err := parseFrameHeader(line)
			if err != nil {
				continue
			}
			// parse signal refs
			for p.pos < len(p.lines) {
				inner := p.peek()
				if inner == "}" {
					p.next()
					break
				}
				p.next()
				inner = strings.TrimSuffix(strings.TrimSpace(inner), ";")
				// sigName, bitOffset
				parts := strings.SplitN(inner, ",", 2)
				if len(parts) == 2 {
					sigName := strings.TrimSpace(parts[0])
					offset, _ := parseInt(strings.TrimSpace(parts[1]))
					fr.Signals = append(fr.Signals, SignalRef{Name: sigName, BitOffset: int(offset)})
				}
			}
			db.frames[fr.ID] = fr
		} else {
			p.next()
		}
	}
	return fmt.Errorf("ldf: unterminated Frames section")
}

func parseFrameHeader(line string) (*Frame, error) {
	// name : id, publisher, length {
	line = strings.TrimSuffix(strings.TrimSpace(line), "{")
	line = strings.TrimSpace(line)
	colonIdx := strings.Index(line, ":")
	if colonIdx < 0 {
		return nil, fmt.Errorf("ldf: invalid frame header: %q", line)
	}
	name := strings.TrimSpace(line[:colonIdx])
	rest := strings.TrimSpace(line[colonIdx+1:])
	parts := strings.Split(rest, ",")
	if len(parts) < 3 {
		return nil, fmt.Errorf("ldf: too few fields in frame header: %q", line)
	}
	id, err := parseInt(strings.TrimSpace(parts[0]))
	if err != nil {
		return nil, fmt.Errorf("ldf: invalid frame ID in %q: %w", line, err)
	}
	publisher := strings.TrimSpace(parts[1])
	length, _ := parseInt(strings.TrimSpace(parts[2]))
	return &Frame{
		Name:      name,
		ID:        uint8(id),
		Publisher: publisher,
		Length:    int(length),
	}, nil
}

func (p *ldfParser) parseScheduleTables(db *DB) error {
	if strings.HasPrefix(p.peek(), "{") {
		p.next()
	}
	for p.pos < len(p.lines) {
		line := p.peek()
		if line == "}" {
			p.next()
			return nil
		}
		// table name {
		if strings.HasSuffix(line, "{") {
			tableName := strings.TrimSpace(strings.TrimSuffix(line, "{"))
			tableName = strings.TrimSpace(tableName)
			p.next()
			var entries []lin.ScheduleEntry
			for p.pos < len(p.lines) {
				inner := p.peek()
				if inner == "}" {
					p.next()
					break
				}
				p.next()
				inner = strings.TrimSuffix(strings.TrimSpace(inner), ";")
				// frame_name delay ms ;
				// or: AssignFrameId { node, frame } delay ms
				if strings.HasPrefix(inner, "AssignFrameId") {
					// diagnostic slot — skip for now
					continue
				}
				parts := strings.Fields(inner)
				if len(parts) >= 3 && strings.EqualFold(parts[1], "delay") {
					// parts[2] = delay_ms (may be "N ms" or "N.0 ms")
					delayStr := parts[2]
					delay, _ := parseUint(delayStr)
					// resolve frame name → ID
					id := frameIDByName(db, parts[0])
					if id > lin.MaxID {
						continue
					}
					entries = append(entries, lin.ScheduleEntry{ID: id, DelayMs: uint32(delay)})
				}
			}
			db.schedules[tableName] = entries
		} else {
			p.next()
		}
	}
	return fmt.Errorf("ldf: unterminated Schedule_tables section")
}

func frameIDByName(db *DB, name string) uint8 {
	for id, f := range db.frames {
		if f.Name == name {
			return id
		}
	}
	return 0xFF
}

func extractQuotedValue(line string) string {
	start := strings.Index(line, `"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(line[start+1:], `"`)
	if end < 0 {
		return ""
	}
	return line[start+1 : start+1+end]
}

func extractFloatValue(line string) float64 {
	parts := strings.Fields(line)
	for _, p := range parts {
		p = strings.TrimSuffix(p, ";")
		if f, err := strconv.ParseFloat(p, 64); err == nil {
			return f
		}
	}
	return 0
}

func parseInt(s string) (int64, error) {
	s = strings.TrimSuffix(strings.TrimSpace(s), ";")
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err := strconv.ParseInt(s[2:], 16, 64)
		return v, err
	}
	return strconv.ParseInt(s, 10, 64)
}

func parseUint(s string) (uint64, error) {
	s = strings.TrimSuffix(strings.TrimSpace(s), ";")
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	// handle float strings like "10.0"
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return uint64(f), nil
	}
	return strconv.ParseUint(s, 10, 64)
}
