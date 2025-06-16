package proxmox

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-resty/resty/v2"
)

// ListAllVms retrieves all VMs from the Proxmox server
func (h Prox) ListAllVms() ([]byte, error) {
	serverURL := os.Getenv("PROXMOX_SERVER_URL")

	restyClient := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	authCookie, csrfToken, err := ConnectToServer()

	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}

	nodesResp, err := restyClient.R().
		SetHeader("Cookie", authCookie).
		SetHeader("CSRFPreventionToken", csrfToken).
		Get(serverURL + "/api2/json/nodes")

	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve nodes: %v", err)
	}

	var nodesData struct {
		Data []Node `json:"data"`
	}
	if err := json.Unmarshal(nodesResp.Body(), &nodesData); err != nil {
		return nil, fmt.Errorf("Failed to parse nodes response: %v", err)
	}

	if len(nodesData.Data) == 0 {
		return nil, fmt.Errorf("No nodes found in the Proxmox cluster")
	}

	// Iterate over nodes and list all VMs
	allVMs := []VM{}
	for _, node := range nodesData.Data {
		vmsResp, err := restyClient.R().
			SetHeader("Cookie", authCookie).
			Get(fmt.Sprintf("%s/api2/json/nodes/%s/qemu", serverURL, node.Node))

		if err != nil {
			return nil, fmt.Errorf("Failed to list VMs for node %s: %v", node.Node, err)
		}

		var vmsData struct {
			Data []VM `json:"data"`
		}
		if err := json.Unmarshal(vmsResp.Body(), &vmsData); err != nil {
			return nil, fmt.Errorf("Failed to parse VMs response for node %s: %v", node.Node, err)
		}

		allVMs = append(allVMs, vmsData.Data...)
	}

	var response VMResponse
	response.VMs = allVMs

	jsonData, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal JSON: %v", err)
	}

	return jsonData, nil
}
