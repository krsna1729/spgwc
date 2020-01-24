// Copyright 2019-2020 go-gtp authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"net"

	"github.com/pkg/errors"

	goipam "github.com/metal-pod/go-ipam"
	v2 "github.com/wmnsk/go-gtp/v2"
	"github.com/wmnsk/go-gtp/v2/messages"
)

type ipam struct {
	*goipam.Ipamer
	prefixes []*goipam.Prefix
}

type spgwc struct {
	cConn *v2.Conn
	cfg   *Config
	errCh chan error
	ipam
}

func newSPGWC(cfg *Config) (*spgwc, error) {
	p := &spgwc{
		errCh: make(chan error, 1),
		cfg:   cfg,
	}

	p.ipam.Ipamer = goipam.New()
	prefix, err := p.ipam.NewPrefix(cfg.UESubnet)
	if err != nil {
		return nil, err
	}
	p.prefixes = append(p.prefixes, prefix)

	return p, nil
}

func (p *spgwc) run(ctx context.Context) error {
	cAddr, err := net.ResolveUDPAddr("udp", p.cfg.S11Addr)
	if err != nil {
		return err
	}
	p.cConn = v2.NewConn(cAddr, 0)
	go func() {
		if err := p.cConn.ListenAndServe(ctx); err != nil {
			log.Println(err)
			return
		}
	}()
	log.Printf("Started listening on %s", cAddr)

	// register handlers for ALL the messages you expect remote endpoint to send.
	p.cConn.AddHandlers(map[uint8]v2.HandlerFunc{
		messages.MsgTypeCreateSessionRequest: p.handleCreateSessionRequest,
		messages.MsgTypeModifyBearerRequest:  p.handleModifyBearerRequest,
		messages.MsgTypeDeleteSessionRequest: p.handleDeleteSessionRequest,
	})

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-p.errCh:
			log.Printf("Warning: %s", err)
		}
	}
}

func (p *spgwc) close() error {
	var errs []error

	if err := p.cConn.Close(); err != nil {
		errs = append(errs, err)
	}

	close(p.errCh)

	if len(errs) > 0 {
		return errors.Errorf("errors while closing S-GW: %v", errs)
	}
	return nil
}

func (p *spgwc) setupUPlane(peerIP, msIP net.IP, otei, itei uint32) error {

	return nil
}
