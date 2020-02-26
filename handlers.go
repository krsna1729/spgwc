// Copyright 2019-2020 go-gtp authors. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be
// found in the LICENSE file.

package main

import (
	"errors"
	"log"
	"net"
	"strings"

	v2 "github.com/wmnsk/go-gtp/v2"
	"github.com/wmnsk/go-gtp/v2/ies"
	"github.com/wmnsk/go-gtp/v2/messages"
)

func parseCSReq(c *v2.Conn, mmeAddr net.Addr, csr *messages.CreateSessionRequest) (*v2.Session, error) {
	requiredTypesIEs := []struct {
		ieType uint8
		ie     *ies.IE
	}{
		{ies.IMSI, csr.IMSI},
		{ies.MSISDN, csr.MSISDN},
		{ies.MobileEquipmentIdentity, csr.MEI},
		{ies.APNRestriction, csr.APN},
		{ies.ServingNetwork, csr.ServingNetwork},
		{ies.RATType, csr.RATType},
		{ies.FullyQualifiedTEID, csr.SenderFTEIDC},
		{ies.BearerContext, csr.BearerContextsToBeCreated},
	}

	var requiredIEs []*ies.IE
	for _, req := range requiredTypesIEs {
		if req.ie == nil {
			return nil, &v2.RequiredIEMissingError{Type: req.ieType}
		}
		requiredIEs = append(requiredIEs, req.ie)
	}

	session, err := c.ParseCreateSession(mmeAddr, requiredIEs...)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (p *spgwc) genCSRes(c *v2.Conn, session *v2.Session, csr *messages.CreateSessionRequest) (messages.Message, error) {
	var err error
	bearer := session.GetDefaultBearer()

	// Allocate UE IP address
	bearer.SubscriberIP, err = p.getSubscriberIP(p.cfg.UESubnet)
	if err != nil {
		log.Println(err)
		return nil, &v2.RequiredParameterMissingError{}
	}

	// TODO: Add session to userplane

	// TODO: Should we return the same S5/S8 IP or replace with our IPs
	var v4, v6 string
	if csr.PGWS5S8FTEIDC.HasIPv4() {
		v4 = csr.PGWS5S8FTEIDC.MustIPAddress()
	}
	if csr.PGWS5S8FTEIDC.HasIPv6() {
		v6 = csr.PGWS5S8FTEIDC.MustIPAddress()
	}
	if v4 == "" && v6 == "" {
		return nil, &v2.RequiredIEMissingError{Type: ies.FullyQualifiedTEID}
	}

	// Allocate FTEIDs
	cIP := strings.Split(p.cfg.S11Addr, ":")[0]
	uIP := strings.Split(p.cfg.UPFs[0].S1UAddr, ":")[0]

	s11SGWFTEID := c.NewSenderFTEID(cIP, "")
	if s11SGWFTEID == nil {
		return nil, errors.New("Unable to allocate v2.IFTypeS11S4SGWGTPC FTEID")
	}
	session.AddTEID(v2.IFTypeS11S4SGWGTPC, s11SGWFTEID.MustTEID())

	uplane := p.selectUPlane()
	if uplane == "" {
		return nil, errors.New("No User planes")
	}

	s1uSGWFTEID := p.uConns[uplane].NewFTEID(v2.IFTypeS1USGWGTPU, uIP, "")
	if s1uSGWFTEID == nil {
		return nil, errors.New("Unable to allocate v2.IFTypeS1USGWGTPU FTEID")
	}
	session.AddTEID(v2.IFTypeS1USGWGTPU, s1uSGWFTEID.MustTEID())

	s11MMETEID, err := session.GetTEID(v2.IFTypeS11MMEGTPC)
	s5s8cPGWFTEID := ies.NewFullyQualifiedTEID(v2.IFTypeS5S8PGWGTPC, s11SGWFTEID.MustTEID(), v4, v6)
	s5s8uPGWFTEID := ies.NewFullyQualifiedTEID(v2.IFTypeS5S8PGWGTPU, s1uSGWFTEID.MustTEID(), v4, v6)

	// Generate Create Session Response message
	csres := messages.NewCreateSessionResponse(
		s11MMETEID, 0,
		ies.NewCause(v2.CauseRequestAccepted, 0, 0, 0, nil),
		s11SGWFTEID,
		s5s8cPGWFTEID.WithInstance(1),
		ies.NewPDNAddressAllocation(bearer.SubscriberIP),
		ies.NewAPNRestriction(v2.APNRestrictionNoExistingContextsorRestriction),
		ies.NewBearerContext(
			ies.NewCause(v2.CauseRequestAccepted, 0, 0, 0, nil),
			ies.NewEPSBearerID(bearer.EBI),
			s1uSGWFTEID,
			s5s8uPGWFTEID.WithInstance(2),
		),
	)
	return csres, nil
}

func (p *spgwc) handleCreateSessionRequest(c *v2.Conn, mmeAddr net.Addr, msg messages.Message) error {
	log.Printf("Received %s from %s", msg.MessageTypeName(), mmeAddr)

	csreq := msg.(*messages.CreateSessionRequest)
	session, err := parseCSReq(c, mmeAddr, csreq)
	if err != nil {
		// TODO: Respond with error
		return err
	}

	csres, err := p.genCSRes(c, session, csreq)
	if err != nil {
		// TODO: Respond with error
		c.RemoveSession(session)
		return err
	}

	if err := c.RespondTo(mmeAddr, csreq, csres); err != nil {
		c.RemoveSession(session)
		return err
	}

	// Activate and add session
	if err := session.Activate(); err != nil {
		return err
	}
	return nil
}

func (p *spgwc) handleModifyBearerRequest(c *v2.Conn, mmeAddr net.Addr, msg messages.Message) error {
	return nil
}

func (p *spgwc) handleDeleteSessionRequest(c *v2.Conn, mmeAddr net.Addr, msg messages.Message) error {
	log.Printf("Received %s from %s", msg.MessageTypeName(), mmeAddr)

	session, err := c.GetSessionByTEID(msg.TEID(), mmeAddr)
	if err != nil {
		dsr := messages.NewDeleteSessionResponse(
			0, 0,
			ies.NewCause(v2.CauseIMSIIMEINotKnown, 0, 0, 0, nil),
		)
		if err := c.RespondTo(mmeAddr, msg, dsr); err != nil {
			return err
		}

		return err
	}

	teid, err := session.GetTEID(v2.IFTypeS5S8SGWGTPC)
	if err != nil {
		log.Println(err)
		return nil
	}
	dsr := messages.NewDeleteSessionResponse(
		teid, 0,
		ies.NewCause(v2.CauseRequestAccepted, 0, 0, 0, nil),
	)
	if err := c.RespondTo(mmeAddr, msg, dsr); err != nil {
		return err
	}

	log.Printf("Session deleted for Subscriber: %s", session.IMSI)
	c.RemoveSession(session)
	return nil
}
