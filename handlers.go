package dispatcher

// Handler

type DefaultHandler struct {
	handler RouteFunc
}

func (d DefaultHandler) Handle(evt Event) {
	d.handler(evt)
}

type StagedHandler struct {
	proc StagedProcessor
}

func (s StagedHandler) Handle(evt Event) {
	s.proc.PreProcess(evt)
	s.proc.Process(evt)
	s.proc.PostProcess(evt)
}
