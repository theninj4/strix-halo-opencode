package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

var upstreamClient = &http.Client{}

type Proxy struct {
	cfg          *Config
	internalURL  *url.URL
	reverseProxy *httputil.ReverseProxy
	runner       *Runner
}

func NewProxy(cfg *Config, internalPort int, runner *Runner) *Proxy {
	u, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", internalPort))
	return &Proxy{
		cfg:          cfg,
		internalURL:  u,
		reverseProxy: httputil.NewSingleHostReverseProxy(u),
		runner:       runner,
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost && r.URL.Path == "/v1/chat/completions" {
		p.handleChatCompletions(w, r)
		return
	}
	p.reverseProxy.ServeHTTP(w, r)
}

func (p *Proxy) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	modelName, _ := payload["model"].(string)
	preset, ok := p.cfg.Presets[modelName]
	if !ok {
		http.Error(w,
			fmt.Sprintf("unknown model %q; known presets: %s", modelName, strings.Join(p.cfg.PresetNames(), ", ")),
			http.StatusBadRequest,
		)
		return
	}

	injectPreset(payload, preset)

	modified, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, "failed to re-encode request", http.StatusInternalServerError)
		return
	}

	resp, err := p.doUpstream(r, modified)
	if err != nil {
		if restartErr := p.runner.EnsureHealthy(); restartErr != nil {
			http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
			return
		}
		resp, err = p.doUpstream(r, modified)
		if err != nil {
			http.Error(w, fmt.Sprintf("upstream error: %v", err), http.StatusBadGateway)
			return
		}
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			_, _ = w.Write(buf[:n])
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			break
		}
	}
}

func (p *Proxy) doUpstream(r *http.Request, body []byte) (*http.Response, error) {
	target := *p.internalURL
	target.Path = r.URL.Path
	target.RawQuery = r.URL.RawQuery

	req, err := http.NewRequestWithContext(r.Context(), r.Method, target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, v := range r.Header {
		req.Header[k] = v
	}
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	return upstreamClient.Do(req)
}

// iniToAPIKey maps ini setting names to OpenAI/llama-server JSON field names.
var iniToAPIKey = map[string]string{
	"temp":             "temperature",
	"top-p":            "top_p",
	"top-k":            "top_k",
	"min-p":            "min_p",
	"repeat-penalty":   "repeat_penalty",
	"presence-penalty": "presence_penalty",
}

func injectPreset(payload map[string]interface{}, preset map[string]string) {
	for k, v := range preset {
		switch k {
		case "chat-template-kwargs":
			var obj interface{}
			if err := json.Unmarshal([]byte(v), &obj); err == nil {
				payload["chat_template_kwargs"] = obj
			}
		case "reasoning":
			if v == "on" {
				payload["reasoning_effort"] = "high"
			} else {
				payload["reasoning_effort"] = "none"
			}
		default:
			if apiKey, ok := iniToAPIKey[k]; ok {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					payload[apiKey] = f
				} else {
					payload[apiKey] = v
				}
			}
		}
	}
}
