package vmware

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

type vCenterTemplate struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func getTemplatesFromVCenter(session string) []vCenterTemplate {
	defer timeTrack(time.Now(), "getTemplatesFromVCenter")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("GET", baseURL+"/api/vcenter/vm-template/library-items", nil)
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

	var templates []vCenterTemplate
	err = json.Unmarshal(body, &templates)
	if err != nil {
		log.Println("Error unmarshalling response: ", err)
	}

	defer resp.Body.Close()
	return templates
}

func fetchTemplateLibraryIdsFromVCenter(session string) []string {
	defer timeTrack(time.Now(), "fetchTemplateLibraryIdsFromVCenter")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("GET", baseURL+"/api/vcenter/vm-template/library-items", nil)
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

	var templates []vCenterTemplate
	err = json.Unmarshal(body, &templates)
	if err != nil {
		log.Println("Error unmarshalling response: ", err)
	}

	var templateIds []string
	for _, template := range templates {
		templateIds = append(templateIds, template.Id)
	}

	defer resp.Body.Close()
	return templateIds
}

func updateTemplatesFromVCenter(session string) {
	defer timeTrack(time.Now(), "updateTemplatesFromVCenter")
	client := createVCenterHTTPClient()
	baseURL := getEnvVar("VCENTER_URL")

	req, err := http.NewRequest("GET", baseURL+"/api/vcenter/vm-template/library-items", nil)
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

	var templates []vCenterTemplate
	err = json.Unmarshal(body, &templates)
	if err != nil {
		log.Println("Error unmarshalling response: ", err)
	}

	for _, template := range templates {
		setToRedis(template.Name, template.Id, 0)
	}

	defer resp.Body.Close()
}
