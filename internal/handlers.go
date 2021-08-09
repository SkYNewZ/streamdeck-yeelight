package internal

import (
	"fmt"
	"strconv"

	"github.com/SkYNewZ/go-yeelight"
	"github.com/SkYNewZ/streamdeck-yeelight/pkg/sdk"
	"github.com/thoas/go-funk"
	"gopkg.in/go-playground/colors.v1"
)

type Action struct {
	Action string
	Event  []sdk.EventName
	Run    func(event *sdk.ReceivedEvent, light yeelight.Yeelight, s *setting) error
	PreRun func(event *sdk.ReceivedEvent, s *setting) error
}

func (a *Action) Handle(event *sdk.ReceivedEvent) error {
	// unhandled action
	if a.Action != "" && event.Action != a.Action {
		return nil
	}

	// unhandled event
	if validEvent := funk.Contains(a.Event, func(e sdk.EventName) bool {
		return event.Event == e
	}); !validEvent && len(a.Event) > 0 {
		return nil
	}

	// read settings
	settings, err := readSettings(event)
	if err != nil {
		return err
	}

	if a.PreRun != nil {
		if err := a.PreRun(event, settings); err != nil {
			return err
		}
	}

	// init connection with the Yeelight
	light, err := getYeelight(settings)
	if err != nil {
		return err
	}

	if a.Run != nil {
		return a.Run(event, light, settings)
	}

	return nil
}

var Toggle = &Action{
	Action: "com.skynewz.yeelight.toggle",
	Event:  []sdk.EventName{sdk.KeyUp},
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
	Event:  []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, s *setting) error {
		if s.Color == "" {
			return fmt.Errorf("invalid color [%s]", s.Color)
		}

		hex, err := colors.ParseHEX(s.Color)
		if err != nil {
			return fmt.Errorf("cannot parse color [%s]: %w", s.Color, err)
		}

		return light.SetRGB(hex.ToRGB().R, hex.ToRGB().G, hex.ToRGB().B)
	},
}

var Brightness = &Action{
	Action: "com.skynewz.yeelight.brightness",
	Event:  []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, s *setting) error {
		if s.Brightness == "" {
			return fmt.Errorf("invalid brightness [%s]", s.Brightness)
		}

		value, err := strconv.ParseUint(s.Brightness, 10, 8)
		if err != nil {
			return fmt.Errorf("cannot parse brightness [%s]: %w", s.Brightness, err)
		}

		return light.SetBrightness(uint8(value))
	},
}

var BrightnessAdjust = &Action{
	Action: "com.skynewz.yeelight.brightness_adjust",
	Event:  []sdk.EventName{sdk.KeyUp},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, s *setting) error {
		if s.Delta == "" {
			return fmt.Errorf("invalid brightness delta [%s]", s.Delta)
		}

		delta, err := strconv.ParseInt(s.Delta, 10, 8)
		if err != nil {
			return fmt.Errorf("cannot parse brightness delta [%s]: %w", s.Delta, err)
		}

		var duration uint64 = 500 // default duration
		if s.Duration != "" {
			if v, err := strconv.ParseUint(s.Duration, 10, 64); err == nil {
				duration = v
			}
		}

		return light.AdjustBright(int8(delta), duration)
	},
}

var DidReceiveSettingsWillAppear = &Action{
	Action: "",
	Event:  []sdk.EventName{sdk.WillAppear, sdk.DidReceiveSettings},
	PreRun: func(event *sdk.ReceivedEvent, s *setting) error {
		return makeYeelight(event, s)
	},
}

var WillDisappear = &Action{
	Action: "",
	Event:  []sdk.EventName{sdk.WillDisappear},
	Run: func(event *sdk.ReceivedEvent, light yeelight.Yeelight, s *setting) error {
		lock.Lock()
		defer lock.Unlock()
		v, ok := yeelights[s.Address]
		if !ok || v == nil {
			// this light is not stored
			return nil
		}

		// filter keys by removing the current disappearing one
		var keys = make([]string, 0)
		for _, k := range v.keys {
			if k != event.Context {
				keys = append(keys, k)
			}
		}

		// stop notifications routine and reset keys with this light
		defer v.cancel()
		if len(keys) == 0 {
			delete(yeelights, s.Address)
			return nil
		}

		v.keys = keys
		return nil
	},
}
