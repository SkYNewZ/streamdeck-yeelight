package internal

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/SkYNewZ/go-yeelight"
	sdk "github.com/SkYNewZ/streamdeck-sdk"
	"gopkg.in/go-playground/colors.v1"
)

var (
	ErrInvalidColor       = errors.New("invalid color")
	ErrInvalidBrightness  = errors.New("invalid brightness")
	ErrInvalidTemperature = errors.New("invalid temperature")
	ErrInvalidDelta       = errors.New("invalid delta")
)

type Action struct {
	Action string
	Events []sdk.EventName
	Run    func(*sdk.ReceivedEvent, yeelight.Yeelight, *setting) error
}

func (a Action) Logf(event *sdk.ReceivedEvent, format string, args ...interface{}) {
	message := fmt.Sprintf("[DEBUG] received event [%s] for action [%s]: ", event.Event, a.Action)
	message += fmt.Sprintf(format, args...)
	streamdeck.Log(message)
}

func (a *Action) Handle(event *sdk.ReceivedEvent) error {
	// unhandled action
	if a.Action != "" && event.Action != a.Action {
		a.Logf(event, "unhandled action")
		return nil
	}

	// unhandled event
	var found bool
	for _, e := range a.Events {
		if e == event.Event {
			found = true
			break
		}
	}

	if !found {
		a.Logf(event, "unhandled event. supported events: %s", a.Events)
		return nil
	}

	// read settings
	a.Logf(event, "reading settings")
	settings, err := readSettings(event)
	if err != nil {
		return err
	}

	// get light on memory
	a.Logf(event, "getting light on memory with settings [%s]", settings)
	light, err := getYeelight(event, settings)
	if err != nil {
		return err
	}

	a.Logf(event, "running action with settings [%s]", settings)
	if a.Run != nil {
		return a.Run(event, light, settings)
	}

	return nil
}

var Toggle = &Action{
	Action: "com.skynewz.yeelight.toggle",
	Events: []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, _ *setting) error {
		var wantedState bool // off by default
		switch {
		case event.Payload.IsInMultiAction:
			// On multi action, StreamDeck give us the wanted state, it's not a toggle
			wantedState = event.Payload.UserDesiredState == 1
		default:
			wantedState = !(event.Payload.State == 1)
		}

		switch wantedState {
		case true:
			return light.On() // Toggle on the light
		case false:
			return light.Off() // Toggle off the light
		}

		return nil
	},
}

var Color = &Action{
	Action: "com.skynewz.yeelight.color",
	Events: []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, settings *setting) error {
		if settings.Color == "" {
			return fmt.Errorf("%w: %s", ErrInvalidColor, settings.Color)
		}

		hex, err := colors.ParseHEX(settings.Color)
		if err != nil {
			return fmt.Errorf("cannot parse color [%s]: %w", settings.Color, err)
		}

		rgb := hex.ToRGB()
		return light.SetRGB(rgb.R, rgb.G, rgb.B)
	},
}

var Brightness = &Action{
	Action: "com.skynewz.yeelight.brightness",
	Events: []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, settings *setting) error {
		if settings.Brightness == "" {
			return fmt.Errorf("%w: %s", ErrInvalidBrightness, settings.Brightness)
		}

		value, err := strconv.Atoi(settings.Brightness)
		if err != nil {
			return fmt.Errorf("cannot parse brightness [%s]: %w", settings.Brightness, err)
		}

		return light.SetBrightness(value)
	},
}

var Temperature = &Action{
	Action: "com.skynewz.yeelight.temperature",
	Events: []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, settings *setting) error {
		if settings.Temperature == "" {
			return fmt.Errorf("%w: %s", ErrInvalidTemperature, settings.Temperature)
		}

		value, err := strconv.Atoi(settings.Temperature)
		if err != nil {
			return fmt.Errorf("cannot parse temperature [%s]: %w", settings.Temperature, err)
		}

		return light.SetColorTemperature(value)
	},
}

var BrightnessAdjust = &Action{
	Action: "com.skynewz.yeelight.brightness.adjust",
	Events: []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, settings *setting) error {
		if settings.Delta == "" {
			return fmt.Errorf("%w: %s", ErrInvalidDelta, settings.Delta)
		}

		delta, err := strconv.Atoi(settings.Delta)
		if err != nil {
			return fmt.Errorf("cannot parse brightness delta [%s]: %w", settings.Delta, err)
		}

		duration := 500 // default duration
		if settings.Duration != "" {
			if v, err := strconv.Atoi(settings.Duration); err == nil {
				duration = v
			}
		}

		return light.AdjustBrightness(delta, duration)
	},
}

var TemperatureAdjust = &Action{
	Action: "com.skynewz.yeelight.temperature.adjust",
	Events: []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, settings *setting) error {
		if settings.Delta == "" {
			return fmt.Errorf("%w: %s", ErrInvalidDelta, settings.Delta)
		}

		delta, err := strconv.Atoi(settings.Delta)
		if err != nil {
			return fmt.Errorf("cannot parse temperature delta [%s]: %w", settings.Delta, err)
		}

		duration := 500 // default duration
		if settings.Duration != "" {
			if v, err := strconv.Atoi(settings.Duration); err == nil {
				duration = v
			}
		}

		return light.AdjustColorTemperature(delta, duration)
	},
}

var WillAppear = &Action{
	Action: "",
	Events: []sdk.EventName{sdk.WillAppear, sdk.DidReceiveSettings},
	Run: func(event *sdk.ReceivedEvent, _ yeelight.Yeelight, settings *setting) error {
		_, err := makeYeelight(event, settings)
		return err
	},
}

var WillDisappear = &Action{
	Action: "",
	Events: []sdk.EventName{sdk.WillDisappear},
	Run: func(event *sdk.ReceivedEvent, _ yeelight.Yeelight, settings *setting) error {
		lock.Lock()
		defer lock.Unlock()
		light, ok := yeelights[settings.Address]
		if !ok || light == nil {
			// this light is not stored
			return nil
		}

		// filter keys by removing the current disappearing one
		keys := make([]string, 0)
		for _, k := range light.keys {
			if k != event.Context {
				keys = append(keys, k)
			}
		}

		// no keys left associated to this light
		// close the connection
		// stop listening to the light events
		if len(keys) == 0 {
			light.cancel()
			delete(yeelights, settings.Address)
			return nil
		}

		light.keys = keys
		return nil
	},
}
