package gcp

import "dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"

func getComputeStatus(status string) string {
	switch status {
	case "RUNNING", "READY":
		return bloopi_agent.StatusGreen

	case "DEPROVISIONING", "STOPPED", "STOPPING", "SUSPENDED", "SUSPENDING", "TERMINATED", "INVALID", "DRAINING":
		return bloopi_agent.StatusRed

	case "PROVISIONING", "STAGING", "REPAIRING", "DELETING", "CREATING":
		return bloopi_agent.StatusOrange
	}

	return bloopi_agent.StatusNoStatus
}
