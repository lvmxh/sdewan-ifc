package openwrt

import (
	"encoding/json"
)

const (
	interfaceBaseURL = "sdewan/v1/"
)

type InterfaceClient struct {
	OpenwrtClient *openwrtClient
}

// Interface API struct
type Interface struct {
	Name            string   `json:"name,omitempty"`
	ReceivedPackets string   `json:"received_packets,omitempty"`
	SendPackets     string   `json:"send_packets,omitempty"`
	Status          string   `json:"status,omitempty"`
	MacAddress      string   `json:"mac_address,omitempty"`
	IpAddress       []string `json:"ip_address,omitempty"`
}

type AvailableInterfaces struct {
	Interfaces []Interface `json:"interfaces"`
}

// get available interfaces
func (s *InterfaceClient) GetAvailableInterfaces() (*AvailableInterfaces, error) {
	response, err := s.OpenwrtClient.Get(interfaceBaseURL + "interfaces")
	if err != nil {
		return nil, err
	}

	var servs AvailableInterfaces
	err2 := json.Unmarshal([]byte(response), &servs)
	if err2 != nil {
		return nil, err2
	}

	return &servs, nil
}
