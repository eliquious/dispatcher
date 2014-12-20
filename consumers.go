package dispatcher

type PredicatedConsumer struct {
	disp    Dispatcher
	pred    Predicate
	handler Handler
}

func (c PredicatedConsumer) Consume(lower, upper int64) {
	for lower <= upper {
		event := c.disp.Get(lower)

		if c.pred.Apply(event) {
			c.handler.Handle(event)
		}
		lower++
	}
}

type PredicatedBatchConsumer struct {
	disp    Dispatcher
	pred    Predicate
	handler BatchHandler
}

func (c PredicatedBatchConsumer) Consume(lower, upper int64) {
	c.handler.StartBatch()
	for lower <= upper {
		event := c.disp.Get(lower)

		if c.pred.Apply(event) {
			c.handler.Handle(event)
		}
		lower++
	}
	c.handler.EndBatch()
}

type PredicatedReplicator struct {
	disp Dispatcher
	pred Predicate
	repl Dispatcher
}

func (p PredicatedReplicator) Consume(lower, upper int64) {
	for lower <= upper {
		event := p.disp.Get(lower)

		if p.pred.Apply(event) {
			p.repl.Emit(event)
		}
		lower++
	}
}
