// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"log"
	"time"
)

// Main function. Prints start message, initializes
// Tweety-Collector application and runs it.
func main() {
	timeStart := time.Now().Format(time.ANSIC)
	var config Config
	err := configurationLoader(&config)
	if err != nil {
		log.Fatal(err.Error())
	}
	go startMetrics()
	app := appInit(&config, timeStart)
	app.startMessage(timeStart)
	app.start(config.Username)
}
