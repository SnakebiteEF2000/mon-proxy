package proxy

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SnakebiteEF2000/mon-proxy/internal/logger"
	"github.com/docker/docker/client"
)

var log = logger.Log

const (
	serverReadTimeout            = 10 * time.Second
	serverWriteTimeout           = 10 * time.Second
	serverIdleTimeout            = 120 * time.Second
	defaultServerShutdownTimeout = 60 * time.Second
)

type ProxyConfig struct {
	DestinationSocket string
	RequiredLabel     string
	Handler           ProxyHandlerConfig
	Forwarder         Forwarder
}

type ProxyHandlerConfig struct {
	DockerClient *client.Client
	SourceSocket string
}

type Forwarder struct {
	Client        *http.Client
	ClientTimeout time.Duration
}

func (p *ProxyConfig) init() error {
	var err error
	p.Handler.DockerClient, err = client.NewClientWithOpts(client.WithHost("unix://" + p.Handler.SourceSocket)) // automatic api negotiation might get enabled here late
	if err != nil {
		return err
	}
	return nil
}

func (p *ProxyConfig) RunProxy(ctx context.Context) {
	serverctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log.Infof("proxy started on %s filtering for label %s", p.DestinationSocket, p.RequiredLabel)
	err := p.init()
	if err != nil {
		log.Criticalf("failed to create docker client for %s: %v", p.Handler.SourceSocket, err)
		os.Exit(3)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /", p.makeHandler(ctx))

	p.preRunCleanup()

	listener, err := net.Listen("unix", p.DestinationSocket)
	if err != nil {
		log.Errorf("failed to listen on socket: %s: %v", p.DestinationSocket, err)
		return
	}

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
		IdleTimeout:  serverIdleTimeout,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	serverErr := new(atomic.Value)
	go func() {
		defer wg.Done()
		defer cancel()

		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr.Store(err)
		}
	}()

	<-serverctx.Done()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), defaultServerShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Errorf("server shutdown error: %W", err)
		return
	}

	wg.Wait()
}

func (p *ProxyConfig) preRunCleanup() error {
	file, err := os.Stat(p.DestinationSocket)
	if !errors.Is(err, fs.ErrNotExist) {
		log.Debugf("file found %s at path %s", file.Name(), p.DestinationSocket)
		err := os.Remove(p.DestinationSocket)
		if err != nil {
			log.Criticalf("failed removing old socket: %s, %W", p.DestinationSocket, err)
			return err
		}
		log.Warningf("removed socket while pre run cleanup: %s", p.DestinationSocket)
	}
	return nil
}
