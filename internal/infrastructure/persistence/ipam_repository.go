package persistence

import (
	"database/sql"
	"fmt"
	"net"
	"strings"

	"github.com/zinrai/ipam-mvp-go/internal/domain"
	"github.com/zinrai/ipam-mvp-go/internal/infrastructure/db"
)

type IPAMRepository struct {
	db *db.DB
}

func NewIPAMRepository(db *db.DB) *IPAMRepository {
	return &IPAMRepository{db: db}
}

func (r *IPAMRepository) CreateNetwork(network *domain.Network) error {
	query := `INSERT INTO networks (cidr, gateway) VALUES ($1, $2) RETURNING id`
	err := r.db.QueryRow(query, network.CIDR, network.Gateway.String()).Scan(&network.ID)
	if err != nil {
		return fmt.Errorf("failed to create network: %v", err)
	}
	return nil
}

func (r *IPAMRepository) GetNetwork(id int) (*domain.Network, error) {
	query := `SELECT id, cidr, gateway FROM networks WHERE id = $1`
	var network domain.Network
	var gatewayStr string
	err := r.db.QueryRow(query, id).Scan(&network.ID, &network.CIDR, &gatewayStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get network: %v", err)
	}
	network.Gateway = net.ParseIP(gatewayStr)
	return &network, nil
}

func (r *IPAMRepository) ListNetworks() ([]*domain.Network, error) {
	query := `SELECT id, cidr, gateway FROM networks`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %v", err)
	}
	defer rows.Close()

	var networks []*domain.Network
	for rows.Next() {
		var network domain.Network
		var gatewayStr string
		if err := rows.Scan(&network.ID, &network.CIDR, &gatewayStr); err != nil {
			return nil, fmt.Errorf("failed to scan network row: %v", err)
		}
		network.Gateway = net.ParseIP(gatewayStr)
		networks = append(networks, &network)
	}
	return networks, nil
}

func (r *IPAMRepository) AllocateIP(networkID int, requestedIP net.IP, hostname string) (*domain.IPAddress, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check if hostname is already in use for this network
	var existingHostname string
	err = tx.QueryRow("SELECT hostname FROM ip_addresses WHERE network_id = $1 AND hostname = $2", networkID, hostname).Scan(&existingHostname)
	if err != sql.ErrNoRows {
		if err == nil {
			return nil, fmt.Errorf("hostname %s is already in use in this network", hostname)
		}
		return nil, fmt.Errorf("failed to check hostname uniqueness: %v", err)
	}

	var ipAddress domain.IPAddress
	var addressStr string

	// First, get the network details including the gateway
	var networkCIDR, gatewayStr string
	err = tx.QueryRow("SELECT cidr, gateway FROM networks WHERE id = $1", networkID).Scan(&networkCIDR, &gatewayStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get network details: %v", err)
	}

	gatewayIP := net.ParseIP(gatewayStr)
	if gatewayIP == nil {
		return nil, fmt.Errorf("invalid gateway IP: %s", gatewayStr)
	}

	if requestedIP != nil {
		// Check if the requested IP is available and not the gateway
		if requestedIP.Equal(gatewayIP) {
			return nil, fmt.Errorf("cannot allocate gateway address %s", gatewayStr)
		}

		query := `
			SELECT id, address::text
			FROM ip_addresses
			WHERE network_id = $1 AND address = $2
			FOR UPDATE
		`
		err = tx.QueryRow(query, networkID, requestedIP.String()).Scan(&ipAddress.ID, &addressStr)
		if err == nil {
			return nil, fmt.Errorf("IP address %s is already allocated", requestedIP.String())
		} else if err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to check IP address: %v", err)
		}

		// If we reach here, the IP is not allocated and not the gateway, so we can allocate it
		query = `
			INSERT INTO ip_addresses (network_id, address, hostname, status)
			VALUES ($1, $2, $3, 'allocated')
			RETURNING id, address::text
		`
		err = tx.QueryRow(query, networkID, requestedIP.String(), hostname).Scan(&ipAddress.ID, &addressStr)
	} else {
		// Get the first available IP address, excluding the gateway
		_, ipNet, err := net.ParseCIDR(networkCIDR)
		if err != nil {
			return nil, fmt.Errorf("failed to parse network CIDR: %v", err)
		}

		for ip := nextIP(ipNet.IP); ipNet.Contains(ip); ip = nextIP(ip) {
			if ip.Equal(gatewayIP) {
				continue
			}

			query := `
				INSERT INTO ip_addresses (network_id, address, hostname, status)
				SELECT $1, $2, $3, 'allocated'
				WHERE NOT EXISTS (
					SELECT 1 FROM ip_addresses WHERE network_id = $1 AND address = $2
				)
				RETURNING id, address::text
			`
			err = tx.QueryRow(query, networkID, ip.String(), hostname).Scan(&ipAddress.ID, &addressStr)
			if err == nil {
				break
			}
			if err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to allocate IP address: %v", err)
			}
		}

		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no available IP addresses in the network")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP address: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	// CIDR表記からIPアドレス部分のみを抽出
	ipOnly := strings.Split(addressStr, "/")[0]

	ipAddress.NetworkID = networkID
	ipAddress.Address = net.ParseIP(ipOnly)
	if ipAddress.Address == nil {
		return nil, fmt.Errorf("failed to parse allocated IP address: %s", ipOnly)
	}
	ipAddress.Hostname = hostname
	ipAddress.Status = "allocated"
	return &ipAddress, nil
}

