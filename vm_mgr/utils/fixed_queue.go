/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package utils

import (
	"context"
	"errors"
)

const (
	WaitForNextElementChanCapacity = 4000
)

// FixedFIFO Fixed capacity FIFO (First In First Out) concurrent queue
type FixedFIFO struct {
	queue chan interface{}
	// queue for watchers that will wait for next elements (if queue is empty at DequeueOrWaitForNextElement execution )
	waitForNextElementChan chan chan interface{}
}

func NewFixedFIFO(capacity int) *FixedFIFO {
	queue := &FixedFIFO{}
	queue.initialize(capacity)

	return queue
}

func (st *FixedFIFO) initialize(capacity int) {
	st.queue = make(chan interface{}, capacity)
	st.waitForNextElementChan = make(chan chan interface{}, WaitForNextElementChanCapacity)
}

// Enqueue enqueues an element. Returns error if queue is locked or it is at full capacity.
func (st *FixedFIFO) Enqueue(value interface{}) error {

	// check if there is a listener waiting for the next element (this element)
	select {
	case listener := <-st.waitForNextElementChan:
		// send the element through the listener's channel instead of enqueue it
		listener <- value

	default:
		// enqueue the element following the "normal way"
		select {
		case st.queue <- value:
		default:
			return errors.New("FixedFIFO queue is at full capacity")
		}
	}

	return nil
}

// DequeueOrWaitForNextElement dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
// Multiple calls to DequeueOrWaitForNextElement() would enqueue multiple "listeners" for future enqueued elements.
func (st *FixedFIFO) DequeueOrWaitForNextElement() (interface{}, error) {
	return st.DequeueOrWaitForNextElementContext(context.Background())
}

// DequeueOrWaitForNextElementContext dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
// Multiple calls to DequeueOrWaitForNextElementContext() would enqueue multiple "listeners" for future enqueued elements.
// When the passed context expires this function exits and returns the context' error
func (st *FixedFIFO) DequeueOrWaitForNextElementContext(ctx context.Context) (interface{}, error) {
	select {
	case value, ok := <-st.queue:
		if ok {
			return value, nil
		}
		return nil, errors.New("internal channel is closed")
	case <-ctx.Done():
		return nil, ctx.Err()
	// queue is empty, add a listener to wait until next enqueued element is ready
	default:
		// channel to wait for next enqueued element
		waitChan := make(chan interface{})

		select {
		// enqueue a watcher into the watchForNextElementChannel to wait for the next element
		case st.waitForNextElementChan <- waitChan:
			// return the next enqueued element, if any
			select {
			case item := <-waitChan:
				return item, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		default:
			// too many watchers (waitForNextElementChanCapacity) enqueued waiting for next elements
			return nil, errors.New("empty queue and can't wait for next element")
		}
	}
}
