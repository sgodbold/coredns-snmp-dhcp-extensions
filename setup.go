package snmp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/gosnmp/gosnmp"
)

type SnmpConfig struct {
	ip       string
	port     uint16
	username string
	password string
	oid      string
	refresh  time.Duration
}

func init() { plugin.Register("snmp", setup) }

func parseConfig(c *caddy.Controller) (*SnmpConfig, error) {
	conf := SnmpConfig{}

	c.Next() // Ignore "snmp" and give us the next token.
	args := c.RemainingArgs()

	if len(args) != 1 {
		return nil, fmt.Errorf("expecting a single arugment of IP:PORT. found %d args", len(args))
	}

	parts := strings.Split(args[0], ":")

	if len(parts) != 2 || len(parts[0]) <= 0 || len(parts[1]) <= 1 {
		return nil, fmt.Errorf("expecting a single arugment of IP:PORT")
	}

	p, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("non numeric port: %s", parts[1])
	}

	conf.ip = parts[0]
	conf.port = uint16(p)
	refresh := time.Duration(15) * time.Second // default update frequency to 15 seconds

	for c.NextBlock() {
		switch c.Val() {
		case "username":
			if c.NextArg() {
				conf.username = c.Val()
			} else {
				return nil, c.ArgErr()
			}
		case "password":
			if c.NextArg() {
				conf.password = c.Val()
			} else {
				return nil, c.ArgErr()
			}
		case "oid":
			if c.NextArg() {
				conf.oid = c.Val()
			} else {
				return nil, c.ArgErr()
			}
		case "refresh":
			if c.NextArg() {
				refreshStr := c.Val()
				_, err := strconv.Atoi(refreshStr)
				if err == nil {
					refreshStr = fmt.Sprintf("%ss", c.Val())
				}
				refresh, err = time.ParseDuration(refreshStr)
				if err != nil {
					return nil, c.Errf("Unable to parse duration: %v", err)
				}
				if refresh <= 0 {
					return nil, fmt.Errorf("refresh interval must be greater than 0: %q", refreshStr)
				}
				conf.refresh = refresh
			} else {
				return nil, c.ArgErr()
			}
		default:
			return nil, fmt.Errorf("unknown property %q", c.Val())
		}
	}

	if conf.oid == "" {
		return nil, errors.New("must configure an oid")
	}

	return &conf, nil
}

func setup(c *caddy.Controller) error {
	snmpConfig, err := parseConfig(c)

	if err != nil {
		return plugin.Error("snmp", err)
	}

	snmpClient := &gosnmp.GoSNMP{
		Target:        snmpConfig.ip,
		Port:          snmpConfig.port,
		Version:       gosnmp.Version3,
		Timeout:       time.Duration(2) * time.Second,
		SecurityModel: gosnmp.UserSecurityModel,
		MsgFlags:      gosnmp.AuthPriv,
		SecurityParameters: &gosnmp.UsmSecurityParameters{
			UserName:                 snmpConfig.username,
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: snmpConfig.password,
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        snmpConfig.password,
		},
	}
	if err := snmpClient.Connect(); err != nil {
		return plugin.Error("snmp", c.Errf("failed to connect snmp client: %v", err))
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := Snmp{
		client:  snmpClient,
		oid:     snmpConfig.oid,
		refresh: snmpConfig.refresh,
		leases:  make(map[string]net.IP),
	}

	if err := s.Run(ctx); err != nil {
		cancel()
		return plugin.Error("snmp", c.Errf("failed to create snmp plugin: %v", err))
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		s.Next = next
		return &s
	})

	c.OnShutdown(func() error {
		cancel()
		snmpClient.Conn.Close()
		return nil
	})

	return nil
}
