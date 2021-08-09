package internal

import (
	"github.com/SkYNewZ/streamdeck-yeelight/pkg/sdk"
)

// our global StreamDeck instance
var streamdeck *sdk.StreamDeck

func RealMain() {
	// Start our plugin
	var err error
	if streamdeck, err = sdk.New(); err != nil {
		panic(err)
	}

	// Register our handlers
	streamdeck.Handler(
		DidReceiveSettingsWillAppear,
		WillDisappear,
		Toggle,
		Color,
		Brightness,
		BrightnessAdjust,
	)

	// Serve the plugin
	streamdeck.Start()
}
