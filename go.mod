module github.com/SkYNewZ/streamdeck-yeelight

go 1.17

require (
	github.com/SkYNewZ/go-yeelight v0.0.0-20220302145130-8201450feef7
	github.com/SkYNewZ/streamdeck-sdk v0.0.0-20220201151608-bc334ba1c199
	github.com/thoas/go-funk v0.9.1
	gopkg.in/go-playground/colors.v1 v1.2.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
)

// Fix https://github.com/advisories/GHSA-wxc4-f4m6-wwqv
require gopkg.in/yaml.v2 v2.2.8 // indirect
