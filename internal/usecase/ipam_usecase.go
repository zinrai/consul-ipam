package usecase

import (
	"net"

	"github.com/zinrai/ipam-mvp-go/internal/domain"
)

type IPAMUseCase struct {
	repo domain.IPAMRepository
}

func NewIPAMUseCase(repo domain.IPAMRepository) *IPAMUseCase {
	return &IPAMUseCase{repo: repo}
}

func (uc *IPAMUseCase) CreateNetwork(network *domain.Network) error {
	return uc.repo.CreateNetwork(network)
}

func (uc *IPAMUseCase) GetNetwork(id int) (*domain.Network, error) {
	return uc.repo.GetNetwork(id)
}

func (uc *IPAMUseCase) ListNetworks() ([]*domain.Network, error) {
	return uc.repo.ListNetworks()
}

func (uc *IPAMUseCase) AllocateIP(networkID int, requestedIP net.IP, hostname string) (*domain.IPAddress, error) {
	return uc.repo.AllocateIP(networkID, requestedIP, hostname)
}

func (uc *IPAMUseCase) ReleaseIP(id int) error {
	return uc.repo.ReleaseIP(id)
}

func (uc *IPAMUseCase) GetIP(id int) (*domain.IPAddress, error) {
	return uc.repo.GetIP(id)
}

func (uc *IPAMUseCase) ListIPs(networkID int) ([]*domain.IPAddress, error) {
	return uc.repo.ListIPs(networkID)
}

func (uc *IPAMUseCase) UpdateIPHostname(id int, hostname string) error {
	return uc.repo.UpdateIPHostname(id, hostname)
}
