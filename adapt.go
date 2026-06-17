// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package lin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	relay "github.com/SoundMatt/RELAY"
)

// Adapt wraps bus as a relay.Node for use with protocol-agnostic applications.
// Adapt does not block; it does not connect.
func Adapt(bus Bus) relay.Node {
	return &linNode{bus: bus}
}

type linNode struct {
	bus Bus
}

func (n *linNode) Protocol() relay.Protocol {
	return relay.LIN
}

// Send publishes msg.Payload as a slave response for the frame ID in msg.ID.
// msg.ID must be a decimal string in range 0–63.
func (n *linNode) Send(ctx context.Context, msg relay.Message) error {
	id, err := strconv.ParseUint(msg.ID, 10, 8)
	if err != nil || id > LINMaxID {
		return fmt.Errorf("lin: invalid frame ID %q: %w", msg.ID, ErrNotConnected)
	}
	return n.bus.Publish(uint8(id), msg.Payload)
}

// Subscribe returns a channel of relay.Message envelopes for all received frames.
func (n *linNode) Subscribe(opts ...relay.SubscriberOption) (<-chan relay.Message, error) {
	cfg := relay.ApplySubscriberOpts(opts)
	ch := make(chan relay.Message, cfg.ChanDepth(64))
	frames, err := n.bus.Subscribe(nil)
	if err != nil {
		return nil, err
	}
	go func() {
		defer close(ch)
		var seq uint64
		for f := range frames {
			msg := f.ToMessage()
			msg.Timestamp = time.Now().UTC()
			msg.Seq = seq
			seq++
			switch cfg.BackPressure {
			case relay.DropNewest:
				select {
				case ch <- msg:
				default:
				}
			case relay.DropOldest:
				select {
				case ch <- msg:
				default:
					<-ch
					ch <- msg
				}
			case relay.Block:
				ch <- msg
			}
		}
	}()
	return ch, nil
}

func (n *linNode) Close() error {
	return n.bus.Close()
}
