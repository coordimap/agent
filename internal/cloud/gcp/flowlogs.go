package gcp

import (
	cloudutils "cleye/internal/cloud/utils"
	"cleye/utils"
	"encoding/json"
	"fmt"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	gcpModel "dev.azure.com/bloopi/bloopi/_git/shared_models.git/gcp"
	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
	"google.golang.org/api/logging/v2"
)

func (crawler *gcpCrawler) getFlowLogsRelationships() ([]*bloopi_agent.Element, error) {
	allFoundRelationships := []*bloopi_agent.Element{}
	startTime := time.Now().UTC().Add(-5 * time.Second)
	endTime := startTime.Add(crawler.crawlInterval - 5*time.Second)

	timeFilter := fmt.Sprintf(`timestamp >= "%s" AND timestamp <= "%s"`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339))

	filter := fmt.Sprintf("resource.type=\"gce_subnetwork\" AND jsonPayload.connection.src_ip!=\"\" AND jsonPayload.connection.dest_ip!=\"\" AND %s", timeFilter)

	entries, errEntries := crawler.logClient.Entries.List(&logging.ListLogEntriesRequest{
		ResourceNames: []string{fmt.Sprintf("projects/%s", crawler.ConfiguredProjectID)},
		Filter:        filter,
	}).Do()
	if errEntries != nil {
		return nil, errEntries
	}

	for _, logEntry := range entries.Entries {
		var jsonPayload flowJSONStructure
		errUnmarshal := json.Unmarshal(logEntry.JsonPayload, &jsonPayload)

		if errUnmarshal != nil {
			return nil, errUnmarshal
		}

		crawlTime, errCrawlTime := time.Parse(time.RFC3339, jsonPayload.StartTime)
		if errCrawlTime != nil {
			crawlTime = time.Now().UTC()
		}

		if jsonPayload.SrcInstance.VmName != "" && jsonPayload.DstInstance.VmName != "" {

			srcVmInternalID := cloudutils.CreateGCPInternalName(crawler.dataSource.DataSourceID, jsonPayload.DstInstance.Zone, gcpModel.TypeVMInstance, jsonPayload.DstInstance.VmName)
			dstVmInternalID := cloudutils.CreateGCPInternalName(crawler.dataSource.DataSourceID, jsonPayload.DstInstance.Zone, gcpModel.TypeVMInstance, jsonPayload.DstInstance.VmName)

			vmRel, errVmRel := utils.CreateRelationship(srcVmInternalID, dstVmInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.FlowTypeRelation, crawlTime)
			if errVmRel == nil {
				allFoundRelationships = append(allFoundRelationships, vmRel)
			}
		}

		// check for gke src and dest
		if jsonPayload.SrcGkeDetails.Cluster.ClusterName != "" && jsonPayload.DstGkeDetails.Cluster.ClusterName != "" {
			if jsonPayload.SrcGkeDetails.Pod.Name != "" && jsonPayload.SrcGkeDetails.Pod.Namespace != "" && jsonPayload.DstGkeDetails.Pod.Namespace != "" && jsonPayload.DstGkeDetails.Pod.Name != "" {
				srcDSID, errSrcDSID := cloudutils.GetMappingDataSourceID(crawler.externalMappings, fmt.Sprintf("%s-%s", jsonPayload.SrcGkeDetails.Cluster.ClusterLocation, jsonPayload.SrcGkeDetails.Cluster.ClusterName))
				if errSrcDSID != nil {
					continue
				}
				dstDSID, errDstDSID := cloudutils.GetMappingDataSourceID(crawler.externalMappings, fmt.Sprintf("%s-%s", jsonPayload.DstGkeDetails.Cluster.ClusterLocation, jsonPayload.DstGkeDetails.Cluster.ClusterName))
				if errDstDSID != nil {
					continue
				}

				srcPodInternalName := cloudutils.CreateKubeInternalName(srcDSID, jsonPayload.SrcGkeDetails.Pod.Namespace, kubernetes.TypePod, jsonPayload.SrcGkeDetails.Pod.Name)
				dstPodInternalName := cloudutils.CreateKubeInternalName(dstDSID, jsonPayload.DstGkeDetails.Pod.Namespace, kubernetes.TypePod, jsonPayload.DstGkeDetails.Pod.Name)

				rel, errRel := utils.CreateRelationship(srcPodInternalName, dstPodInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.FlowTypeRelation, crawlTime)
				if errRel == nil {
					allFoundRelationships = append(allFoundRelationships, rel)
				}
			}
			// TODO: check for workload relationships in the pod
			// TODO: check for service relationships
		}
	}

	return allFoundRelationships, nil
}
