package jobs

func Allow(jobType string) bool {
	switch jobType {
	case "observe_logs", "deploy", "start", "stop", "restart", "delete":
		return true
	default:
		return false
	}
}
