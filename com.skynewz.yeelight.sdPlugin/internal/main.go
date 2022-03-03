package internal

import sdk "github.com/SkYNewZ/streamdeck-sdk"

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
		WillAppear,
		WillDisappear,
		Toggle,
		Color,
		Brightness,
		BrightnessAdjust,
		Temperature,
		TemperatureAdjust,
	)

	// Serve the plugin
	streamdeck.Start()
}
