package observe

type Report struct {
	SchemaVersion  int         `json:"schemaVersion"`
	HostID         string      `json:"hostId"`
	HostName       string      `json:"hostName"`
	AgentVersion   string      `json:"agentVersion"`
	OS             string      `json:"os"`
	Arch           string      `json:"arch"`
	CPUPercent     float64     `json:"cpuPercent"`
	MemUsedBytes   int64       `json:"memUsedBytes"`
	MemTotalBytes  int64       `json:"memTotalBytes"`
	DiskUsedBytes  int64       `json:"diskUsedBytes"`
	DiskTotalBytes int64       `json:"diskTotalBytes"`
	Docker         DockerInfo  `json:"docker"`
	Containers     []Container `json:"containers"`
}

type DockerInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Error     string `json:"error,omitempty"`
}

type Container struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	State    string            `json:"state"`
	Health   string            `json:"health"`
	Labels   map[string]string `json:"labels"`
	Networks []string          `json:"networks"`
}

type AssignedJob struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	AppID       string `json:"appId"`
	ContainerID string `json:"containerId"`
	Tail        int    `json:"tail"`
	Slug        string `json:"slug,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	Template    string `json:"template,omitempty"`
	SpecHash    string `json:"specHash,omitempty"`
}

type PollResponse struct {
	Jobs []AssignedJob `json:"jobs"`
}

type JobResult struct {
	State   string `json:"state"`
	Summary string `json:"summary"`
	Logs    string `json:"logs,omitempty"`
	Code    string `json:"code,omitempty"`
}
