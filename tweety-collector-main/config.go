// Package main initializes and run Tweety-Collector
// application and its methods.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	etcdEndpoint = "tweety-database-tck-test.demobet.lan:2379"
)

// Structure represents configuration data which is
// stored in config.json file
type Config struct {
	Username    string `json:"Username"`
	Treshold    int    `json:"Treshold"`
	Friends     int    `json:"Friends"`
	Workers     int    `json:"Workers"`
	Bearer      string `json:"Bearer"`
	DBSaverAddr string `json:"DBSaverAddr"`
	DBSaverPort string `json:"DBSaverPort"`
	CounterAddr string `json:"CounterAddr"`
	CounterPort string `json:"CounterPort"`
	LogDir      string `json:"LogDir"`
	LogLevel    int64  `json:"LogLevel"`
}

// Function loads configuration data into variable
// of type Config from config.json using etcd.
func configurationLoader(config *Config) error {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdEndpoint},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("%scannot create etcd client: %v", space, err)
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := cli.Get(ctx, "CollectorConfig")
	cancel()
	if err != nil {
		return fmt.Errorf("%scannot create key-value getter: %v", space, err)
	}
	err = json.Unmarshal(resp.Kvs[0].Value, config)
	if err != nil {
		return fmt.Errorf("%sconfiguration loading error: %v", space, err)
	}
	return nil
}

// // Function loads configuration data into variable
// // of type Config from config.json file.
// func configurationLoader(config *Config) error {
// 	configPath := "/tmp/config.json"
// 	if isWindowsOs() {
// 		configPath = "config.json"
// 	}
// 	f, err := os.Open(configPath)
// 	if err != nil {
// 		return fmt.Errorf("%sconfiguration loading error: %v", space, err)
// 	}
// 	err = json.NewDecoder(f).Decode(config)
// 	if err != nil {
// 		return fmt.Errorf("%sconfiguration loading error: %v", space, err)
// 	}
// 	f.Close()
// 	return nil
// }

// // Method checks if OS type is windows.
// // OS type is set through command line.
// // Default OS is Linux.
// func isWindowsOs() bool {
// 	boolPtr := flag.Bool("win", false, "os type")
// 	flag.Parse()
// 	return *boolPtr
// }
