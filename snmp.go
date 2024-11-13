package snmp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"
	"github.com/gosnmp/gosnmp"
	"github.com/miekg/dns"
)

type Snmp struct {
	Next plugin.Handler

	client  *gosnmp.GoSNMP
	oid     string
	refresh time.Duration

	leases map[string]net.IP
}

type Lease struct {
	Ip       string `json:"ip""`
	Fqdn     string `json:"fqdn"`
	Hostname string `json:"hostname"`
	Mac      string `json:"mac"`
}

var log = clog.NewWithPlugin("snmp")

func (s *Snmp) Run(ctx context.Context) error {
	if err := s.updateLeases(); err != nil {
		return err
	}

	go func() {
		timer := time.NewTimer(s.refresh)
		defer timer.Stop()
		for {
			timer.Reset(s.refresh)
			select {
			case <-ctx.Done():
				log.Debugf("Breaking out of Snmp update loop for: %v", ctx.Err())
				return
			case <-timer.C:
				if err := s.updateLeases(); err != nil && ctx.Err() == nil {
					log.Warningf("Failed to update leases: %v", err)
				}
			}
		}
	}()

	return nil
}

// ServeDNS implements the plugin.Handler interface. This method gets called when example is used
// in a Server.
func (s *Snmp) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	qname := state.Name()

	log.Debugf("Received request for %s", qname)

	ip, match := s.leases[qname]
	if !match {
		log.Debugf("lease for '%s' does not exist\n", qname)
		// Call next plugin (if any).
		return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
	}

	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(60)}
	rr.A = ip

	answers := make([]dns.RR, 1)
	answers[0] = rr

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	w.WriteMsg(m)
	log.Debugf("responded to request %s with ip %s", qname, ip)
	return dns.RcodeSuccess, nil
}

func (s *Snmp) updateLeases() error {
	barr, err := snmpGet(s.client, s.oid)
	if err != nil {
		return err
	}
	parts := bytes.Split(barr, []byte("\n"))

	clearLeases(s.leases)

	l := Lease{}
	for _, b := range parts {
		err := json.Unmarshal(b, &l)
		if err != nil {
			log.Warningf("unmarshal error %s", err)
			continue
		}

		log.Debugf("update %s\t:\t%s", l.Fqdn+".", l.Ip)

		ip := net.ParseIP(l.Ip)

		s.leases[l.Fqdn+"."] = ip
	}

	log.Debugf("updated %d leases", len(s.leases))

	return nil
}

func clearLeases(leases map[string]net.IP) {
	for k := range leases {
		delete(leases, k)
	}
}

func snmpGet(client *gosnmp.GoSNMP, oid string) ([]byte, error) {
	packet, err := client.Get([]string{oid})

	if err != nil {
		return nil, err
	}
	if len(packet.Variables) != 1 {
		return nil, fmt.Errorf("%d results instead of 1", len(packet.Variables))
	}
	if packet.Variables[0].Type != gosnmp.OctetString {
		return nil, fmt.Errorf("unknown PDU type")
	}

	data := packet.Variables[0].Value.([]byte)

	return data, nil
}

// Name implements the Handler interface.
func (e *Snmp) Name() string { return "snmp" }
