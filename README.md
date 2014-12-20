dispatcher
==========

Dispatcher is an event processor with an LMAX Disruptor at its core. Using batch writes, rates greater than **1 Million events per second** are easy to obtain with the upper bound being just shy of **20 Million events per second** (~50ns per event) on a MacBook Pro. Emitting single events is still performant at **400k+ events per second** on average and reaching **800k+ events per second** under heavy load (512 workers emitting 10k events each).

Dispatcher allows you to register many producers as well as add consumers to named event channels. Go's interfaces are used heavily to allow the most flexibility with extensions. Each event must implement `Channel() []byte` and `Data []byte`. Using byte arrays for both of these allow for the most flexibility.

Example
-------

```go
package main

import (
  "fmt"
  "runtime"
  "github.com/eliquious/dispatcher"
)

func main() {
	runtime.GOMAXPROCS(8)

  // size of LMAX Disruptor ring
	var BufferSize int64 = 256 * 1024
	fmt.Println("BufferSize: ", BufferSize)

  // Create dispatcher
	router := dispatcher.NewDispatcher(dispatcher.Config{BufferSize: BufferSize})

  // Number of producers
  var WORKERS = 256
	fmt.Println("Workers: ", WORKERS)

  // Register event handler on channel called 'emit'
	var count int
	router.RouteFunc("emit", func(evt dispatcher.Event) {
	  
	  // print the count every 1000 events
		if count%1000 == 0 {
			fmt.Println(count)
		}
		count++
	})
	
	// Register event handler
	router.RouteFunc("completed", func(evt dispatcher.Event) {
		fmt.Println(string(evt.Data()))
	})

  // start the dispatcher
	router.Start()

  // register and start all the workers
	for w := 0; w < WORKERS; w++ {

    // Batch writes
		// router.Register(BatchWork{w, router, false})
		
		// single writes
		router.Register(SimpleWork{w, router, false})
	}

  // wait until all the producers have finished and all the events have been processed
	router.Join()

  // get dispatcher metrics
	metrics := router.GetMetrics()
	fmt.Println("\nStarted: ", metrics.Started)
	fmt.Println("Total Events: ", metrics.TotalEvents())
	fmt.Println("Processed: ", metrics.Processed())
	fmt.Println("Elapsed: ", metrics.Elapsed())
	fmt.Println("Events Per Second: ", metrics.EventsPerSecond())
	fmt.Println("Nanos Per Event: ", metrics.NanosecondsPerEvent())
	fmt.Println("Stopped: ", metrics.Stopped)
}

```

Producer
--------

```go
// Create byte array for event data
// Tip: Reusing a byte array helps to minimize allocations inside the producers
// This is not always possible but when it is, performance can increase dramatically.
var DATA = make([]byte, 1024)

// SimpleWork type simply produces 10k events on the 'emit' channel
type SimpleWork struct {
	Id        int                       // Used as a worker ID
	Router    dispatcher.Dispatcher     // Dispatcher interface
	isRunning bool                      // boolean used to verify worker is still running
}

// Start is called by the Dispatcher upon registration
func (s SimpleWork) Start() {

  // I'm now running
	s.isRunning = true
	
	// Emit events
	for i := 0; i < 1000; i++ {
	  if !s.isRunning {
	    break
	  }
	
	  // Emit single event to channel 'emit'
	  // The NewEvent method is simply a helper method for creating events.
		s.Router.Emit(dispatcher.NewEvent("emit", DATA))
	}
	
	// Emit an event on the "completed" channel
	s.Router.Emit(dispatcher.NewEvent("completed", []byte(fmt.Sprint("Completed Worker ", s.Id))))
}

// Called when the dispatcher needs to exit
func (s SimpleWork) Stop() {
	s.isRunning = false
}

// Verifies worker is still running
func (s SimpleWork) IsRunning() bool {
	return s.isRunning
}

```

Batch Producer
--------------

```go

// Number of events to emit at one time
var BATCHSIZE = 16

// BatchWork is an event producer which emits events in batches.
// Due to the Disruptor semantics, batch writes are highly performant and can yield much higher operations per second.
type BatchWork struct {
	Id        int                     // worker id
	Router    dispatcher.Dispatcher   // Dispatcher interface
	isRunning bool                    // verifies worker is running
}

// Called when the worker is registered
func (s BatchWork) Start() {

  // Start running
	s.isRunning = true
	
	// Create message buffer
	messages := make([]dispatcher.Event, BATCHSIZE)
	
	// Emit 100k events
	for i := 0; i < 100000; i += BATCHSIZE {
    if !s.isRunning {
      break
    }
    // populate batch event buffer
		for idx := 0; idx < BATCHSIZE; idx++ {
			messages[idx] = dispatcher.NewEvent("emit", DATA)
		}

    // perform batch write
		s.Router.Batch(messages)
	}
	
	// emit event on "completed" channel
	s.Router.Emit(dispatcher.NewEvent("completed", []byte(fmt.Sprint("Completed Worker ", s.Id))))
}

// Called by dispatcher if needs to stop
func (s BatchWork) Stop() {
	s.isRunning = false
}

// used to verify worker is still executing
func (s BatchWork) IsRunning() bool {
	return s.isRunning
}

```
