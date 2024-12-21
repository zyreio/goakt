/*
 * MIT License
 *
 * Copyright (c) 2022-2024  Arsene Tochemey Gandote
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package actors

import (
	"sync"

	gods "github.com/Workiva/go-datastructures/queue"
)

// BoundedMailbox defines a bounded mailbox using ring buffer queue
// This mailbox is thread-safe
type BoundedMailbox struct {
	underlying   *gods.RingBuffer
	waitingMutex sync.Mutex
	waitingEmpty []chan struct{}
}

// enforce compilation error
var _ Mailbox = (*BoundedMailbox)(nil)

// NewBoundedMailbox creates a new instance BoundedMailbox
func NewBoundedMailbox(capacity int) *BoundedMailbox {
	return &BoundedMailbox{
		underlying:   gods.NewRingBuffer(uint64(capacity)),
		waitingEmpty: make([]chan struct{}, 0),
	}
}

// Enqueue places the given value in the mailbox
// This will return an error when the mailbox is full
func (mailbox *BoundedMailbox) Enqueue(msg *ReceiveContext) error {
	return mailbox.underlying.Put(msg)
}

// Dequeue takes the mail from the mailbox
// It returns nil when the mailbox is empty
func (mailbox *BoundedMailbox) Dequeue() (msg *ReceiveContext) {
	defer mailbox.notifyWaitingEmpty()
	if mailbox.underlying.Len() > 0 {
		item, _ := mailbox.underlying.Get()
		return item.(*ReceiveContext)
	}
	return nil
}

func (mailbox *BoundedMailbox) notifyWaitingEmpty() {
	mailbox.waitingMutex.Lock()
	defer mailbox.waitingMutex.Unlock()
	if mailbox.IsEmpty() && len(mailbox.waitingEmpty) > 0 {
		for _, ch := range mailbox.waitingEmpty {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		mailbox.waitingEmpty = make([]chan struct{}, 0)
	}
}

func (mailbox *BoundedMailbox) WaitEmpty() chan struct{} {
	mailbox.waitingMutex.Lock()
	defer mailbox.waitingMutex.Unlock()
	ch := make(chan struct{})
	mailbox.waitingEmpty = append(mailbox.waitingEmpty, ch)
	return ch
}

// IsEmpty returns true when the mailbox is empty
func (mailbox *BoundedMailbox) IsEmpty() bool {
	return mailbox.underlying.Len() == 0
}

func (mailbox *BoundedMailbox) IsFull() bool {
	return mailbox.underlying.Len() == mailbox.underlying.Cap()
}

// Len returns queue length
func (mailbox *BoundedMailbox) Len() int64 {
	return int64(mailbox.underlying.Len())
}

// Dispose will dispose of this queue and free any blocked threads
// in the Enqueue and/or Dequeue methods.
func (mailbox *BoundedMailbox) Dispose() {
	mailbox.underlying.Dispose()
}
