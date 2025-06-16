package hypervisors

// HypervisorFunctions defines the interface for hypervisor operations
type HypervisorFunctions interface {
	ListAllVms() ([]byte, error)
	ListVmsUser([]int, string) ([]map[string]interface{}, error)
	CreateServer(string, int, int, int, string, string, string, string, string) (map[string]string, error)
	DeleteServer(int, string) (string, error)
}
