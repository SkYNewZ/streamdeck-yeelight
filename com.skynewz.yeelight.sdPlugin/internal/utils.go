package internal

import (
	"context"
	"errors"
	"sync"

	"github.com/SkYNewZ/go-yeelight"
	sdk "github.com/SkYNewZ/streamdeck-sdk"
)

var (
	ErrMissingSettings = errors.New("missing action settings")

	yeelights = make(map[string]*yeelightAndKeys, 0)
	lock      = &sync.Mutex{}
)

// yeelightAndKeys is used to manage global state.
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

func makeYeelight(event *sdk.ReceivedEvent, settings *setting) (yeelight.Yeelight, error) {
	// if this light already registered on a key
	lock.Lock()
	defer lock.Unlock()

	lightAndKeys, ok := yeelights[settings.Address]
	if !ok || lightAndKeys == nil {
		// initialize a connection
		light, err := yeelight.New(settings.Address)
		if err != nil {
			return nil, err
		}

		// store it
		ctx, cancel := context.WithCancel(context.Background())
		yeelights[settings.Address] = &yeelightAndKeys{
			light:  light,
			keys:   []string{event.Context},
			cancel: cancel,
			on:     false, // shutdown by default
		}

		if power, err := yeelights[settings.Address].light.IsPowerOn(); err == nil {
			yeelights[settings.Address].on = power
		}

		// listen for notifications on the light
		notificationsCh, err := yeelights[settings.Address].light.Listen(ctx)
		if err != nil {
			return nil, err
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
		}(notificationsCh, yeelights[settings.Address])
		return light, nil
	}

	// refresh the state
	streamdeck.SetState(event.Context, uint8(boolToInt(lightAndKeys.on)))

	// if this light contains this current event key context
	for _, c := range lightAndKeys.keys {
		if c == event.Context {
			return lightAndKeys.light, nil // light already registered and contain this key, nothing to do
		}
	}

	lightAndKeys.keys = append(lightAndKeys.keys, event.Context) // else, append this key
	return lightAndKeys.light, nil
}

// readSettings of given event supported settings.
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

func getYeelight(event *sdk.ReceivedEvent, settings *setting) (yeelight.Yeelight, error) {
	light, found := yeelights[settings.Address]
	if !found || light == nil {
		// like not registered on memory, make it
		return makeYeelight(event, settings)
	}

	return light.light, nil
}

func boolToInt(value bool) (v int) {
	if value {
		v = 1
	}

	return
}
