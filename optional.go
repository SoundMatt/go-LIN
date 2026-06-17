// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin

import "context"

// HealthStatus reports the operational state of a node.
type HealthStatus int

const (
	HealthOK       HealthStatus = 0
	HealthDegraded HealthStatus = 1
	HealthDown     HealthStatus = 2
)

// Health carries the health status and optional detail string.
type Health struct {
	Status  HealthStatus `json:"status"`
	Details string       `json:"details,omitempty"`
}

// Metrics carries runtime counters for a node.
// Field names and semantics are identical to relay.Metrics (§9) so that
// implementations can be swapped once the RELAY module exports this type.
type Metrics struct {
	WriteCount     uint64 `json:"write_count"`
	DeliverCount   uint64 `json:"deliver_count"`
	DropCount      uint64 `json:"drop_count"`
	BytesWritten   uint64 `json:"bytes_written"`
	BytesDelivered uint64 `json:"bytes_delivered"`
	ErrorCount     uint64 `json:"error_count"`
}

// HealthProvider is an optional interface; if implemented it MUST conform to
// RELAY spec §9.
type HealthProvider interface {
	Health() Health
}

// MetricsProvider is an optional interface; if implemented it MUST conform to
// RELAY spec §9.
type MetricsProvider interface {
	Metrics() Metrics
}

// Drainer extends any node with graceful shutdown. If implemented it MUST
// conform to RELAY spec §9.
type Drainer interface {
	CloseWithDrain(ctx context.Context) error
}
