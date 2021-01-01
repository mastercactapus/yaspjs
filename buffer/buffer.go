package buffer

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"
)

type Buffer struct {
	rwc io.ReadWriteCloser

	onRead   func(string)
	onUpdate func(CommandResponse)

	h Handler

	cfg FlowConfig

	readQ     *Queue
	writeQ    *Queue
	priorityQ *Queue
	onReadQ   *Queue
	onUpdateQ *Queue
	metaQ     *Queue
	ctrlCh    chan string

	mx sync.Mutex
}

type CommandResponse struct {
	QueueItem

	Queued bool
	Sent   bool
	Done   bool

	Err error
}

func NewBuffer(cfg Config) *Buffer {
	b := &Buffer{
		cfg: cfg.FlowConfig().WithDefaults(),
		rwc: cfg.ReadWriteCloser,
		h:   cfg.Handler,

		ctrlCh:    make(chan string),
		readQ:     NewQueue(),
		writeQ:    NewQueue(),
		priorityQ: NewQueue(),
		metaQ:     NewQueue(),

		onReadQ:   NewQueue(),
		onUpdateQ: NewQueue(),

		onRead:   cfg.OnRead,
		onUpdate: cfg.OnUpdate,
	}
	b.writeQ.SetCondition(func(item interface{}) bool { return b.h.CheckBuffer(item.(QueueItem).Data) })
	b.priorityQ.SetCondition(func(item interface{}) bool { return b.h.CheckBuffer(item.(QueueItem).Data) })

	go b.readLoop()
	go b.loop()
	go b.callbackLoop()

	if cfg.PollInterval > 0 {
		go b.pollLoop(cfg.PollInterval)
	}

	return b
}
func (b *Buffer) callbackLoop() {
	for {
		select {
		case line := <-b.onReadQ.Data():
			b.onRead(line.(string))
		case item := <-b.onUpdateQ.Data():
			b.onUpdate(item.(CommandResponse))
		}
	}
}

func (b *Buffer) pollLoop(itvl time.Duration) {
	t := time.NewTicker(itvl)
	defer t.Stop()

	for range t.C {
		cmd := b.h.PollCommand()
		if cmd == "" {
			continue
		}
		err := b.Queue("", cmd)
		if err != nil {
			// TODO: error
			panic(err)
		}
	}
}
func (b *Buffer) handleRead(line string) {
	b.onReadQ.Push(line + "\n")
	for _, resp := range b.h.HandleResponse(line) {
		b.onUpdateQ.Push(resp)
	}
}
func (b *Buffer) handleMeta(line string) {
	resp := b.h.HandleMeta(line)
	if resp != "" {
		b.onReadQ.Push(resp + "\n")
	}
}
func (b *Buffer) handleWrite(item QueueItem) {
	_, err := io.WriteString(b.rwc, item.Data)
	if err != nil {
		// TODO: error
		panic(err)
	}
	b.onUpdateQ.Push(CommandResponse{
		QueueItem: item,
		Sent:      true,
	})

	for _, resp := range b.h.HandleInput(item) {
		b.onUpdateQ.Push(resp)
	}
}

func (b *Buffer) loop() {

	for {
		b.priorityQ.ReCheck()
		b.writeQ.ReCheck()

		select {
		case chr := <-b.ctrlCh:
			b.handleWrite(QueueItem{Data: chr})
			continue
		case line := <-b.metaQ.Data():
			b.handleMeta(line.(string))
			continue
		case line := <-b.readQ.Data():
			b.handleRead(line.(string))
			continue
		default:
		}

		select {
		case chr := <-b.ctrlCh:
			b.handleWrite(QueueItem{Data: chr})
			continue
		case line := <-b.metaQ.Data():
			b.handleMeta(line.(string))
			continue
		case item := <-b.priorityQ.Data():
			b.handleWrite(item.(QueueItem))
			continue
		case line := <-b.readQ.Data():
			b.handleRead(line.(string))
			continue
		default:
		}

		select {
		case chr := <-b.ctrlCh:
			b.handleWrite(QueueItem{Data: chr})
		case item := <-b.priorityQ.Data():
			b.handleWrite(item.(QueueItem))
		case item := <-b.writeQ.Data():
			b.handleWrite(item.(QueueItem))
		case line := <-b.readQ.Data():
			b.handleRead(line.(string))
		case line := <-b.metaQ.Data():
			b.handleMeta(line.(string))
		}
	}
}

func (b *Buffer) readLoop() {
	r := bufio.NewScanner(b.rwc)
	if b.cfg.RecvSplitFunc != nil {
		r.Split(b.cfg.RecvSplitFunc)
	}
	for r.Scan() {
		if r.Text() == "" {
			continue
		}
		err := b.readQ.Push(r.Text())
		if err != nil {
			// TODO: error
			panic(err)
		}
	}
}

func (b *Buffer) Close() error {
	panic("not implemented")
}

func (b *Buffer) queueLine(item QueueItem) error {
	if b.cfg.IsMeta(item.Data) {
		return b.metaQ.Push(item.Data)
	}

	item.Data = b.cfg.WrapInput(item.Data)
	defer b.onUpdateQ.Push(CommandResponse{QueueItem: item, Queued: true})

	if b.cfg.IsControl(item.Data) {
		return b.priorityQ.Push(item)
	}

	return b.writeQ.Push(item)
}

func (b *Buffer) WriteQueueLen() int {
	return b.priorityQ.Len() + b.writeQ.Len()
}

func (b *Buffer) Queue(id, data string) error {
	ctrl, data := b.cfg.SplitControlChars(data)
	for _, chr := range ctrl {
		b.ctrlCh <- string(chr)
	}

	s := bufio.NewScanner(strings.NewReader(data))
	var lines []string
	for s.Scan() {
		if s.Text() == "" {
			continue
		}
		lines = append(lines, s.Text())
	}

	if len(lines) == 1 {
		return b.queueLine(QueueItem{ID: id, Data: lines[0]})
	}

	for i, line := range lines {
		err := b.queueLine(QueueItem{
			ID:     id,
			Seq:    i + 1,
			SeqMax: len(lines),
			Data:   line,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
