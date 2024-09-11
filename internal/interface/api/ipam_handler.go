package api

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"

	"github.com/zinrai/ipam-mvp-go/internal/domain"
	"github.com/zinrai/ipam-mvp-go/internal/usecase"
)

type IPAMHandler struct {
	useCase *usecase.IPAMUseCase
}

func NewIPAMHandler(useCase *usecase.IPAMUseCase) *IPAMHandler {
	return &IPAMHandler{useCase: useCase}
}

func (h *IPAMHandler) HandleNetwork(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.createNetwork(w, r)
	case http.MethodGet:
		h.listNetworks(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *IPAMHandler) HandleIP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.allocateIP(w, r)
	case http.MethodDelete:
		h.releaseIP(w, r)
	case http.MethodGet:
		h.listIPs(w, r)
	case http.MethodPut:
		h.updateIPHostname(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *IPAMHandler) createNetwork(w http.ResponseWriter, r *http.Request) {
	var network domain.Network
	if err := json.NewDecoder(r.Body).Decode(&network); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.useCase.CreateNetwork(&network); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(network)
}

func (h *IPAMHandler) listNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.useCase.ListNetworks()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(networks)
}

func (h *IPAMHandler) allocateIP(w http.ResponseWriter, r *http.Request) {
	var request struct {
		NetworkID   int    `json:"network_id"`
		RequestedIP string `json:"requested_ip"`
		Hostname    string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var requestedIP net.IP
	if request.RequestedIP != "" {
		requestedIP = net.ParseIP(request.RequestedIP)
		if requestedIP == nil {
			http.Error(w, "Invalid IP address", http.StatusBadRequest)
			return
		}
	}

	ip, err := h.useCase.AllocateIP(request.NetworkID, requestedIP, request.Hostname)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		ID        int    `json:"id"`
		NetworkID int    `json:"network_id"`
		Address   string `json:"address"`
		Hostname  string `json:"hostname"`
		Status    string `json:"status"`
	}{
		ID:        ip.ID,
		NetworkID: ip.NetworkID,
		Address:   ip.Address.String(),
		Hostname:  ip.Hostname,
		Status:    ip.Status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *IPAMHandler) releaseIP(w http.ResponseWriter, r *http.Request) {
	ipID, err := strconv.Atoi(r.URL.Query().Get("ip_id"))
	if err != nil {
		http.Error(w, "Invalid IP ID", http.StatusBadRequest)
		return
	}
	if err := h.useCase.ReleaseIP(ipID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *IPAMHandler) listIPs(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(r.URL.Query().Get("network_id"))
	if err != nil {
		http.Error(w, "Invalid network ID", http.StatusBadRequest)
		return
	}
	ips, err := h.useCase.ListIPs(networkID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(ips)
}

func (h *IPAMHandler) updateIPHostname(w http.ResponseWriter, r *http.Request) {
	var request struct {
		IPID     int    `json:"ip_id"`
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.useCase.UpdateIPHostname(request.IPID, request.Hostname); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
