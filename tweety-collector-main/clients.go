// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"net/http"
	"sync"
	"time"

	com "gitlab.com/leapbit-practice/tweety-lib-communication/comms"
)

// Cache struct represents Tweety-Counter client
// internal cache for ids that failed to sent.
type Cache struct {
	CacheLock sync.Mutex
	Ids       map[string]int64
}

// Twitter API client structure.
type HTTPClientTwitter struct {
	Client http.Client
	Bearer string
}

// Tweety-Counter client structure.
type HTTPClientCounter struct {
	Client      http.Client
	Addr        string
	Port        string
	DataChannel chan []string
	Logger      *com.TweetyLogger
	Cache
}

// Tweety-DBSaver client structure.
type HTTPClientDBSaver struct {
	Client          http.Client
	Addr            string
	Port            string
	UserChannel     chan com.ReqUser
	LocationChannel chan UserLocationPair
	Logger          *com.TweetyLogger
}

// Twitter client constructor.
func NewTwitterClient(bearer string) *HTTPClientTwitter {
	twitterClient := &HTTPClientTwitter{
		Client: http.Client{Timeout: time.Duration(40) * time.Second},
		Bearer: bearer,
	}
	return twitterClient
}

// Tweety-Counter client constructor.
func NewCounterClient(addr string, port string, logger *com.TweetyLogger) *HTTPClientCounter {
	counterClient := &HTTPClientCounter{
		Client:      http.Client{Timeout: time.Duration(40) * time.Second},
		Addr:        addr,
		Port:        port,
		DataChannel: make(chan []string, 1000),
		Logger:      logger,
		Cache: Cache{
			Ids: make(map[string]int64),
		},
	}
	return counterClient
}

// Tweety-DBSaver client constructor.
func NewDBSaverClient(addr string, port string, logger *com.TweetyLogger) *HTTPClientDBSaver {
	dbsaverClient := &HTTPClientDBSaver{
		Client:          http.Client{Timeout: time.Duration(40) * time.Second},
		Addr:            addr,
		Port:            port,
		UserChannel:     make(chan com.ReqUser, 1000),
		LocationChannel: make(chan UserLocationPair, 1000),
		Logger:          logger,
	}
	return dbsaverClient
}

// Method clears Tweety-Counter clients cache memory.
func (counter *HTTPClientCounter) clearCache() {
	for {
		time.Sleep(1 * time.Hour)
		counter.CacheLock.Lock()
		counter.Ids = make(map[string]int64)
		counter.CacheLock.Unlock()
		counter.Logger.LogData(com.WARNING, "Tweety-Counter client cache cleared!!!")
	}
}
