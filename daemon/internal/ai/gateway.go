package ai

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Gateway exposes Ghost's configured agent-tier model as a localhost
// OpenAI-compatible endpoint (ADR 0003). Point `pi`, Herdr, or any
// OpenAI-client tool at http://127.0.0.1:7700/v1 with the GhOSt session token
// as the API key, and it inherits the user's model + key + routing — one place
// to hold credentials, one place to audit AI traffic.
//
// v1 proxies to openai-compatible providers (Ollama / vLLM / llama.cpp — the
// local-first case). Anthropic providers return 501 (use an OpenAI-compatible
// endpoint, or call Ghost directly).
type Gateway struct{}

func NewGateway() *Gateway { return &Gateway{} }

func (gw *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, p, ok := LoadConfig().AgentProvider()
	if !ok {
		http.Error(w, `{"error":"no model configured — set one in Settings → Ghost AI"}`, http.StatusServiceUnavailable)
		return
	}

	switch p.Type {
	case "openai-compatible":
		base, err := url.Parse(strings.TrimRight(p.URL, "/")) // e.g. http://host:8000/v1
		if err != nil || base.Host == "" {
			http.Error(w, `{"error":"provider URL is invalid"}`, http.StatusBadGateway)
			return
		}
		key := p.Key()
		proxy := &httputil.ReverseProxy{
			// Stream responses immediately (SSE chat completions).
			FlushInterval: -1,
			Director: func(req *http.Request) {
				req.URL.Scheme = base.Scheme
				req.URL.Host = base.Host
				// Incoming path is /v1/<rest>; the provider URL already ends in
				// /v1, so map /v1/<rest> → <base>/<rest>.
				req.URL.Path = base.Path + strings.TrimPrefix(req.URL.Path, "/v1")
				req.Host = base.Host
				// Swap our session token for the provider's key (or none).
				if key != "" {
					req.Header.Set("Authorization", "Bearer "+key)
				} else {
					req.Header.Del("Authorization")
				}
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				log.Printf("ai/gateway: upstream error: %v", err)
				http.Error(w, `{"error":"upstream model unreachable"}`, http.StatusBadGateway)
			},
		}
		log.Printf("ai/gateway: %s %s -> %s", r.Method, r.URL.Path, base.Host)
		proxy.ServeHTTP(w, r)

	case "anthropic":
		http.Error(w, `{"error":"the gateway proxies openai-compatible providers; the Anthropic provider isn't proxied yet — point at an openai-compatible endpoint, or use Ghost directly"}`, http.StatusNotImplemented)

	default:
		http.Error(w, `{"error":"no model configured"}`, http.StatusServiceUnavailable)
	}
}
