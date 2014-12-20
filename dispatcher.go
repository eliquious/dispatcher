package dispatcher

import (
	"fmt"
	disruptor "github.com/smartystreets/go-disruptor"
	"sync/atomic"
	"time"
)

// Config

type Config struct {
	BufferSize int64
}

// Metrics

type Metrics struct {
	totalEvents      uint64
	bytesTransferred uint64
	consumers        uint32
	producers        uint32
	processed        uint64
	elapsed          time.Duration
}

func (m *Metrics) IncrEvents(count uint64) uint64 {
	return atomic.AddUint64(&m.totalEvents, count)
}

func (m *Metrics) TotalEvents() uint64 {
	return m.totalEvents
}

func (m *Metrics) IncrTotalBytes(bytes uint64) uint64 {
	return atomic.AddUint64(&m.bytesTransferred, bytes)
}

func (m *Metrics) TotalBytes() uint64 {
	return m.bytesTransferred
}

func (m *Metrics) IncrConsumer() uint32 {
	return atomic.AddUint32(&m.consumers, 1)
}

func (m *Metrics) Consumers() uint32 {
	return m.consumers
}

func (m *Metrics) IncrProducer() uint32 {
	return atomic.AddUint32(&m.producers, 1)
}

func (m *Metrics) Producers() uint32 {
	return m.producers
}

func (m *Metrics) Elapsed() time.Duration {
	return m.elapsed
}

func (m *Metrics) IncrProcessed(done uint64) uint64 {
	return atomic.AddUint64(&m.processed, done)
}

func (m *Metrics) Processed() uint64 {
	return m.processed
}

type EndConsumer struct {
	disp *DefaultDispatcher
}

func (e EndConsumer) Consume(lower, upper int64) {
	// fmt.Println("End Consumer: ", lower, upper)
	(*e.disp).Metrics.IncrProcessed(uint64(upper+1) - uint64(lower))
}

func NewDispatcher(conf Config) Dispatcher {

	// set up config
	if conf.BufferSize < 0 {
		conf.BufferSize = int64(64 * 1024)
	}

	// create distruptor
	var shared disruptor.SharedDisruptor
	// var waitGroup sync.WaitGroup
	return &DefaultDispatcher{
		BufferSize: conf.BufferSize,
		BufferMask: conf.BufferSize - 1,
		Metrics:    Metrics{},
		ring:       make([]Event, conf.BufferSize),
		consumers:  make([]disruptor.Consumer, 0),
		controller: shared,
		completed:  make(chan bool),
		kill:       make(chan bool),
		running:    false,
		// producers:  make([]Worker, 0),
		// waitGroup:  waitGroup,
	}
}

type DefaultDispatcher struct {
	BufferSize int64
	BufferMask int64
	Metrics    Metrics
	ring       []Event
	consumers  []disruptor.Consumer
	controller disruptor.SharedDisruptor
	completed  chan bool
	kill       chan bool
	running    bool
	// producers  []Worker
	// waitGroup  sync.WaitGroup
}

func (d *DefaultDispatcher) GetMetrics() Metrics {
	return d.Metrics
}

func (d *DefaultDispatcher) Start() {

	// create disruptor
	d.controller = disruptor.Configure(d.BufferSize).
		WithConsumerGroup(d.consumers...).
		WithConsumerGroup(EndConsumer{d}).
		BuildShared()
	d.controller.Start()
}

func (d *DefaultDispatcher) Stop() {

	// Stopping all handlers
	d.controller.Stop()

	// send poison pill for each producer
	var i uint32 = 0
	for ; i < d.Metrics.Producers(); i++ {
		d.kill <- true
	}

}

// Add consumer
func (d *DefaultDispatcher) RouteConsumer(c Consumer) {

	// add consumer
	d.consumers = append(d.consumers, c)

	// increment metrics
	d.Metrics.IncrConsumer()
}

func (d *DefaultDispatcher) Route(channel string, handler Handler) {

	// create predicated consumer
	consumer := PredicatedConsumer{d, EqualChannelPredicate{[]byte(channel)}, handler}

	// add consumer
	d.RouteConsumer(consumer)
}

func (d *DefaultDispatcher) RouteFunc(channel string, handler RouteFunc) {

	// Add route
	d.Route(channel, DefaultHandler{handler})
}

func (d *DefaultDispatcher) Emit(evt Event) {

	// get ring writer
	writer := d.controller.Writer()

	// reserve place in ring
	sequence := writer.Reserve(1)

	// write to ring
	d.ring[sequence&d.BufferMask] = evt
	writer.Commit(sequence, sequence)

	// update metrics
	d.Metrics.IncrEvents(1)
	d.Metrics.IncrTotalBytes(uint64(len(evt.Data())))
}

func (d *DefaultDispatcher) Batch(evts []Event) {

	// get ring writer
	writer := d.controller.Writer()

	// reserve multiple places in the ring
	reservations := int64(len(evts))
	sequence := writer.Reserve(reservations)

	// write all events to ring
	var idx, bytes int
	for lower := sequence - reservations + 1; lower <= sequence; lower++ {
		d.ring[lower&d.BufferMask] = evts[idx]

		// increment metrics
		bytes += len(evts[idx].Data())
		idx++
	}

	// commit events to ring buffer
	writer.Commit(sequence-reservations+1, sequence)

	// metrics
	d.Metrics.IncrEvents(uint64(len(evts)))
	d.Metrics.IncrTotalBytes(uint64(bytes))
}

// Returns an offset in the buffer
func (d *DefaultDispatcher) Get(lower int64) Event {
	return d.ring[lower&d.BufferMask]
}

// Sets an offset in the buffer
func (d *DefaultDispatcher) Set(lower int64, evt Event) {
	d.ring[lower&d.BufferMask] = evt
}

// Wait for all processing to be completed
func (d *DefaultDispatcher) Join() {

	numProducers := d.Metrics.Producers()

	for {
		select {
		case <-d.completed:
			numProducers--
		default:
			// fmt.Println("JOIN: Sleeping: ", d.Metrics.Processed(), " : ", d.Metrics.TotalEvents())
			if d.Metrics.Processed() == d.Metrics.TotalEvents() && numProducers <= 0 {
				fmt.Println("Closing Joiner")

				// Stopping consumers
				d.Stop()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}
}

func (d *DefaultDispatcher) Register(w Work) {

	worker := func(completed chan bool, kill chan bool) {

		go func(done chan bool, work Work) {
			work.Start()
			done <- true
		}(completed, w)

		for {
			select {
			case <-kill:
				w.Stop()
				completed <- true
			default:
				time.Sleep(time.Microsecond)
			}
		}
	}
	d.RegisterFunc(worker)
}

func (d *DefaultDispatcher) RegisterFunc(worker Worker) {
	go worker(d.completed, d.kill)
	d.Metrics.IncrProducer()
}

func (d *DefaultDispatcher) IsRunning() bool {
	return d.running
}
