package proxmox

import (
	auth "CSC-SelfServiceBackend/app/auth"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

// CreateServer creates a new VM on Proxmox from a template and adjusts resources if needed.
func (h Prox) CreateServer(name string, memory int, cores int, size int, user_sid string, user_full_name string, serverDescription string, selectedOs string, subdomain string) (map[string]string, error) {
	serverURL := os.Getenv("PROXMOX_SERVER_URL")
	nodeName := os.Getenv("PROXMOX_NODE")
	templateVMID := 9000
	diskBus := "scsi0"

	//maximum resource limits
	maxMemory, err := strconv.Atoi(os.Getenv("PROXMOX_MAX_MEMORY"))
	if err != nil {
		return nil, fmt.Errorf("invalid PROXMOX_MAX_MEMORY value: %v", err)
	}
	if memory > maxMemory {
		memory = maxMemory
	}
	maxCpuCores, err := strconv.Atoi(os.Getenv("PROXMOX_MAX_CPU_CORES"))
	if err != nil {
		return nil, fmt.Errorf("invalid PROXMOX_MAX_CPU_CORES value: %v", err)
	}
	if cores > maxCpuCores {
		cores = maxCpuCores
	}
	maxDiskSize, err := strconv.Atoi(os.Getenv("PROXMOX_MAX_DISK_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid PROXMOX_MAX_DISK_SIZE value: %v", err)
	}
	if size > maxDiskSize {
		size = maxDiskSize
	}

	client := resty.New().SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})

	authCookie, csrfToken, err := ConnectToServer()
	if err != nil {
		return nil, fmt.Errorf("authentication error: %v", err)
	}

	// Get next available VMID
	vmidResp, err := client.R().
		SetHeader("Cookie", authCookie).
		SetHeader("CSRFPreventionToken", csrfToken).
		Get(fmt.Sprintf("%s/api2/json/cluster/nextid", serverURL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch next VMID: %v", err)
	}
	var vmidData struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(vmidResp.Body(), &vmidData); err != nil {
		return nil, fmt.Errorf("failed to parse nextid response: %v", err)
	}
	vmid, err := strconv.Atoi(vmidData.Data)
	if err != nil {
		return nil, fmt.Errorf("invalid VMID returned: %v", err)
	}

	// Fetch template config
	templateResp, err := client.R().
		SetHeader("Cookie", authCookie).
		SetHeader("CSRFPreventionToken", csrfToken).
		Get(fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/config", serverURL, nodeName, templateVMID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template config: %v", err)
	}
	var config struct {
		Data struct {
			Cores  interface{} `json:"cores"`
			Memory interface{} `json:"memory"`
			Scsi0  string      `json:"scsi0"`
		} `json:"data"`
	}
	if err := json.Unmarshal(templateResp.Body(), &config); err != nil {
		return nil, fmt.Errorf("failed to parse template config: %v", err)
	}

	// Convert cores and memory from interface{} to int safely
	var templateCores, templateMemory int
	switch v := config.Data.Cores.(type) {
	case string:
		templateCores, _ = strconv.Atoi(v)
	case float64:
		templateCores = int(v)
	default:
		templateCores = 0
	}
	switch v := config.Data.Memory.(type) {
	case string:
		templateMemory, _ = strconv.Atoi(v)
	case float64:
		templateMemory = int(v)
	default:
		templateMemory = 0
	}

	// Parse disk size from scsi0 string, e.g. "local:9000/vm-9000-disk-0.qcow2,size=32G"
	templateDiskSize := 0
	if diskConf := config.Data.Scsi0; diskConf != "" {
		// Look for size=NUMBERG or size=NUMBERg pattern
		re := regexp.MustCompile(`size=(\d+)[Gg]`)
		matches := re.FindStringSubmatch(diskConf)
		if len(matches) == 2 {
			templateDiskSize, _ = strconv.Atoi(matches[1])
		}
	}

	// Clone the VM
	cloneForm := map[string]string{
		"newid": strconv.Itoa(vmid),
		"name":  name,
		"full":  "1",
	}
	_, err = client.R().
		SetHeader("Cookie", authCookie).
		SetHeader("CSRFPreventionToken", csrfToken).
		SetFormData(cloneForm).
		Post(fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/clone", serverURL, nodeName, templateVMID))
	if err != nil {
		return nil, fmt.Errorf("failed to clone VM: %v", err)
	}

	//sample running the setup script in the template
	// Build the cloud-init script to run on first boot

	/* userData := fmt.Sprintf(`#!/bin/bash
	   /path/to/startup.sh %s %s %s
	   `, param1, param2, param3)*/

	// Update configuration if resources requested exceed template
	configForm := map[string]string{
		"sockets":    "1",
		"numa":       "0",
		"cpu":        "x86-64-v2-AES",
		"scsihw":     "virtio-scsi-single",
		"net0":       "virtio,bridge=vmbr1,firewall=1",
		"ciuser":     "ubuntu",
		"cipassword": "ubuntu",
		//"user-data":  userData,
	}
	if memory > templateMemory {
		configForm["memory"] = strconv.Itoa(memory)
	}
	if cores > templateCores {
		configForm["cores"] = strconv.Itoa(cores)
	}
	_, err = client.R().
		SetHeader("Cookie", authCookie).
		SetHeader("CSRFPreventionToken", csrfToken).
		SetFormData(configForm).
		Put(fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/config", serverURL, nodeName, vmid))
	if err != nil {
		return nil, fmt.Errorf("config update failed: %v", err)
	}

	// Resize disk if needed
	if size > templateDiskSize {
		resizeForm := map[string]string{
			"disk": diskBus,
			"size": fmt.Sprintf("+%dG", size-templateDiskSize),
		}
		_, err = client.R().
			SetHeader("Cookie", authCookie).
			SetHeader("CSRFPreventionToken", csrfToken).
			SetFormData(resizeForm).
			Put(fmt.Sprintf("%s/api2/json/nodes/%s/qemu/%d/resize", serverURL, nodeName, vmid))
		if err != nil {
			return nil, fmt.Errorf("disk resize failed: %v", err)
		}
	}

	// Save to DB
	db, err := auth.ConnectToDB()
	if err != nil {
		return nil, err
	}
	stmt, err := db.Prepare("INSERT INTO virtual_machines (vmid, user_sid, full_name, description, expires_at, selectedOs, subdomain) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	expiresAt := time.Unix(time.Now().Add(time.Hour*24*7*24).Unix(), 0).Format("2006-01-02 15:04:05")
	_, err = stmt.Exec(vmid, user_sid, user_full_name, serverDescription, expiresAt, selectedOs, subdomain)
	if err != nil {
		return nil, err
	}

	return map[string]string{"Message": "VM cloned and configured successfully"}, nil
}
