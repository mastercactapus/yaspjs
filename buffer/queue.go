package buffer

import (
	"errors"
	"fmt"
)

// A Queue is used to manage a set of commands.
type Queue struct {
	push      chan interface{}
	reset     chan chan []interface{}
	filter    chan queueFilterReq
	unshift   chan interface{}
	items     chan interface{}
	data      chan string
	len       chan int
	byteLen   chan int
	condition chan func(interface{}) bool
	recheck   chan struct{}
	buffer    chan []interface{}

	reqClose chan struct{}
	close    chan struct{}
}

type queueFilterReq struct {
	filter func(interface{}) bool
	resp   chan []interface{}
}

// A QueueItem contains data and an optional ID to be stored in the Queue.
type QueueItem struct {
	// ID is an arbitrary string that can be used by the caller to identify a queued item.
	ID string

	Data string

	Seq, SeqMax int
}

type ByteLenable interface {
	ByteLen() int
}

func (q QueueItem) ByteLen() int { return len(q.Data) }

func calcByteLen(item interface{}) int {
	switch t := item.(type) {
	case ByteLenable:
		return t.ByteLen()
	case string:
		return len(t)
	case []byte:
		return len(t)
	default:
		panic(fmt.Sprintf("unsupported type: %T", t))
	}
}

// ErrClosed is returned for write operations after Close has been called.
var ErrClosed = errors.New("closed")

// NewQueue will return a new non-blocking Queue.
func NewQueue() *Queue {
	q := &Queue{
		push:    make(chan interface{}),
		reset:   make(chan chan []interface{}),
		filter:  make(chan queueFilterReq),
		unshift: make(chan interface{}),
		items:   make(chan interface{}),
		data:    make(chan string),
		len:     make(chan int),
		byteLen: make(chan int),
		recheck: make(chan struct{}),
		buffer:  make(chan []interface{}),

		condition: make(chan func(interface{}) bool),
	}
	go q.loop()
	return q
}

func (q *Queue) loop() {
	defer close(q.len)
	defer close(q.byteLen)
	defer close(q.data)
	defer close(q.buffer)
	defer close(q.close)

	var buf []interface{}
	var byteLen int
	conditional := func(interface{}) bool { return true }

	reset := func(ch chan []interface{}) {
		byteLen = 0
		removed := make([]interface{}, len(buf))
		copy(removed, buf)
		buf = buf[:0]
		ch <- removed
	}

	filter := func(req queueFilterReq) {
		filtered := buf[:0]
		var removed []interface{}
		for _, data := range buf {
			if !req.filter(data) {
				removed = append(removed, data)
				continue
			}
			filtered = append(filtered, data)
		}
		buf = filtered
		req.resp <- removed
	}

	unshift := func(item interface{}) {
		if len(buf) == 0 {
			buf = append(buf, item)
		} else {
			buf = append(buf[:1], buf...)
			buf[0] = item
		}
		byteLen += calcByteLen(item)
	}

	for {
		if len(buf) == 0 || !conditional(buf[0]) {
			select {
			case q.buffer <- buf:
			case <-q.recheck:
			case cond := <-q.condition:
				conditional = cond
			case item := <-q.push:
				buf = append(buf, item)
				byteLen += calcByteLen(item)
			case ch := <-q.reset:
				reset(ch)
			case req := <-q.filter:
				filter(req)
			case item := <-q.unshift:
				unshift(item)
			case q.byteLen <- byteLen:
			case q.len <- len(buf):
			case <-q.reqClose:
				return
			}
			continue
		}

		select {
		case q.buffer <- buf:
		case <-q.recheck:
		case cond := <-q.condition:
			conditional = cond
		case data := <-q.push:
			buf = append(buf, data)
			byteLen += calcByteLen(data)
		case ch := <-q.reset:
			reset(ch)
		case req := <-q.filter:
			filter(req)
		case item := <-q.unshift:
			unshift(item)
		case q.items <- buf[0]:
			byteLen -= calcByteLen(buf[0])
			buf = buf[1:]
		case q.byteLen <- byteLen:
		case q.len <- len(buf):
		case <-q.reqClose:
			return
		}
	}
}

// ReCheck forces the conditional to be re-evaluated.
func (q *Queue) ReCheck() {
	select {
	case <-q.close:
	case q.recheck <- struct{}{}:
	}
}

// SetCondition will set a function that must return true before data will be returned from Shift or a channel.
func (q *Queue) SetCondition(fn func(interface{}) bool) error {
	if fn == nil {
		fn = func(interface{}) bool { return true }
	}
	select {
	case <-q.close:
		return ErrClosed
	case q.condition <- fn:
	}
	return nil
}

// Close will terminate and cleanup the Queue. All write methods will return `ErrClosed`
// and read methods will return empty values.
func (q *Queue) Close() error {
	select {
	case <-q.close:
		return ErrClosed
	case q.reqClose <- struct{}{}:
	}
	return nil
}

// Push will append an item to the end of the Queue.
func (q *Queue) Push(data interface{}) error {
	select {
	case <-q.close:
		return ErrClosed
	case q.push <- data:
	}
	return nil
}

// Reset will empty the Queue returning a slice of the removed items.
func (q *Queue) Reset() []interface{} {
	ch := make(chan []interface{}, 1)
	select {
	case <-q.close:
		return nil
	case q.reset <- ch:
	}

	return <-ch
}

// Filter will remove all items from the Queue that `filterFn` returns false for, returning
// any items removed.
func (q *Queue) Filter(filterFn func(interface{}) bool) []interface{} {
	ch := make(chan []interface{}, 1)
	select {
	case <-q.close:
		return nil
	case q.filter <- queueFilterReq{filter: filterFn, resp: ch}:
	}
	return <-ch
}

// UnShift will prepend an item to the start of the Queue.
func (q *Queue) UnShift(data interface{}) error {
	select {
	case <-q.close:
		return ErrClosed
	case q.unshift <- data:
	}
	return nil
}

// Buffer returns the internal buffer state.
func (q *Queue) Buffer() []interface{} { return <-q.buffer }

// Shift will return and remove the first item in the Queue. If empty, it will
// block until data is added, or the Queue is closed.
//
// It is equivelant to `<-q.Data()`.
func (q *Queue) Shift() interface{} { return <-q.items }

// Data will return a channel that is fed (and consumes) items from the Queue.
func (q *Queue) Data() <-chan interface{} { return q.items }

// ByteLen will return the number of bytes currently in the Queue.
func (q *Queue) ByteLen() int { return <-q.byteLen }

// Len will return the number of items currently in the Queue.
func (q *Queue) Len() int { return <-q.len }
