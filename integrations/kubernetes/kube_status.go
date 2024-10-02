package kubernetes

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
)

func getDeploymentStatus(deploymentConditions []appsv1.DeploymentCondition) string {
	var mostRecentStatusTime time.Time
	deploymentStatus := bloopi_agent.StatusNoStatus

	for _, condition := range deploymentConditions {
		if mostRecentStatusTime.After(condition.LastUpdateTime.Time) {
			continue
		}

		mostRecentStatusTime = condition.LastUpdateTime.Time

		if condition.Type == appsv1.DeploymentProgressing {
			deploymentStatus = bloopi_agent.StatusNoStatus
		} else if condition.Type == appsv1.DeploymentAvailable {
			deploymentStatus = bloopi_agent.StatusGreen
		} else if condition.Type == appsv1.DeploymentReplicaFailure {
			deploymentStatus = bloopi_agent.StatusRed
		}
	}

	return deploymentStatus
}

func getPodStatus(conditions []v1.PodCondition) string {
	var mostRecentStatusTime time.Time
	podStatus := bloopi_agent.StatusNoStatus

	for _, condition := range conditions {
		if mostRecentStatusTime.After(condition.LastProbeTime.Time) {
			continue
		}

		mostRecentStatusTime = condition.LastProbeTime.Time

		switch condition.Type {
		case v1.AlphaNoCompatGuaranteeDisruptionTarget:
			podStatus = bloopi_agent.StatusRed

		case v1.ContainersReady, v1.PodReady:
			podStatus = bloopi_agent.StatusGreen

		case v1.PodInitialized, v1.PodScheduled:
			podStatus = bloopi_agent.StatusOrange
		}
	}

	return podStatus
}
