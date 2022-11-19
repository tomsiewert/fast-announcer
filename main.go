package main

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"os"

	"github.com/mdlayher/netx/eui64"
	"github.com/tomsiewert/fast-announcer/config"
	"github.com/vishvananda/netlink"
)

func readConfig(path string) *config.Configuration {
	var configuration config.Configuration
	configFile, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()

	configContent, _ := io.ReadAll(configFile)
	json.Unmarshal(configContent, &configuration)
	return &configuration
}

func parseMac(s string) net.HardwareAddr {
	mac, err := net.ParseMAC(s)
	if err != nil {
		log.Fatal(err)
	}
	return mac
}

func createRule(family string, source *net.IPNet, dst *net.IPNet, table int) *netlink.Rule {
	rule := netlink.NewRule()
	switch family {
	case "ipv4":
		rule.Family = netlink.FAMILY_V4
	case "ipv6":
		rule.Family = netlink.FAMILY_V6
	default:
		log.Println("No family defined for " + source.String() + " " + dst.String())
	}
	if source != nil {
		rule.Src = source
	}
	if dst != nil {
		rule.Dst = dst
	}
	if table != 0 {
		rule.Table = table
	}
	return rule
}

func main() {
	args := os.Args
	domainID := args[1]
	domainAction := args[2]

	if domainAction == "pre-start" || domainAction == "pre-stop" {
		os.Exit(0)
	}

	conf := readConfig("/var/lib/infra/network/" + domainID + ".json")

	parsedMac := parseMac(conf.MacAddress)
	linkLocal, err := eui64.ParseMAC(net.ParseIP("fe80::"), parsedMac)
	if err != nil {
		log.Fatal(err)
	}

	link, err := netlink.LinkByName(conf.Interface)
	if err != nil {
		log.Fatal(err)
	}

	for _, ip := range conf.IPAddresses {
		parsedIp, parsedNet, _ := net.ParseCIDR(ip.Address)

		srcRule := createRule(ip.Family, parsedNet, nil, conf.Table)
		dstRule := createRule(ip.Family, nil, parsedNet, conf.Table)

		gw := linkLocal
		if ip.Family != "ipv6" {
			gw = nil
		}

		route := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       parsedNet,
			Gw:        gw,
			Table:     conf.Table,
		}

		var neigh netlink.Neigh
		if ip.Family == "ipv4" {
			neigh = netlink.Neigh{
				LinkIndex:    link.Attrs().Index,
				State:        netlink.NUD_PERMANENT,
				IP:           parsedIp,
				HardwareAddr: parseMac(conf.MacAddress),
			}
		}

		switch domainAction {
		case "post-start":
			log.Println("Add src rule for " + parsedIp.String())
			if err := netlink.RuleAdd(srcRule); err != nil {
				log.Println(err)
			}
			log.Println("Add dst rule for " + parsedIp.String())
			if err := netlink.RuleAdd(dstRule); err != nil {
				log.Println(err)
			}
			log.Println("Add route for " + parsedIp.String())
			if err := netlink.RouteAdd(&route); err != nil {
				log.Println(err)
			}
			if ip.Family == "ipv4" {
				log.Println("Add neigh for " + parsedIp.String())
				if err := netlink.NeighAdd(&neigh); err != nil {
					log.Println(err)
				}
			}
		case "post-stop":
			log.Println("Del src rule for " + parsedIp.String())
			if err := netlink.RuleDel(srcRule); err != nil {
				log.Println(err)
			}
			log.Println("Del dst rule for " + parsedIp.String())
			if err := netlink.RuleDel(dstRule); err != nil {
				log.Println(err)
			}
			log.Println("Del route for " + parsedIp.String())
			if err := netlink.RouteDel(&route); err != nil {
				log.Println(err)
			}
			if ip.Family == "ipv4" {
				log.Println("Del neigh for " + parsedIp.String())
				if err := netlink.NeighDel(&neigh); err != nil {
					log.Println(err)
				}
			}
		default:
			log.Fatal("No matching action found")
		}
	}

	for _, network := range conf.IPNetworks {
		_, parsedNet, _ := net.ParseCIDR(network.Network)

		srcRule := createRule(network.Family, parsedNet, nil, conf.Table)
		dstRule := createRule(network.Family, nil, parsedNet, conf.Table)

		gw := linkLocal
		if network.Family == "ipv4" {
			gw = net.ParseIP(network.NextHop)
		}

		route := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       parsedNet,
			Gw:        gw,
			Flags:     int(netlink.FLAG_ONLINK),
			Table:     conf.Table,
		}
		switch domainAction {
		case "post-start":
			log.Println("Add src rule for " + parsedNet.String())
			if err := netlink.RuleAdd(srcRule); err != nil {
				log.Println(err)
			}
			log.Println("Add dst rule for " + parsedNet.String())
			if err := netlink.RuleAdd(dstRule); err != nil {
				log.Println(err)
			}
			log.Println("Add route for " + parsedNet.String())
			if err := netlink.RouteAdd(&route); err != nil {
				log.Println(err)
			}
		case "post-stop":
			log.Println("Del src rule for " + parsedNet.String())
			if err := netlink.RuleDel(srcRule); err != nil {
				log.Println(err)
			}
			log.Println("Del dst rule for " + parsedNet.String())
			if err := netlink.RuleDel(dstRule); err != nil {
				log.Println(err)
			}
			log.Println("Del route for " + parsedNet.String())
			if err := netlink.RouteDel(&route); err != nil {
				log.Println(err)
			}
		default:
			log.Fatal("No matching action found")
		}
	}
}
