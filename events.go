package dispatcher

func NewEvent(channel string, data []byte) Event {
	return SimpleEvent{[]byte(channel), data}
}

//Events
type SimpleEvent struct {
	channel []byte
	data    []byte
}

func (s SimpleEvent) Data() []byte {
	return s.data
}

func (s SimpleEvent) Channel() []byte {
	return s.channel
}
