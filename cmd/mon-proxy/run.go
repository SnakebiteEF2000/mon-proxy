package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"sync"

	"github.com/SnakebiteEF2000/mon-proxy/internal/logger"
	"github.com/SnakebiteEF2000/mon-proxy/internal/util/cfg"
)

type StatusCode = int

var (
	log          = logger.Log
	sourceSocket string
)

const (
	StatusOK StatusCode = iota
	StatusFailedSocketConfig
)

func init() {
	/*debug := flag.Bool("debug", false, "set debug log level")
	flag.Parse()*/

	debugStr := cfg.GetEnv("DEBUG_MODE", "false")
	debug, _ := strconv.ParseBool(debugStr)

	logger.SetupLogger(debug)

	sourceSocket = cfg.GetEnv("SOURCE_SOCKET", "/var/run/docker.sock")
	log.Debug("source socket: ", sourceSocket)
}

func run() int {
	log.Info("Starting...")

	configs := GetSocketConfigsMOCK() // debug only
	//configs := GetSocketConfigs()
	if len(configs) == 0 {
		log.Critical("No valid socket configuration found")
		return StatusFailedSocketConfig
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var wg sync.WaitGroup

	for _, config := range configs {
		config.Handler.SourceSocket = sourceSocket
		wg.Add(1)
		go func() {
			defer wg.Done()
			config.RunProxy(ctx)
		}()
	}

	wg.Wait()

	log.Critical("shuting down...")

	return StatusOK
}
