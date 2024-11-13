package snmp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"
	"github.com/gosnmp/gosnmp"

	"github.com/miekg/dns"
)

// These are integ tests, needs a real snmp server

func TestSnmpDhcp(t *testing.T) {
	snmpClient := &gosnmp.GoSNMP{
		Target:        "192.168.1.1",
		Port:          161,
		Version:       gosnmp.Version3,
		Timeout:       time.Duration(2) * time.Second,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		MaxOids:       1,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 "user",
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: "password",
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        "password",
		},
	}
	if err := snmpClient.Connect(); err != nil {
		t.Error("failed to connect client")
	}

	s := Snmp{
		Next:    test.ErrorHandler(),
		client:  snmpClient,
		oid:     ".1.3.6.1.4.1.8072.1.3.2.3.1.2.5.115.116.101.118.101",
		refresh: time.Duration(15) * time.Second,
		leases:  make(map[string]net.IP),
	}

	ctx := context.TODO()
	r := new(dns.Msg)
	r.SetQuestion("dhcp-test.priv.godbold.cloud", dns.TypeA)

	// Create a new Recorder that captures the result, this isn't actually used in this test
	// as it just serves as something that implements the dns.ResponseWriter interface.
	rec := dnstest.NewRecorder(&test.ResponseWriter{})

	// Call our plugin directly, and check the result.
	s.updateLeases()
	s.ServeDNS(ctx, rec, r)
}

func TestUpdateLeases(t *testing.T) {
	snmpClient := &gosnmp.GoSNMP{
		Target:        "192.168.1.1",
		Port:          161,
		Version:       gosnmp.Version3,
		Timeout:       time.Duration(2) * time.Second,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		MaxOids:       1,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 "user",
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: "password",
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        "password",
		},
	}
	if err := snmpClient.Connect(); err != nil {
		t.Error("failed to connect client")
	}

	s := Snmp{
		Next:    test.ErrorHandler(),
		client:  snmpClient,
		oid:     ".1.3.6.1.4.1.8072.1.3.2.3.1.2.5.115.116.101.118.101",
		refresh: time.Duration(15) * time.Second,
		leases:  make(map[string]net.IP),
	}

	s.updateLeases()

	print(len(s.leases))
}
