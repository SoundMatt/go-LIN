// Copyright (c) 2026 Matt Jones. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package virtual_test

import (
	"context"
	"errors"
	"testing"

	lin "github.com/SoundMatt/go-LIN"
	"github.com/SoundMatt/go-LIN/virtual"
)

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

//fusa:test REQ-VIRT-002

func TestPublish_andSendHeader(t *testing.T) {
	b, _ := virtual.New()
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
	if f.ID != 0x10 {
		t.Errorf("frame ID = 0x%02X, want 0x10", f.ID)
	}
	for i, b := range want {
		if f.Data[i] != b {
			t.Errorf("data[%d] = 0x%02X, want 0x%02X", i, f.Data[i], b)
		}
	}
}

func TestPublish_removeResponse(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	_ = b.Publish(0x10, []byte{0x01})
	_ = b.Publish(0x10, nil) // remove
	_, err := b.SendHeader(context.Background(), 0x10)
	if !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("expected ErrNoResponse, got %v", err)
	}
}

func TestPublish_invalidID(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()
	if err := b.Publish(0x40, []byte{1}); err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

//fusa:test REQ-VIRT-003

func TestSendHeader_noResponse(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	_, err := b.SendHeader(context.Background(), 0x20)
	if !errors.Is(err, lin.ErrNoResponse) {
		t.Errorf("expected ErrNoResponse, got %v", err)
	}
}

func TestSendHeader_invalidID(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()
	_, err := b.SendHeader(context.Background(), 0xFF)
	if err == nil {
		t.Error("expected error for ID > MaxID")
	}
}

func TestSendHeader_checksumEnhanced(t *testing.T) {
	b, _ := virtual.New()
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

func TestSendHeader_checksumClassic(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	data := []byte{0x11, 0x22}
	_ = b.PublishClassic(0x05, data)

	f, err := b.SendHeader(context.Background(), 0x05)
	if err != nil {
		t.Fatalf("SendHeader: %v", err)
	}
	if f.ChecksumType != lin.ClassicChecksum {
		t.Errorf("ChecksumType = %v, want ClassicChecksum", f.ChecksumType)
	}
}

//fusa:test REQ-VIRT-004

func TestSubscribe_receivesFrame(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	ch, err := b.Subscribe(lin.Filter{ID: 0x10})
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
		t.Error("expected frame on subscriber channel, got nothing")
	}
}

func TestSubscribe_filteredOut(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	ch, _ := b.Subscribe(lin.Filter{ID: 0x20}) // not 0x10

	_ = b.Publish(0x10, []byte{0xFF})
	_, _ = b.SendHeader(context.Background(), 0x10)

	select {
	case <-ch:
		t.Error("subscriber for 0x20 received frame for 0x10")
	default:
	}
}

func TestSubscribe_allFilter(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	ch, _ := b.Subscribe(lin.Filter{All: true})

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
		t.Errorf("all-filter subscriber received %d frames, want 2", count)
	}
}

func TestSubscribe_multipleSubscribers(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	ch1, _ := b.Subscribe(lin.Filter{ID: 0x10})
	ch2, _ := b.Subscribe(lin.Filter{All: true})

	_ = b.Publish(0x10, []byte{0xAB})
	_, _ = b.SendHeader(context.Background(), 0x10)

	for _, ch := range []<-chan lin.Frame{ch1, ch2} {
		select {
		case f := <-ch:
			if f.ID != 0x10 {
				t.Errorf("got frame ID 0x%02X, want 0x10", f.ID)
			}
		default:
			t.Error("subscriber did not receive frame")
		}
	}
}

//fusa:test REQ-VIRT-005

func TestClose_idempotent(t *testing.T) {
	b, _ := virtual.New()
	if err := b.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := b.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestClose_closesSubscriberChannels(t *testing.T) {
	b, _ := virtual.New()
	ch, _ := b.Subscribe(lin.Filter{All: true})
	b.Close()

	_, open := <-ch
	if open {
		t.Error("subscriber channel should be closed after bus.Close()")
	}
}

func TestClose_publishAfterClose(t *testing.T) {
	b, _ := virtual.New()
	b.Close()
	if err := b.Publish(0x10, []byte{1}); err == nil {
		t.Error("expected error from Publish after Close")
	}
}

func TestClose_sendHeaderAfterClose(t *testing.T) {
	b, _ := virtual.New()
	b.Close()
	if _, err := b.SendHeader(context.Background(), 0x10); err == nil {
		t.Error("expected error from SendHeader after Close")
	}
}

func TestConcurrent(t *testing.T) {
	b, _ := virtual.New()
	defer b.Close()

	const workers = 8
	_ = b.Publish(0x01, []byte{0x01, 0x02})

	done := make(chan struct{})
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

func FuzzSendHeader(f *testing.F) {
	f.Add(uint8(0x10), []byte{0x01, 0x02, 0x03, 0x04})
	f.Add(uint8(0x00), []byte{0xFF})
	f.Add(uint8(0x3F), []byte{0xAA, 0x55, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66})

	f.Fuzz(func(t *testing.T, id uint8, data []byte) {
		b, _ := virtual.New()
		defer b.Close()
		if id > lin.MaxID || len(data) == 0 || len(data) > lin.MaxDataLen {
			return
		}
		_ = b.Publish(id, data)
		_, _ = b.SendHeader(context.Background(), id)
	})
}
