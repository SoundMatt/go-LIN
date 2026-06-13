// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Command quickstart demonstrates go-LIN's virtual in-process bus.
//
// A slave goroutine publishes synthetic window-position frames;
// a master goroutine drives the schedule and prints each received frame.
//
// This is the Docker quickstart entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/master"
	"github.com/SoundMatt/go-LIN/slave"
	"github.com/SoundMatt/go-LIN/virtual"
)

const (
	windowID  = 0x10 // window position frame
	mirrorID  = 0x11 // mirror angle frame
	intervalMs = 500
)

func main() {
	bus, err := virtual.New()
	if err != nil {
		log.Fatalf("virtual.New: %v", err)
	}
	defer bus.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Slave: publish window position and mirror angle
	slaveNode := slave.New(bus)
	go slaveTask(ctx, slaveNode)

	// Master: drive the schedule
	m := master.New(bus)
	if err := m.SetSchedule([]lin.ScheduleEntry{
		{ID: windowID, DelayMs: uint32(intervalMs)},
		{ID: mirrorID, DelayMs: uint32(intervalMs)},
	}); err != nil {
		log.Fatalf("SetSchedule: %v", err)
	}

	m.OnFrame(func(f lin.Frame) {
		switch f.ID {
		case windowID:
			pos := f.Data[0]
			fmt.Printf("RX  window_pos  %02X#%02X   pos=%d%%\n", f.ID, pos, pos)
		case mirrorID:
			angle := int8(f.Data[0])
			fmt.Printf("RX  mirror_angle %02X#%02X  angle=%d°\n", f.ID, uint8(angle), angle)
		}
	})
	m.OnError(func(err error) {
		log.Printf("master error: %v", err)
	})

	hostname, _ := os.Hostname()
	log.Printf("[%s] go-LIN quickstart — virtual bus demo", hostname)
	log.Printf("Schedule: window_pos(0x%02X) + mirror_angle(0x%02X) every %dms", windowID, mirrorID, intervalMs)

	if err := m.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("master.Run: %v", err)
	}
}

func slaveTask(ctx context.Context, s *slave.Node) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var tick int
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate window position sweeping 0–100%
			pos := uint8(50 + 50*math.Sin(float64(tick)/10.0))
			if err := s.SetResponse(windowID, []byte{pos}); err != nil {
				log.Printf("slave SetResponse windowID: %v", err)
			}

			// Simulate mirror angle sweeping ±45°
			angle := int8(45 * math.Sin(float64(tick)/7.0))
			if err := s.SetResponse(mirrorID, []byte{uint8(angle)}); err != nil {
				log.Printf("slave SetResponse mirrorID: %v", err)
			}
			tick++
		}
	}
}
