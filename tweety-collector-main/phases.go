// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"os"
	"os/signal"
	"syscall"

	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
)

// Function prints out start message to user.
func (app *App) startMessage(timeStart string) {
	app.Logger.LogData(com.CLEAN, "")
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
	app.Logger.LogData(com.INFO, "Tweety-Collector initialized.")
	app.Logger.LogData(com.INFO, "Initialization time: %+v", timeStart)
	app.Logger.LogData(com.INFO, "Version: %s", version)
	app.Logger.LogData(com.INFO, "Author: %s", author)
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
}

// Secure clients shutdown method.
func (app *App) shutdownAwait() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(app.CounterClient.DataChannel)
		close(app.DBSaverClient.UserChannel)
		close(app.DBSaverClient.LocationChannel)
		app.WorkersWaitGroup.Wait()
		app.TwitterClient.Client.CloseIdleConnections()
		app.CounterClient.Client.CloseIdleConnections()
		app.DBSaverClient.Client.CloseIdleConnections()
		app.shutdownMessage()
		err := app.Logger.File.Close()
		if err != nil {
			com.TweetyLog(com.ERROR, "Log file can't close. error: %s", err)
		}
		os.Exit(0)
	}()
}

// Function prints out shutdown message to user.
func (app *App) shutdownMessage() {
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
	app.Logger.LogData(com.INFO, "Thank you for using Tweety-Collector application.")
	app.Logger.LogData(com.INFO, "Client will shutdown gracefully in just a few seconds.")
	app.Logger.LogData(com.INFO, "Thanks for all the fish! Hope we see again!")
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
	app.Logger.LogData(com.CLEAN, "%s%s", delimiter, delimiter)
}
