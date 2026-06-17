// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package virtual_test

import (
	"context"
	"errors"
	"testing"
	"time"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

// ── New ───────────────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-011
//fusa:test REQ-LIN-012
//fusa:test REQ-LIN-013
//fusa:test REQ-LIN-014
//fusa:test REQ-VIRT-001

func TestNew(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

// ── Publish ───────────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-019
//fusa:test REQ-VIRT-002
//fusa:test REQ-VIRT-003
//fusa:test REQ-VIRT-004
//fusa:test REQ-VIRT-005
//fusa:test REQ-VIRT-019

func TestPublish_storesResponse(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	want := []byte{0x01, 0x02, 0x03, 0x04}
	if err := b.Publish(0x10, want); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	ctx := context.Background()
	f, err := b.SendHeader(ctx, 0x10)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	for i, v := range want {
		if f.Data[i] != v {
			t.Errorf("data[%d] = 0x%02X, want 0x%02X", i, f.Data[i], v)
		}
	}
}

func TestPublish_removesResponse(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	_ = b.Publish(0x10, []byte{0x01})
	_ = b.Publish(0x10, nil)
	_, err = b.SendHeader(context.Background(), 0x10)
	if !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("expected ErrNoResponse after nil publish, got %v", err)
	}
}

func TestPublish_rejectsHighID(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err := b.Publish(0x40, []byte{1}); err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

func TestPublish_afterClose(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
	if err := b.Publish(0x10, []byte{1}); err == nil {
		t.Error("expected error from Publish after Close")
	}
}

// ── SendHeader ────────────────────────────────────────────────────────────────

//fusa:test REQ-VIRT-006
//fusa:test REQ-VIRT-007
//fusa:test REQ-VIRT-008
//fusa:test REQ-VIRT-009
//fusa:test REQ-VIRT-010

func TestSendHeader_noResponse(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	_, err = b.SendHeader(context.Background(), 0x20)
	if !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("expected ErrNoResponse, got %v", err)
	}
}

func TestSendHeader_rejectsHighID(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if _, err := b.SendHeader(context.Background(), 0xFF); err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

func TestSendHeader_setsCorrectPID(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	_ = b.Publish(0x12, []byte{0x01})
	f, err := b.SendHeader(context.Background(), 0x12)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.ID != 0x12 {
		t.Errorf("frame ID = 0x%02X, want 0x12", f.ID)
	}
}

func TestSendHeader_enhancedChecksum(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	data := []byte{0xAA, 0x55}
	_ = b.Publish(0x12, data)
	f, err := b.SendHeader(context.Background(), 0x12)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	pid := lin.ProtectID(0x12)
	wantCS := lin.CalcChecksum(pid, data, lin.EnhancedChecksum)
	if f.Checksum != wantCS {
		t.Errorf("checksum = 0x%02X, want 0x%02X", f.Checksum, wantCS)
	}
	if f.ChecksumType != lin.EnhancedChecksum {
		t.Errorf("ChecksumType = %v, want EnhancedChecksum", f.ChecksumType)
	}
}

func TestSendHeader_classicChecksum(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	_ = b.PublishClassic(0x05, []byte{0x11, 0x22})
	f, err := b.SendHeader(context.Background(), 0x05)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.ChecksumType != lin.ClassicChecksum {
		t.Errorf("ChecksumType = %v, want ClassicChecksum", f.ChecksumType)
	}
}

func TestSendHeader_broadcastsToSubscriber(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ch, err := b.Subscribe([]lin.Filter{{ID: 0x10}})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	_ = b.Publish(0x10, []byte{0xFF})
	_, _ = b.SendHeader(context.Background(), 0x10)

	select {
	case f := <-ch:
		if f.ID != 0x10 {
			t.Errorf("received frame ID 0x%02X, want 0x10", f.ID)
		}
	default:
		t.Error("expected frame on subscriber channel")
	}
}

// ── Subscribe ─────────────────────────────────────────────────────────────────

//fusa:test REQ-LIN-020
//fusa:test REQ-VIRT-011
//fusa:test REQ-VIRT-012
//fusa:test REQ-VIRT-013
//fusa:test REQ-VIRT-014

func TestSubscribe_exactFilterIsolation(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ch, _ := b.Subscribe([]lin.Filter{{ID: 0x20}})
	_ = b.Publish(0x10, []byte{0xFF})
	_, _ = b.SendHeader(context.Background(), 0x10)

	select {
	case <-ch:
		t.Error("subscriber for 0x20 must not receive frame for 0x10")
	default:
	}
}

func TestSubscribe_allFilter(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ch, _ := b.Subscribe([]lin.Filter{{All: true}})
	_ = b.Publish(0x01, []byte{0x01})
	_ = b.Publish(0x02, []byte{0x02})
	_, _ = b.SendHeader(context.Background(), 0x01)
	_, _ = b.SendHeader(context.Background(), 0x02)

	count := 0
	for i := 0; i < 2; i++ {
		select {
		case <-ch:
			count++
		default:
		}
	}
	if count != 2 {
		t.Errorf("all-filter received %d frames, want 2", count)
	}
}

func TestSubscribe_multipleSubscribersIndependent(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	ch1, _ := b.Subscribe([]lin.Filter{{ID: 0x10}})
	ch2, _ := b.Subscribe([]lin.Filter{{All: true}})

	_ = b.Publish(0x10, []byte{0xAB})
	_, _ = b.SendHeader(context.Background(), 0x10)

	for i, ch := range []<-chan lin.Frame{ch1, ch2} {
		select {
		case f := <-ch:
			if f.ID != 0x10 {
				t.Errorf("subscriber %d: got frame ID 0x%02X, want 0x10", i, f.ID)
			}
		default:
			t.Errorf("subscriber %d: did not receive frame", i)
		}
	}
}

// ── Close ─────────────────────────────────────────────────────────────────────

//fusa:test REQ-VIRT-015
//fusa:test REQ-VIRT-016
//fusa:test REQ-VIRT-017

func TestClose_closesSubscriberChannels(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	ch, _ := b.Subscribe([]lin.Filter{{All: true}})
	b.Close()
	_, open := <-ch
	if open {
		t.Error("subscriber channel must be closed after bus.Close()")
	}
}

func TestClose_idempotent(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestClose_sendHeaderAfterClose(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
	if _, err := b.SendHeader(context.Background(), 0x10); err == nil {
		t.Error("expected error from SendHeader after Close")
	}
}

//fusa:test REQ-VIRT-018

func TestConcurrent(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	const workers = 8
	_ = b.Publish(0x01, []byte{0x01, 0x02})

	done := make(chan struct{}, workers)
	for i := 0; i < workers; i++ {
		go func() {
			ctx := context.Background()
			for j := 0; j < 50; j++ {
				_, _ = b.SendHeader(ctx, 0x01)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < workers; i++ {
		<-done
	}
}

func TestPublish_defensiveCopy(t *testing.T) {
	b, err := virtual.New()
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	data := []byte{0x01, 0x02, 0x03}
	_ = b.Publish(0x10, data)
	data[0] = 0xFF // mutate caller's slice

	f, err := b.SendHeader(context.Background(), 0x10)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.Data[0] == 0xFF {
		t.Error("stored response was mutated by caller: defensive copy not taken")
	}
}

func FuzzSendHeader(f *testing.F) {
	f.Add(uint8(0x10), []byte{0x01, 0x02, 0x03, 0x04})
	f.Add(uint8(0x00), []byte{0xFF})
	f.Add(uint8(0x3F), []byte{0xAA, 0x55, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66})

	f.Fuzz(func(t *testing.T, id uint8, data []byte) {
		b, err := virtual.New()
		if err != nil {
			t.Fatal(err)
		}
		defer b.Close()
		if id > lin.MaxID || len(data) == 0 || len(data) > lin.MaxDataLen {
			return
		}
		_ = b.Publish(id, data)
		_, _ = b.SendHeader(context.Background(), id)
	})
}

// ── Health (optional interface) ───────────────────────────────────────────────

func TestHealth_openBus(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()
	h := b.Health()
	if h.Status != lin.HealthOK {
		t.Errorf("Health: expected HealthOK, got %v", h.Status)
	}
}

func TestHealth_closedBus(t *testing.T) {
	b, _ := virtual.New()
	b.Close()
	h := b.Health()
	if h.Status != lin.HealthDown {
		t.Errorf("Health after Close: expected HealthDown, got %v", h.Status)
	}
}

// ── Metrics (optional interface) ──────────────────────────────────────────────

func TestMetrics_countsFrames(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	_ = b.Publish(0x10, []byte{0xAA, 0xBB})
	ch, _ := b.Subscribe([]lin.Filter{{ID: 0x10}})
	_, _ = b.SendHeader(context.Background(), 0x10)

	// Drain the subscriber channel so deliver count is updated.
	<-ch

	m := b.Metrics()
	if m.WriteCount != 1 {
		t.Errorf("WriteCount = %d, want 1", m.WriteCount)
	}
	if m.DeliverCount != 1 {
		t.Errorf("DeliverCount = %d, want 1", m.DeliverCount)
	}
	if m.BytesWritten != 2 {
		t.Errorf("BytesWritten = %d, want 2", m.BytesWritten)
	}
}

// ── CloseWithDrain (optional interface) ───────────────────────────────────────

func TestCloseWithDrain_drainsThenCloses(t *testing.T) {
	b, _ := virtual.New()

	_ = b.Publish(0x01, []byte{0x01})
	ch, _ := b.Subscribe([]lin.Filter{{All: true}})
	_, _ = b.SendHeader(context.Background(), 0x01)

	// Drain the channel, then close.
	<-ch

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := b.CloseWithDrain(ctx); err != nil {
		t.Fatalf("CloseWithDrain: %v", err)
	}
	_, open := <-ch
	if open {
		t.Error("subscriber channel must be closed after CloseWithDrain")
	}
}
