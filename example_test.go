// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin_test

import (
	"fmt"

	lin "github.com/SoundMatt/go-LIN"
)

// ExampleProtectID computes the Protected Identifier for a 6-bit LIN frame
// ID by appending the two LIN parity bits.
func ExampleProtectID() {
	pid := lin.ProtectID(0x10)
	fmt.Printf("0x%02X\n", pid)
	// Output: 0x50
}

// ExampleVerifyPID recovers the raw frame ID from a Protected Identifier and
// rejects one with a corrupted parity bit.
func ExampleVerifyPID() {
	pid := lin.ProtectID(0x10)

	id, err := lin.VerifyPID(pid)
	fmt.Printf("id=0x%02X err=%v\n", id, err)

	_, err = lin.VerifyPID(pid ^ 0x80) // flip the top parity bit
	fmt.Println(err != nil)
	// Output:
	// id=0x10 err=<nil>
	// true
}

// ExampleCalcChecksum computes both LIN checksum variants for the same
// payload: classic (LIN 1.x, data only) and enhanced (LIN 2.x, PID + data).
func ExampleCalcChecksum() {
	pid := lin.ProtectID(0x10)
	data := []byte{0x01, 0x02, 0x03, 0x04}

	classic := lin.CalcChecksum(0, data, lin.ClassicChecksum)
	enhanced := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	fmt.Printf("classic=0x%02X enhanced=0x%02X\n", classic, enhanced)
	// Output: classic=0xF5 enhanced=0xA5
}

// ExampleValidateFrame accepts a well-formed frame and rejects a
// structurally invalid one — here, data exceeding the 8-byte maximum.
func ExampleValidateFrame() {
	ok := lin.Frame{ID: 0x10, Data: []byte{0x01, 0x02, 0x03, 0x04}}
	fmt.Println(lin.ValidateFrame(ok) == nil)

	bad := lin.Frame{ID: 0x10, Data: make([]byte, lin.LINMaxDataLen+1)}
	fmt.Println(lin.ValidateFrame(bad) != nil)
	// Output:
	// true
	// true
}

// ExampleFrame_ToMessage converts a Frame to the RELAY relay.Message
// envelope used by cross-protocol tooling (spec §15.3/§15.7.3).
func ExampleFrame_ToMessage() {
	f := lin.Frame{ID: 0x10, Data: []byte{0x01, 0x02, 0x03, 0x04}, ChecksumType: lin.EnhancedChecksum}
	msg := f.ToMessage()
	fmt.Println(msg.Protocol, msg.ID, msg.Meta["lin.checksum_type"])
	// Output: LIN 16 enhanced
}
