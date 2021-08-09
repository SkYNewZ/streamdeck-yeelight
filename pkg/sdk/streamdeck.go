package sdk

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/gorilla/websocket"
)

// StreamDeck will handle events to send/received to/from the StreamDeck application
type StreamDeck struct {
	// UUID is a unique identifier string that should be used to register the plugin once the WebSocket is opened
	UUID string

	// Info containing the Stream Deck application information and devices information.
	Info *Info

	conn    *websocket.Conn
	readCh  chan *ReceivedEvent
	writeCh chan *SendEvent

	// handlers will process incoming events
	handlers []HandlerFunc
}

var (
	port          = flag.Int("port", 0, "The port that should be used to create the WebSocket")
	pluginUUID    = flag.String("pluginUUID", "", "A unique identifier string that should be used to register the plugin once the WebSocket is opened")
	registerEvent = flag.String("registerEvent", "", "The event type that should be used to register the plugin once the WebSocket is opened")
	info          = flag.String("info", "", "A stringified json containing the Stream Deck application information and devices information.")

	ErrMissingPort          = errors.New("missing -port")
	ErrMissingUUID          = errors.New("missing -pluginUUID")
	ErrMissingRegisterEvent = errors.New("missing -registerEvent")
	ErrMissingOrInvalidInfo = errors.New("missing or invalid -info")
)

// New create our plugin, listen to websocket events and register handlers
func New() (*StreamDeck, error) {
	flag.Parse()

	// port
	if v := *port; v == 0 {
		return nil, ErrMissingPort
	}

	// string values
	if *pluginUUID == "" {
		return nil, ErrMissingUUID
	}

	if *registerEvent == "" {
		return nil, ErrMissingRegisterEvent
	}

	if *info == "" {
		return nil, ErrMissingOrInvalidInfo
	}

	// info json object
	var r Info
	if err := json.Unmarshal([]byte(*info), &r); err != nil {
		return nil, ErrMissingOrInvalidInfo
	}

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://localhost:%d", *port), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot init websocket connection: %w", err)
	}

	streamdeck := &StreamDeck{
		UUID:     *pluginUUID,
		Info:     &r,
		conn:     conn,
		readCh:   make(chan *ReceivedEvent),
		writeCh:  make(chan *SendEvent),
		handlers: make([]HandlerFunc, 0),
	}

	if err := streamdeck.register(*registerEvent); err != nil {
		return nil, fmt.Errorf("cannot register plugin: %w", err)
	}

	return streamdeck, nil
}

// Start to serve the plugin
// Ensure you call Handler before
func (s *StreamDeck) Start() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	go s.reader(ctx) // read incoming events
	go s.writer(ctx) // send events
	s.process(ctx)   // will block until ctx is closed
}

// reader listen on incoming messages and send them to dedicated channel
func (s *StreamDeck) reader(ctx context.Context) {
	defer func() {
		close(s.readCh)
		_ = s.conn.Close()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			//_ = s.conn.SetReadDeadline(time.Now().Add(readWait))

			var event ReceivedEvent
			if err := s.conn.ReadJSON(&event); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					s.Log("[ERROR] unexpected close connection: %v", err)
					return
				}

				s.Log("[ERROR] read message: %v", err)
				return
			}

			s.readCh <- &event
		}
	}
}

// writer listen on write channel and send messages
func (s *StreamDeck) writer(ctx context.Context) {
	defer close(s.writeCh)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-s.writeCh:
			//_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))

			if err := s.conn.WriteJSON(event); err != nil {
				s.Log("[ERROR] write event [%s] for action [%s]: %v", event.Event, event.Action, err)
				return
			}
		}
	}
}

// process will listen to incoming events and process them
func (s *StreamDeck) process(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-s.readCh:
			if !ok {
				// The channel is closed
				return
			}

			// Send event to all to registered handlers
			go func(event *ReceivedEvent) {
				for _, h := range s.handlers {
					if err := h(event); err != nil {
						s.Log("[ERROR] event [%s] action [%s]: %v", event.Event, event.Action, err)
						s.Alert(event.Context)
					}
				}
			}(e)
		}
	}
}
