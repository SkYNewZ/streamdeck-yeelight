package sdk

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetLevel(log.TraceLevel)
}

// Log send log message to StreamDeck SDK
func (s *StreamDeck) Log(format string, a ...interface{}) {
	log.Debugf(format, a...)
	s.writeCh <- &SendEvent{
		Event:   LogMessage,
		Payload: &SendEventPayload{Message: fmt.Sprintf(format, a...)},
	}
}
