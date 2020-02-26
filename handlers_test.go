// Copyright 2019-2020 go-gtp authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"net"
	"reflect"
	"testing"

	v1 "github.com/wmnsk/go-gtp/v1"
	v2 "github.com/wmnsk/go-gtp/v2"
	"github.com/wmnsk/go-gtp/v2/messages"
)

func Test_parseCSReq(t *testing.T) {
	type args struct {
		c       *v2.Conn
		mmeAddr net.Addr
		csr     *messages.CreateSessionRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *v2.Session
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCSReq(tt.args.c, tt.args.mmeAddr, tt.args.csr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCSReq() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseCSReq() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_spgwc_genCSRes(t *testing.T) {
	type fields struct {
		cConn  *v2.Conn
		uConns []*v1.UPlaneConn
		cfg    *Config
		errCh  chan error
		ipam   ipam
	}
	type args struct {
		c       *v2.Conn
		session *v2.Session
		csr     *messages.CreateSessionRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    messages.Message
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &spgwc{
				cConn:  tt.fields.cConn,
				uConns: tt.fields.uConns,
				cfg:    tt.fields.cfg,
				errCh:  tt.fields.errCh,
				ipam:   tt.fields.ipam,
			}
			got, err := p.genCSRes(tt.args.c, tt.args.session, tt.args.csr)
			if (err != nil) != tt.wantErr {
				t.Errorf("spgwc.genCSRes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("spgwc.genCSRes() = %v, want %v", got, tt.want)
			}
		})
	}
}
