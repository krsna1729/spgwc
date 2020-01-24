// Copyright 2019-2020 go-gtp authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

// Command spgwc is an implementation of S/PGW-C only with GTP-related features.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var configPath = flag.String("config", "./spgwc.yml", "Path to the configuration file.")
	flag.Parse()
	log.SetPrefix("[S/PGW-C] ")

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Println(err)
		return
	}

	spgwc, err := newSPGWC(cfg)
	if err != nil {
		log.Printf("failed to initialize P-GW: %s", err)
		return
	}
	defer spgwc.close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGHUP)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fatalCh := make(chan error)
	go func() {
		if err := spgwc.run(ctx); err != nil {
			fatalCh <- err
		}
	}()

	for {
		select {
		case sig := <-sigCh:
			// TODO: reload config on receiving SIGHUP
			log.Println(sig)
			return
		case err := <-spgwc.errCh:
			log.Printf("WARN: %s", err)
		case err := <-fatalCh:
			log.Printf("FATAL: %s", err)
			return
		}
	}
}
