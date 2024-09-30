package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/SnakebiteEF2000/mon-proxy/internal/util/normalize"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"golang.org/x/net/publicsuffix"
)

func (p *ProxyConfig) makeHandler(ctx context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := normalize.NormalizePath(r.URL.Path) // can be removed
		log.Debugf("request path: %v", r.URL.Path)
		//log.Debugf("normalized path %s", path)

		if strings.Contains(path, "/containers/json") {
			log.Debugf("filtered containers hit for path: %s", path)
			p.filterContainers(ctx, w, r)
			return
		}
		p.proxyRequest(ctx, w, r)
	})
}

func (p *ProxyConfig) filterContainers(ctx context.Context, w http.ResponseWriter, _ *http.Request) {
	labelFilter := filters.NewArgs()
	labelFilter.Add("label", p.RequiredLabel)

	containers, err := p.Handler.DockerClient.ContainerList(ctx, container.ListOptions{
		Filters: labelFilter,
	})
	if err != nil {
		log.Errorf("failed requesting docker: %W", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(containers)

	// Info only
	for _, ctr := range containers {
		log.Infof("%v filtered by: %s (status: %s)\n", ctr.Names, p.RequiredLabel, ctr.Status) // ctr.Names will show all nmes from the array -> might rever it to this
	}
}

func (p *ProxyConfig) makeClient() {
	if p.Forwarder.Client == nil {
		jar, _ := cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})

		p.Forwarder.Client = &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					return net.Dial("unix", p.Handler.SourceSocket)
				},
			},
			Timeout: 10 * time.Second, // use default timeout p.Forwarder.ClientTimeout
			Jar:     jar,
		}
	}
}

// should be refactored to use the go docker client and no direct requests
func (p *ProxyConfig) proxyRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	p.makeClient()
	log.Debugf("proxying request to: %s %s", r.Method, r.URL.Path)

	req, err := http.NewRequestWithContext(ctx, r.Method, fmt.Sprintf("http://docker%s", r.URL.RequestURI()), r.Body)
	if err != nil {
		log.Errorf("error creating proxy request: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	copyHeader(req.Header, r.Header)

	resp, err := p.Forwarder.Client.Do(req)
	if err != nil {
		log.Error("error executing proxy request: %w", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("error reading response body: %v", err)
		http.Error(w, "error reading response", http.StatusInternalServerError)
		return err
	}

	if resp.StatusCode >= 400 {
		log.Warningf("response body: %s", string(body))
	}

	w.Write(body)
	return nil
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
