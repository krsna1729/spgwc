// Copyright 2019-2020 go-gtp authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"context"
	"log"
	"net"

	"github.com/pkg/errors"

	v1 "github.com/wmnsk/go-gtp/v1"
	v2 "github.com/wmnsk/go-gtp/v2"
	"github.com/wmnsk/go-gtp/v2/messages"

	goipam "github.com/metal-pod/go-ipam"
)

type ipam struct {
	*goipam.Ipamer
}

type spgwc struct {
	cConn  *v2.Conn
	uConns map[string]*v1.UPlaneConn
	cfg    *Config
	errCh  chan error
	ipam
}

func newSPGWC(cfg *Config) (*spgwc, error) {
	p := &spgwc{
		uConns: make(map[string]*v1.UPlaneConn),
		errCh:  make(chan error, 1),
		cfg:    cfg,
	}

	p.ipam.Ipamer = goipam.New()
	_, err := p.ipam.NewPrefix(cfg.UESubnet)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *spgwc) run(ctx context.Context) error {
	for _, u := range p.cfg.UPFs {
		p.initUPlane(u.S1UAddr)
	}

	cAddr, err := net.ResolveUDPAddr("udp", p.cfg.S11Addr)
	if err != nil {
		return err
	}
	p.cConn = v2.NewConn(cAddr, v2.IFTypeS11S4SGWGTPC, 0)
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

func (p *spgwc) getSubscriberIP(subnet string) (string, error) {
	ip, err := p.AcquireIP(subnet)
	if err != nil {
		return "", err
	}
	return ip.IP.String(), nil
}

func (p *spgwc) initUPlane(s1uAddr string) error {
	// TODO: Connect to dataplane
	s1u, err := net.ResolveUDPAddr("udp", s1uAddr)
	if err != nil {
		return err
	}
	p.uConns[s1uAddr] = v1.NewUPlaneConn(s1u)
	return nil
}

func (p *spgwc) setupUPlane(peerIP, msIP net.IP, otei, itei uint32) error {
	log.Println(peerIP, msIP, otei, itei)
	return nil
}

func (p *spgwc) selectUPlane() string {
	for k, _ := range p.uConns {
		return k
	}
	return ""
}