func (r *IPAMRepository) ReleaseIP(id int) error {
	query := `UPDATE ip_addresses SET status = 'available', hostname = NULL WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to release IP address: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("IP address not found")
	}
	return nil
}

func (r *IPAMRepository) GetIP(id int) (*domain.IPAddress, error) {
    query := `SELECT id, network_id, address::text, hostname, status FROM ip_addresses WHERE id = $1`
    var ip domain.IPAddress
    var addressStr string
    err := r.db.QueryRow(query, id).Scan(&ip.ID, &ip.NetworkID, &addressStr, &ip.Hostname, &ip.Status)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to get IP address: %v", err)
    }
    ip.Address = net.ParseIP(strings.Split(addressStr, "/")[0])
    return &ip, nil
}

func (r *IPAMRepository) ListIPs(networkID int) ([]*domain.IPAddress, error) {
    query := `SELECT id, network_id, address::text, hostname, status FROM ip_addresses WHERE network_id = $1`
    rows, err := r.db.Query(query, networkID)
    if err != nil {
        return nil, fmt.Errorf("failed to list IP addresses: %v", err)
    }
    defer rows.Close()

    var ips []*domain.IPAddress
    for rows.Next() {
        var ip domain.IPAddress
        var addressStr string
        if err := rows.Scan(&ip.ID, &ip.NetworkID, &addressStr, &ip.Hostname, &ip.Status); err != nil {
            return nil, fmt.Errorf("failed to scan IP address row: %v", err)
        }
        ip.Address = net.ParseIP(strings.Split(addressStr, "/")[0])
        ips = append(ips, &ip)
    }
    return ips, nil
}

func (r *IPAMRepository) UpdateIPHostname(id int, hostname string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	var networkID int
	err = tx.QueryRow("SELECT network_id FROM ip_addresses WHERE id = $1", id).Scan(&networkID)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %v", err)
	}

	// Check if hostname is already in use for this network
	var existingHostname string
	err = tx.QueryRow("SELECT hostname FROM ip_addresses WHERE network_id = $1 AND hostname = $2 AND id != $3", networkID, hostname, id).Scan(&existingHostname)
	if err != sql.ErrNoRows {
		if err == nil {
			return fmt.Errorf("hostname %s is already in use in this network", hostname)
		}
		return fmt.Errorf("failed to check hostname uniqueness: %v", err)
	}

	query := `UPDATE ip_addresses SET hostname = $1 WHERE id = $2`
	result, err := tx.Exec(query, hostname, id)
	if err != nil {
		return fmt.Errorf("failed to update IP hostname: %v", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("IP address not found")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// nextIP returns the next IP address in the subnet
func nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for j := len(next) - 1; j >= 0; j-- {
		next[j]++
		if next[j] > 0 {
			break
		}
	}
	return next
}
