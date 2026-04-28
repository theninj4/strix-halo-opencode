# llama-runner

A small Go wrapper around `llama-server` that adds named model presets via an INI config file. It lets us load model weights once and expose several virtual models using different settings. Behind the scenes it starts `llama-server` on an internal port, waits for it to become healthy, then listens on a public port and proxies requests to it — rewriting `POST /v1/chat/completions` bodies to inject preset parameters based on the `model` field.

## How it works

1. On startup, reads a presets INI file (default: `presets.ini`).
2. Spawns `llama-server` on `<public port + 1>` (internal, bound to `127.0.0.1`).
3. Waits up to 2 minutes for `llama-server` to report healthy at `/health`.
4. Starts an HTTP proxy on the public port. All requests are forwarded to `llama-server` except `POST /v1/chat/completions`, where the `model` field is matched against a named preset and the request body is augmented with that preset's parameters.

## Config file format

The INI file is the same one llama-server expects:
https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md#model-presets

## Compiling

This produces an exectuable `llama-runner` for Linux amd64:
```
GOOS=linux GOARCH=amd64 go build -o llama-runner ./...
```

## Usage

```
llama-runner [--server <path>] [--config <path>]

  --server   path to llama-server binary (default: llama-server)
  --config   path to presets INI file    (default: presents.ini)
```
