package vmware

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

func powerOn(session, vmId string) bool {
	defer timeTrack(time.Now(), "powerOn")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("POST", baseURL+"/api/vcenter/vm/"+vmId+"/power?action=start", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	if resp.StatusCode != 200 {
		return false
	}

	return true
}

func powerOff(session, vmId string) bool {
	defer timeTrack(time.Now(), "powerOff")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("POST", baseURL+"/api/vcenter/vm/"+vmId+"/power?action=stop", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	if resp.StatusCode != 200 {
		return false
	}

	return true
}

func forcePowerOff(session, vmId string) bool {
	defer timeTrack(time.Now(), "forcePowerOff")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("POST", baseURL+"/api/vcenter/vm/"+vmId+"/power?action=stop", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	if resp.StatusCode != 200 {
		return false
	}

	return true
}

func reset(session, vmId string) bool {
	defer timeTrack(time.Now(), "reset")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("POST", baseURL+"/api/vcenter/vm/"+vmId+"/power?action=reset", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	if resp.StatusCode != 200 {
		return false
	}

	return true
}

func suspend(session, vmId string) bool {
	defer timeTrack(time.Now(), "suspend")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("POST", baseURL+"/api/vcenter/vm/"+vmId+"/power?action=suspend", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	if resp.StatusCode != 200 {
		return false
	}

	return true
}

func getPowerState(session, vmId string) string {
	defer timeTrack(time.Now(), "getPowerState")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("GET", baseURL+"/api/vcenter/vm/"+vmId+"/power", nil)
	if err != nil {
		log.Println("Error creating request: ", err)
	}

	req.Header.Add("vmware-api-session-id", session)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error sending request: ", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading response: ", err)
	}

	var powerState struct {
		State string `json:"state"`
	}

	err = json.Unmarshal(body, &powerState)
	if err != nil {
		log.Println("Error unmarshalling response: ", err)
	}

	defer resp.Body.Close()
	return powerState.State
}
