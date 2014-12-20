package dispatcher

import (
	"bytes"
	"strings"
)

type EqualChannelPredicate struct {
	channel []byte
}

func (e EqualChannelPredicate) Apply(evt Event) bool {
	return bytes.Equal(e.channel, evt.Channel())
}

type PrefixChannelPredicate struct {
	prefix string
}

func (p PrefixChannelPredicate) Apply(evt Event) bool {
	return strings.HasPrefix(string(evt.Channel()), p.prefix)
}

type Filter func(Event) bool

type FunctionalPredicate struct {
	filter Filter
}

func (f FunctionalPredicate) Apply(e Event) bool {
	return f.filter(e)
}
