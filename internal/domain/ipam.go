package domain

import (
	"net"
)

type Network struct {
	ID      int
	CIDR    string
	Gateway net.IP
}

type IPAddress struct {
	ID        int
	NetworkID int
	Address   net.IP
	Hostname  string
	Status    string // e.g., "available", "allocated"
}

type IPAMRepository interface {
	CreateNetwork(network *Network) error
	GetNetwork(id int) (*Network, error)
	ListNetworks() ([]*Network, error)
	AllocateIP(networkID int, requestedIP net.IP, hostname string) (*IPAddress, error)
	ReleaseIP(id int) error
	GetIP(id int) (*IPAddress, error)
	ListIPs(networkID int) ([]*IPAddress, error)
	UpdateIPHostname(id int, hostname string) error
}
