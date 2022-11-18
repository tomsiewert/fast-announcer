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

	switch domainAction {
	case "post-start":
		for _, ip := range conf.IPAddresses {
			parsedIp, parsedNet, _ := net.ParseCIDR(ip.Address)

			log.Println("Add src rule for " + parsedNet.String())
			srcRule := createRule(ip.Family, parsedNet, nil, conf.Table)
			if err := netlink.RuleAdd(srcRule); err != nil {
				log.Fatal(err)
			}

			log.Println("Add dst rule for " + parsedNet.String())
			dstRule := createRule(ip.Family, nil, parsedNet, conf.Table)
			if err := netlink.RuleAdd(dstRule); err != nil {
				log.Fatal(err)
			}

			gw := linkLocal
			if ip.Family != "ipv6" {
				gw = nil
			}

			log.Println("Add route for " + parsedNet.String())
			route := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       parsedNet,
				Gw:        gw,
				Table:     conf.Table,
			}
			if err := netlink.RouteAdd(&route); err != nil {
				log.Fatal(err)
			}

			if ip.Family == "ipv4" {
				log.Println("Add neighbour for " + parsedIp.String())
				neigh := netlink.Neigh{
					LinkIndex:    link.Attrs().Index,
					State:        netlink.NUD_PERMANENT,
					IP:           parsedIp,
					HardwareAddr: parseMac(conf.MacAddress),
				}
				if err := netlink.NeighAdd(&neigh); err != nil {
					log.Fatal(err)
				}
			}
		}

		for _, network := range conf.IPNetworks {
			_, parsedNet, _ := net.ParseCIDR(network.Network)

			log.Println("Add src rule for " + parsedNet.String())
			srcRule := createRule(network.Family, parsedNet, nil, conf.Table)
			if err := netlink.RuleAdd(srcRule); err != nil {
				log.Fatal(err)
			}

			log.Println("Add dst rule for " + parsedNet.String())
			dstRule := createRule(network.Family, nil, parsedNet, conf.Table)
			if err := netlink.RuleAdd(dstRule); err != nil {
				log.Fatal(err)
			}

			gw := linkLocal
			if network.Family == "ipv4" {
				gw = net.ParseIP(network.NextHop)
			}

			log.Println("Add route for " + parsedNet.String())
			route := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       parsedNet,
				Gw:        gw,
				Flags:     int(netlink.FLAG_ONLINK),
				Table:     conf.Table,
			}
			if err := netlink.RouteAdd(&route); err != nil {
				log.Fatal(err)
			}
		}
	case "post-stop":
		for _, ip := range conf.IPAddresses {
			parsedIp, parsedNet, _ := net.ParseCIDR(ip.Address)

			log.Println("Delete src rule for " + parsedNet.String())
			srcRule := createRule(ip.Family, parsedNet, nil, conf.Table)
			if err := netlink.RuleDel(srcRule); err != nil {
				log.Fatal(err)
			}

			log.Println("Delete dst rule for " + parsedNet.String())
			dstRule := createRule(ip.Family, nil, parsedNet, conf.Table)
			if err := netlink.RuleDel(dstRule); err != nil {
				log.Fatal(err)
			}

			gw := linkLocal
			if ip.Family != "ipv6" {
				gw = nil
			}

			log.Println("Delete route for " + parsedNet.String())
			route := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       parsedNet,
				Gw:        gw,
				Table:     conf.Table,
			}
			if err := netlink.RouteDel(&route); err != nil {
				log.Fatal(err)
			}

			if ip.Family == "ipv4" {
				log.Println("Delete neighbour for " + parsedIp.String())
				neigh := netlink.Neigh{
					LinkIndex:    link.Attrs().Index,
					State:        netlink.NUD_PERMANENT,
					IP:           parsedIp,
					HardwareAddr: parseMac(conf.MacAddress),
				}
				if err := netlink.NeighDel(&neigh); err != nil {
					log.Fatal(err)
				}
			}
		}

		for _, network := range conf.IPNetworks {
			_, parsedNet, _ := net.ParseCIDR(network.Network)

			log.Println("Add src rule for " + parsedNet.String())
			srcRule := createRule(network.Family, parsedNet, nil, conf.Table)
			if err := netlink.RuleAdd(srcRule); err != nil {
				log.Fatal(err)
			}

			log.Println("Add dst rule for " + parsedNet.String())
			dstRule := createRule(network.Family, nil, parsedNet, conf.Table)
			if err := netlink.RuleAdd(dstRule); err != nil {
				log.Fatal(err)
			}

			gw := linkLocal
			if network.Family == "ipv4" {
				gw = net.ParseIP(network.NextHop)
			}

			log.Println("Add route for " + parsedNet.String())
			route := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Dst:       parsedNet,
				Gw:        gw,
				Flags:     int(netlink.FLAG_ONLINK),
				Table:     conf.Table,
			}
			if err := netlink.RouteDel(&route); err != nil {
				log.Fatal(err)
			}
		}
	default:
		log.Println("No action matched.")
	}
}
