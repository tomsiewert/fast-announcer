package config

type IP struct {
	Family    string `json:"family"`
	Address   string `json:"address"`
}

type IPNetwork struct {
	Family  string `json:"family"`
	Network string `json:"network"`
	NextHop string `json:"next_hop"`
}

type Configuration struct {
	DomainID    int         `json:"domain_id"`
	IPAddresses []IP        `json:"ip_addresses"`
	IPNetworks  []IPNetwork `json:"ip_networks"`
	MacAddress  string      `json:"mac_address"`
	Interface   string      `json:"interface"`
	Table       int         `json:"table"`
}
