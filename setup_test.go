package snmp

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestSetupRoute53(t *testing.T) {
	tests := []struct {
		body          string
		expectedError bool
	}{
		{`snmp`, true},
		{`snmp :`, true},
		{`snmp 192.168.1.1:1234`, true},
		{`snmp 192.168.1.1:1234 {
    username USER
}`, true},
		{`snmp 192.168.1.1:1234 {
    username USER
	password PASS
}`, true},
		{`snmp 192.168.1.1:1234 {
    username USER
	password PASS
	refresh 5m
}`, true},
		{`snmp 192.168.1.1:1234 {
    username USER
	password PASS
	refresh -5m
	oid .1.2.3.4
}`, true},
		{`snmp 192.168.1.1:1234 {
    username USER
	password PASS
	refresh 5m
	oid .1.2.3.4
}`, false},
	}

	for _, test := range tests {
		c := caddy.NewTestController("dns", test.body)
		if _, err := parseConfig(c); (err == nil) == test.expectedError {
			t.Errorf("Unexpected errors: %v", err)
		}
	}
}
