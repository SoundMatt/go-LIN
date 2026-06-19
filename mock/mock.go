// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package mock provides a fully functional in-process LIN bus for testing.
// It wraps the virtual package and satisfies lin.MasterBus.
//
// Usage:
//
//	bus, _ := mock.New()
//	defer bus.Close()
package mock

import "github.com/SoundMatt/go-LIN/virtual"

// New returns a fully functional in-process LIN bus for testing.
// It implements lin.MasterBus (Publish, Subscribe, SendHeader, SetSchedule, Close).
//
//fusa:req REQ-MOCK-001
func New() (*virtual.Bus, error) {
	return virtual.New()
}
