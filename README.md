# coredns-snmp-dhcp-extensions
Plugin for CoreDNS to create A records using data from SNMP 

## SNMP Data
Your SNMP server must send leases in a single OID in the following json format, one lease per line.

```
{"ip":"10.0.0.1","fqdn":"host1.example.com","hostname":"host1","mac":"0e:70:e1:98:XX:XX"}
{"ip":"10.0.0.2","fqdn":"host2.example.com","hostname":"host2","mac":"0e:70:e1:98:XX:XX"}
```

Only ip and fqn are used.

If you're running pfsense (or maybe anything with NET-SNMP), see https://github.com/sgodbold/pfsense-snmpd-dhcp-extension for extending your snmp server to provid this data.

## Coredns Configuration
```
. {
  snmp 192.168.1.1:161 {
        username MYUSER
        password MYPASSWORD
        refresh 15s
        oid .1.3.6.1.4.1.8072.1.3.2.3.1.2.5.115.116.101.118.101
  }
}
```

## Install
1. clone repo 
1. get Coredns source
1. in coredns/plugin direction `ln -s REPO_LOCATION snmp`
2. add `snmp:snmp` to plugin/plugin.cfg (order matters, i think right after the `hosts` plugin is good)
4. go generate
5. go build
6. ./coredns
