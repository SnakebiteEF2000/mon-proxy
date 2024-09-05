package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type SocketConfig struct {
	DestinationSocket string
	RequiredLabel     string
}

var (
	logger       *log.Logger
	sourceSocket string
	allSockets   []SocketConfig
	socketMutex  sync.Mutex
)

func init() {
	logger = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	sourceSocket = getEnv("SOURCE_SOCKET", "/var/run/docker.sock")
}

func main() {
	logger.Println("Starting Docker proxy with multiple sockets")

	configs := getSocketConfigs()
	if len(configs) == 0 {
		logger.Fatal("No valid socket configurations found")
	}

	socketMutex.Lock()
	allSockets = configs
	socketMutex.Unlock()

	setupSignalHandler(globalCleanup)

	var wg sync.WaitGroup
	for _, config := range configs {
		wg.Add(1)
		go func(cfg SocketConfig) {
			defer wg.Done()
			runProxy(cfg)
		}(config)
	}

	wg.Wait()
}

func getSocketConfigs() []SocketConfig {
	var configs []SocketConfig
	i := 1
	for {
		destSocket := getEnv(fmt.Sprintf("DESTINATION_SOCKET_%d", i), "")
		label := getEnv(fmt.Sprintf("REQUIRED_LABEL_%d", i), "")

		if destSocket == "" || label == "" {
			break
		}

		configs = append(configs, SocketConfig{
			DestinationSocket: destSocket,
			RequiredLabel:     label,
		})
		i++
	}
	return configs
}

func runProxy(config SocketConfig) {
	logger.Printf("Starting proxy for %s, filtering by %s", config.DestinationSocket, config.RequiredLabel)

	cli, err := client.NewClientWithOpts(client.WithHost("unix://" + sourceSocket))
	if err != nil {
		logger.Fatalf("Failed to create Docker client for %s: %v", config.DestinationSocket, err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", createHandler(cli, config))

	listener, err := net.Listen("unix", config.DestinationSocket)
	if err != nil {
		logger.Fatalf("Failed to listen on socket %s: %v", config.DestinationSocket, err)
	}

	cleanup := func() {
		listener.Close()
		if err := os.Remove(config.DestinationSocket); err != nil {
			logger.Printf("Error removing socket file %s: %v", config.DestinationSocket, err)
		} else {
			logger.Printf("Cleaned up and removed socket file: %s", config.DestinationSocket)
		}
	}
	defer cleanup()

	logger.Printf("Proxy listening on %s", config.DestinationSocket)
	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("HTTP server error on %s: %v", config.DestinationSocket, err)
	}
}

func createHandler(cli *client.Client, config SocketConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("[%s] Received request: %s %s", config.DestinationSocket, r.Method, r.URL.Path)
		path := normalizePath(r.URL.Path)

		if strings.HasPrefix(path, "/containers") {
			filterContainers(cli, w, r, config)
			return
		}
		proxyRequest(w, r)
	}
}

func filterContainers(cli *client.Client, w http.ResponseWriter, r *http.Request, config SocketConfig) {
	path := normalizePath(r.URL.Path)

	switch {
	case path == "/containers/json":
		handleContainerList(cli, w, config.RequiredLabel)
	case strings.HasPrefix(path, "/containers/"):
		handleContainerOperation(cli, w, r, config)
	default:
		logger.Printf("Unhandled request: %s", path)
		proxyRequest(w, r)
	}
}

func handleContainerList(cli *client.Client, w http.ResponseWriter, requiredLabel string) {
	ctx := context.Background()
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", requiredLabel)

	options := container.ListOptions{Filters: filterArgs}
	containers, err := cli.ContainerList(ctx, options)
	if err != nil {
		logger.Printf("Error listing containers: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)
}

func handleContainerOperation(cli *client.Client, w http.ResponseWriter, r *http.Request, config SocketConfig) {
	ctx := context.Background()
	path := normalizePath(r.URL.Path)
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		logger.Printf("Invalid container request: %s", path)
		proxyRequest(w, r)
		return
	}

	containerIDOrName := parts[2]
	// If the path includes more parts (like /containers/{name}/json), join them
	if len(parts) > 3 {
		containerIDOrName = strings.Join(parts[2:len(parts)-1], "/")
	}

	logger.Printf("Checking access for container: %s, method: %s, path: %s", containerIDOrName, r.Method, path)
	container, err := cli.ContainerInspect(ctx, containerIDOrName)
	if err != nil {
		logger.Printf("Error inspecting container %s: %v", containerIDOrName, err)
		proxyRequest(w, r)
		return
	}

	if !hasRequiredLabel(container, config.RequiredLabel) {
		logger.Printf("Access denied for container %s: missing or incorrect required label", containerIDOrName)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Allow all read-only operations
	if isReadOnlyOperation(r.Method, parts) {
		logger.Printf("Access granted for read-only operation on container %s: %s %s", containerIDOrName, r.Method, path)
		proxyRequest(w, r)
		return
	}

	logger.Printf("Access denied for non-read-only operation on container %s: %s %s", containerIDOrName, r.Method, path)
	http.Error(w, "Access denied for non-read-only operation", http.StatusForbidden)
}

func hasRequiredLabel(container types.ContainerJSON, requiredLabel string) bool {
	parts := strings.SplitN(requiredLabel, "=", 2)
	if len(parts) != 2 {
		return false
	}
	value, ok := container.Config.Labels[parts[0]]
	return ok && value == parts[1]
}

func proxyRequest(w http.ResponseWriter, r *http.Request) {
	logger.Printf("Proxying request to: %s %s", r.Method, r.URL.Path)
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", sourceSocket)
			},
		},
	}

	outreq, err := http.NewRequest(r.Method, fmt.Sprintf("http://docker%s", r.URL.RequestURI()), r.Body)
	if err != nil {
		logger.Printf("Error creating proxy request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	copyHeader(outreq.Header, r.Header)

	resp, err := httpClient.Do(outreq)
	if err != nil {
		logger.Printf("Error executing proxy request: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	logger.Printf("Proxy request completed with status: %d", resp.StatusCode)
	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("Error reading response body: %v", err)
		http.Error(w, "Error reading response", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode >= 400 {
		logger.Printf("Error response body: %s", string(body))
	}

	w.Write(body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func setupSignalHandler(cleanup func()) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Printf("Received termination signal: %v", sig)
		cleanup()
		os.Exit(0)
	}()
}

func normalizePath(path string) string {
	// Remove API version prefix if present
	parts := strings.SplitN(path, "/", 3)
	if len(parts) > 2 && strings.HasPrefix(parts[1], "v1.") {
		return "/" + parts[2]
	}
	return path
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func globalCleanup() {
	logger.Println("Performing global cleanup")
	socketMutex.Lock()
	defer socketMutex.Unlock()

	for _, config := range allSockets {
		if err := os.Remove(config.DestinationSocket); err != nil {
			logger.Printf("Error removing socket file %s: %v", config.DestinationSocket, err)
		} else {
			logger.Printf("Cleaned up and removed socket file: %s", config.DestinationSocket)
		}
	}
}

func isReadOnlyOperation(method string, pathParts []string) bool {
	if method != "GET" && method != "HEAD" {
		return false
	}

	// Allow all GET and HEAD requests for container-specific operations
	if len(pathParts) > 3 {
		return true
	}

	// Allow general container inspection
	return true
}
