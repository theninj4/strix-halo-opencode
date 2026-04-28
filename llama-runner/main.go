package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	serverBin := flag.String("server", "llama-server", "path to llama-server binary")
	configFile := flag.String("config", "presents.ini", "path to presets ini file")
	flag.Parse()

	cfg, err := ParseConfig(*configFile)
	if err != nil {
		log.Fatalf("parse config: %v", err)
	}

	proxyPort, err := cfg.ProxyPort()
	if err != nil {
		log.Fatalf("invalid port in config: %v", err)
	}
	internalPort := proxyPort + 1

	args := cfg.LlamaServerArgs(internalPort)
	runner, err := StartRunner(*serverBin, args, internalPort)
	if err != nil {
		log.Fatalf("start llama-server: %v", err)
	}
	defer runner.Kill()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		log.Println("shutting down")
		runner.Kill()
		os.Exit(0)
	}()

	proxy := NewProxy(cfg, internalPort)
	addr := fmt.Sprintf("%s:%d", cfg.ProxyHost(), proxyPort)
	log.Printf("proxy listening on %s (presets: %s)", addr, strings.Join(cfg.PresetNames(), ", "))
	if err := http.ListenAndServe(addr, proxy); err != nil {
		log.Fatalf("proxy server: %v", err)
	}
}
