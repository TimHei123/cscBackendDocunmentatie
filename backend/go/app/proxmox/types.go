package proxmox

// Node represents a Proxmox node in the cluster
type Node struct {
	Node string `json:"node"`
}

// VM represents a virtual machine in Proxmox
type VM struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	Mem       int64   `json:"mem"` // Adjusted to int64 to handle large values
	MaxMem    int64   `json:"maxmem"`
	CPU       float64 `json:"cpu"`
	CPUs      float64 `json:"cpus"`
	VMID      int     `json:"vmid"`
	NodeName  string  `json:"node"`
	DiskWrite int64   `json:"diskwrite"`
	DiskRead  int64   `json:"diskread"`
	NetIn     int64   `json:"netin"`
	NetOut    int64   `json:"netout"`
}

// VMResponse represents the structure of the JSON response containing multiple VMs
type VMResponse struct {
	VMs []VM `json:"vms"`
}

// VMRes represents the structure of the JSON response containing a single VM
type VMRes struct {
	Data VM `json:"data"` // Fix: single VM, not a slice
}

// Prox implements the HypervisorFunctions interface for Proxmox operations
type Prox struct{}
