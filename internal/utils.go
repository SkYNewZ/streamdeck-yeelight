package internal

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/SkYNewZ/go-yeelight"
	"github.com/SkYNewZ/streamdeck-yeelight/pkg/sdk"
)

var (
	ErrMissingSettings = errors.New("missing action settings")
	yeelights          = make(map[string]*yeelightAndKeys, 0)
	lock               = &sync.Mutex{}
)

// yeelightAndKeys is used to manage global state
type yeelightAndKeys struct {
	light  yeelight.Yeelight
	keys   []string           // array of context where this light is defined
	cancel context.CancelFunc // stop routine listening to device notification
	on     bool
}

type setting struct {
	Address     string
	Color       string
	Brightness  string
	Delta       string
	Duration    string
	Temperature string
}

func makeYeelight(event *sdk.ReceivedEvent, s *setting) error {
	// if this light already registered on a key
	lock.Lock()
	defer lock.Unlock()

	v, ok := yeelights[s.Address]
	if !ok || v == nil {
		// initialize a connection
		light, err := yeelight.New(s.Address)
		if err != nil {
			return err
		}

		// store it
		ctx, cancel := context.WithCancel(context.Background())
		yeelights[s.Address] = &yeelightAndKeys{
			light:  light,
			keys:   []string{event.Context},
			cancel: cancel,
			on:     false, // shutdown by default
		}

		if power, err := yeelights[s.Address].light.IsPowerOn(); err == nil {
			yeelights[s.Address].on = power
		}

		// listen for notifications on the light
		notificationsCh, err := yeelights[s.Address].light.Listen(ctx)
		if err != nil {
			return err
		}

		// routine to handle remote changed state
		go func(ch <-chan *yeelight.Notification, light *yeelightAndKeys) {
			for notification := range ch {
				if notification.Method != yeelight.Props {
					continue
				}

				// is it a power change ?
				power, ok := notification.Params["power"]
				if !ok {
					continue
				}

				// yes, set the state according to
				switch power {
				case "on":
					light.on = true
				case "off":
					light.on = false
				}

				for _, state := range light.keys {
					streamdeck.SetState(state, uint8(boolToInt(light.on)))
				}
			}
		}(notificationsCh, yeelights[s.Address])
		return nil
	}

	// refresh the state
	streamdeck.SetState(event.Context, uint8(boolToInt(v.on)))

	// if this light contains this current event key context
	for _, c := range v.keys {
		if c == event.Context {
			return nil // light already registered and contain this key, nothing to do
		}
	}

	v.keys = append(v.keys, event.Context) // else, append this key
	return nil
}

// readSettings of given event supported settings
func readSettings(event *sdk.ReceivedEvent) (*setting, error) {
	settings := event.Payload.Settings
	if settings == nil {
		return nil, ErrMissingSettings
	}

	getValue := func(value interface{}) string {
		if v, ok := value.(string); ok {
			return v
		}

		return ""
	}

	return &setting{
		Address:     getValue(settings["address"]),
		Color:       getValue(settings["color"]),
		Brightness:  getValue(settings["brightness"]),
		Delta:       getValue(settings["delta"]),
		Duration:    getValue(settings["duration"]),
		Temperature: getValue(settings["temperature"]),
	}, nil
}

func getYeelight(settings *setting) (yeelight.Yeelight, error) {
	light, found := yeelights[settings.Address]
	if !found || light == nil {
		return nil, fmt.Errorf("cannot find Yeelight for address [%s]", settings.Address)
	}

	return light.light, nil
}

func boolToInt(value bool) (v int) {
	if value {
		v = 1
	}

	return
}
