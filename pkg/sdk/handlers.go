package sdk

// HandlerFunc receive event from StreamDeck SDK
type HandlerFunc func(event *ReceivedEvent) error

type Handler interface {
	Handle(event *ReceivedEvent) error
}

func (s *StreamDeck) Handler(h ...Handler) {
	for _, handler := range h {
		s.handlers = append(s.handlers, handler.Handle)
	}
}

func (s *StreamDeck) HandlerFunc(h ...HandlerFunc) {
	for _, handler := range h {
		s.handlers = append(s.handlers, handler)
	}
}
