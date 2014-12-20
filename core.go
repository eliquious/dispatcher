package dispatcher

// Event is the core message passing structure.
// This interface defines the message structure needed by the handlers.
//
// Each event must have an associated channel and a byte array as content.
//
// Requiring a byte array as the data content may seem awkward in some situations,
// however, it is also the most generic data structure.
type Event interface {
	Channel() []byte
	Data() []byte
}

// Emitter interface is fulfilled by the Dispacher
// and acts as an entry point into the Dispatcher.
type Emitter interface {
	Emit(evt Event)
	Batch(es []Event)
}

// RouteFunc is simply any function which takes in a single Event.
type RouteFunc func(Event)

// Handler allows for more complex interactions than a RouteFunc.
type Handler interface {
	Handle(evt Event)
}

type BatchHandler interface {
	Handler

	StartBatch()
	EndBatch()
}

// Consumer is a low-level interface which directly hooks
// into the underlying Disruptor implementation.
//
// This interface allow for very fine-grained control over how events are handled.
type Consumer interface {
	Consume(lower, upper int64)
}

// Router is used by the Dispatcher to maintain a list of all the existing event handlers.
type Router interface {
	Route(channel string, h Handler)
	RouteFunc(channel string, h RouteFunc)
	RouteConsumer(c Consumer)
}

// Registry simply allows the Dispatcher to track all existing producers
type Registry interface {
	Register(w Work)
	RegisterFunc(w Worker)
}

// PreProcessor
type PreProcessor interface {
	PreProcess(evt Event)
}

// Processor
type Processor interface {
	Process(evt Event)
}

// PostProcessor
type PostProcessor interface {
	PostProcess(evt Event)
}

// StagedProcessor is meant to be used as a more complex
// consumer which requires several stages of processing.
type StagedProcessor interface {
	PreProcessor
	Processor
	PostProcessor
}

// Work is a simple interface which ony has 3 methods: Start(), Stop() and IsRunning()
type Work interface {
	Start()
	Stop()
	IsRunning() bool
}

// Base worker type
type Worker func(completed chan bool, kill chan bool)

// Predicate is a simple interface simply used to filter events before they are sent to a Handler.
type Predicate interface {
	Apply(evt Event) bool
}

// Joiner is another simple interface employed by the Dispatcher which
// fulfilles the purpose of waiting on all the producers to finish processing.
type Joiner interface {
	Join()
}

// Dispatcher handles all the event traffic and management.
type Dispatcher interface {
	Work
	Router
	Emitter
	Joiner
	Registry

	// Get is a low-level method which is meant to only be used from a Consumer.
	Get(lower int64) Event

	// Get is a low-level method which is meant to only be used from a Consumer.
	Set(lower int64, evt Event)

	// metrics
	GetMetrics() Metrics
}
