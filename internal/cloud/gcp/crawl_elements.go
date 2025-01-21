package gcp

import (
	cloudutils "cleye/internal/cloud/utils"
	"cleye/utils"
	"context"
	"fmt"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	gcpModel "dev.azure.com/bloopi/bloopi/_git/shared_models.git/gcp"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/compute/v1"
	run "google.golang.org/api/run/v1"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"google.golang.org/api/storage/v1"
)

func (gcpCrawler *gcpCrawler) GetLocationsAndZones(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allElems := []*bloopi_agent.Element{}

	regionsCall := client.Regions.List(gcpCrawler.ConfiguredProjectID)
	regionList, err := regionsCall.Do()
	if err != nil {
		return allElems, fmt.Errorf("could not list regions becaue %v", err)
	}

	// Iterate through regions and their zones
	for _, region := range regionList.Items {
		regionInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, "", gcpModel.TypeRegion, region.Name)
		regionElem, errRegionElem := utils.CreateElement(region, region.Name, regionInternalName, gcpModel.TypeRegion, bloopi_agent.StatusNoStatus, "", crawlTime)
		if errRegionElem == nil {
			allElems = append(allElems, regionElem)
		}

		zonesCall := client.Zones.List(gcpCrawler.ConfiguredProjectID)
		zoneList, err := zonesCall.Filter(fmt.Sprintf("name=%s*", region.Name)).Do()
		if err != nil {
			log.Printf("Error getting zones for region %s: %v", region.Name, err)
			continue
		}

		for _, zone := range zoneList.Items {
			zoneInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, "", gcpModel.TypeZone, zone.Name)
			zoneElem, errZoneElem := utils.CreateElement(zone, zone.Name, zoneInternalName, gcpModel.TypeZone, bloopi_agent.StatusNoStatus, "", crawlTime)
			if errZoneElem == nil {
				allElems = append(allElems, zoneElem)
			}

			rel, errRel := utils.CreateRelationship(regionInternalName, zoneInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				allElems = append(allElems, rel)
			}
		}
	}
	return allElems, nil
}

func (gcpCrawler *gcpCrawler) GetBuckets(crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allBucketElements := []*bloopi_agent.Element{}

	client, err := storage.NewService(context.Background(), gcpCrawler.clientOpts...)
	if err != nil {
		return allBucketElements, fmt.Errorf("could not create storage client because %v", err)
	}

	buckets, errBuckets := client.Buckets.List(gcpCrawler.ConfiguredProjectID).Do()
	if errBuckets != nil {
		return allBucketElements, fmt.Errorf("could not retrieve all buckets because %v", errBuckets)
	}

	for _, bucket := range buckets.Items {
		zone := strings.ToLower(bucket.Location)
		zoneInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, "", gcpModel.TypeZone, zone)
		bucketInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeBucket, bucket.Name)
		elem, errElem := utils.CreateElement(bucket, bucket.Name, bucketInternalName, gcpModel.TypeBucket, bloopi_agent.StatusNoStatus, "", crawlTime)
		if errElem == nil {
			allBucketElements = append(allBucketElements, elem)
		}

		rel, errRel := utils.CreateRelationship(zoneInternalName, bucketInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			allBucketElements = append(allBucketElements, rel)
		}
	}

	return allBucketElements, nil
}

func (gcpCrawler *gcpCrawler) GetCloudRuns(crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allCloudRuns := []*bloopi_agent.Element{}

	client, errClient := run.NewService(context.Background(), gcpCrawler.clientOpts...)
	if errClient != nil {
		return allCloudRuns, fmt.Errorf("could not create a cloud run client because %v", errClient)
	}

	parent := fmt.Sprintf("projects/%s/locations/-", gcpCrawler.ConfiguredProjectID)
	services, errServices := client.Projects.Locations.Services.List(parent).Do()
	if errServices != nil {
		return allCloudRuns, fmt.Errorf("failed to list Cloud Run services: %v", errServices)
	}

	for _, service := range services.Items {
		elem, errElem := utils.CreateElement(service, service.Metadata.Name, service.Metadata.Name, gcpModel.TypeCloudRun, bloopi_agent.StatusNoStatus, service.Metadata.ResourceVersion, crawlTime)
		if errElem == nil {
			allCloudRuns = append(allCloudRuns, elem)
		}
	}

	return allCloudRuns, nil
}

func (gcpCrawler *gcpCrawler) GetComputeElems(crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	logger := log.With().Str("DataSourceType", "gcp").Str("ProjectID", gcpCrawler.ConfiguredProjectID).Str("DataSourceID", gcpCrawler.dataSource.DataSourceID).Logger()
	allComputeElems := []*bloopi_agent.Element{}
	client, errClient := createComputeClient(gcpCrawler.clientOpts)
	if errClient != nil {
		return allComputeElems, fmt.Errorf("could not create a compute instance because %v", errClient)
	}

	regionsAndZones, errRegionsAndZones := gcpCrawler.GetLocationsAndZones(client, crawlTime)
	if errRegionsAndZones == nil {
		allComputeElems = append(allComputeElems, regionsAndZones...)
	}

	vmInstanceElems, errVMInstanceElems := gcpCrawler.GetVMInstances(client, crawlTime)
	if errVMInstanceElems != nil {
		logger.Err(errVMInstanceElems).Msg("could not retrieve VM instances")
	} else {
		allComputeElems = append(allComputeElems, vmInstanceElems...)
	}

	nodeGroupElems, errNodeGroupElems := gcpCrawler.getNodeGroups(client, crawlTime)
	if errNodeGroupElems != nil {
		logger.Err(errNodeGroupElems).Msg("could not retrieve node group")
	} else {
		allComputeElems = append(allComputeElems, nodeGroupElems...)
	}

	instanceGroupElems, errInstanceGroupElems := gcpCrawler.getInstanceGroups(client, crawlTime)
	if errInstanceGroupElems != nil {
		logger.Err(errInstanceGroupElems).Msg("could not retrieve instance groups")
	} else {
		allComputeElems = append(allComputeElems, instanceGroupElems...)
	}

	diskElems, errDiskElems := gcpCrawler.getDisks(client, crawlTime)
	if errDiskElems != nil {
		logger.Err(errDiskElems).Msg("could not retrieve disks")
	} else {
		allComputeElems = append(allComputeElems, diskElems...)
	}

	networkElems, errNetworkElems := gcpCrawler.getNetworks(client, crawlTime)
	if errNetworkElems != nil {
		logger.Err(errNetworkElems).Msg("could not retrieve networks")
	} else {
		allComputeElems = append(allComputeElems, networkElems...)
	}

	subnetworkElems, errSubnetworkElems := gcpCrawler.getSubNetworks(client, crawlTime)
	if errSubnetworkElems != nil {
		logger.Err(errSubnetworkElems).Msg("could not retrieve networks")
	} else {
		allComputeElems = append(allComputeElems, subnetworkElems...)
	}

	return allComputeElems, nil
}

func (gcpCrawler *gcpCrawler) GetVMInstances(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allVMInstanceElems := []*bloopi_agent.Element{}

	instances, errInstances := client.Instances.AggregatedList(gcpCrawler.ConfiguredProjectID).Do()
	if errInstances != nil {
		return allVMInstanceElems, fmt.Errorf("could not retrieve the instances because %v", errInstances)
	}

	for scopedZone, list := range instances.Items {
		for _, instance := range list.Instances {
			zone := getZoneFromScopedZone(scopedZone)
			zoneInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, "", gcpModel.TypeZone, zone)
			instanceInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeVMInstance, instance.Name)

			utils.AddRelationship(&allVMInstanceElems, zoneInternalName, instanceInternalID, bloopi_agent.ParentChildTypeRelation, crawlTime)

			instanceElem, errInstanceElem := utils.CreateElement(instance, instance.Name, instanceInternalID, gcpModel.TypeVMInstance, getComputeStatus(instance.Status), "", crawlTime)
			if errInstanceElem == nil {
				allVMInstanceElems = append(allVMInstanceElems, instanceElem)
			}

			for _, disk := range instance.Disks {
				split := strings.Split(disk.Source, "/")
				diskInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeDisk, split[len(split)-1])
				fmt.Println("vm-disk: ", diskInternalID, instanceInternalID)
				diskRel, errDiskRel := utils.CreateRelationship(instanceInternalID, diskInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errDiskRel == nil {
					allVMInstanceElems = append(allVMInstanceElems, diskRel)
				}
			}
		}
	}

	return allVMInstanceElems, nil
}

func (gcpCrawler *gcpCrawler) getNodeGroups(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allNodeGroups := []*bloopi_agent.Element{}

	nodeGroups, errNodeGroups := client.NodeGroups.AggregatedList(gcpCrawler.ConfiguredProjectID).Do()
	if errNodeGroups != nil {
		return allNodeGroups, fmt.Errorf("could not get all node groups because %s", errNodeGroups)
	}

	for zone, list := range nodeGroups.Items {
		for _, nodeGroup := range list.NodeGroups {
			zoneInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, "", gcpModel.TypeZone, zone)
			nodeGroupInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeNodeGroup, nodeGroup.Name)
			nodeGroupElem, errNodeGroupElem := utils.CreateElement(nodeGroup, nodeGroup.Name, nodeGroupInternalName, gcpModel.TypeNodeGroup, getComputeStatus(nodeGroup.Status), "", crawlTime)
			if errNodeGroupElem == nil {
				allNodeGroups = append(allNodeGroups, nodeGroupElem)
			}

			utils.AddRelationship(&allNodeGroups, zoneInternalName, nodeGroupInternalName, bloopi_agent.ParentChildTypeRelation, crawlTime)

			nodeGroupNodes, errNodeGroupNodes := client.NodeGroups.ListNodes(gcpCrawler.ConfiguredProjectID, zone, nodeGroup.Name).Do()
			if errNodeGroupNodes != nil {
				continue
			}

			for _, nodeGroupNode := range nodeGroupNodes.Items {
				nodeGroupNodeInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeNodeGroup, nodeGroupNode.Name)
				fmt.Println(nodeGroupNode.Name, nodeGroupNodeInternalName)
			}
		}
	}

	return allNodeGroups, nil
}

func (gcpCrawler *gcpCrawler) getInstanceGroups(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allInstanceGroupElems := []*bloopi_agent.Element{}

	instanceGroups, errInstanceGroups := client.InstanceGroups.AggregatedList(gcpCrawler.ConfiguredProjectID).Do()
	if errInstanceGroups != nil {
		return allInstanceGroupElems, errInstanceGroups
	}

	for scopedZone, list := range instanceGroups.Items {
		for _, instanceGroup := range list.InstanceGroups {
			zone := getZoneFromScopedZone(scopedZone)
			zoneInternalName := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, "", gcpModel.TypeZone, zone)
			instanceGroupInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeInstanceGroup, instanceGroup.Name)

			instanceGroupElem, errInstanceGroupElem := utils.CreateElement(instanceGroup, instanceGroup.Name, instanceGroupInternalID, gcpModel.TypeInstanceGroup, bloopi_agent.StatusNoStatus, "", crawlTime)
			if errInstanceGroupElem == nil {
				allInstanceGroupElems = append(allInstanceGroupElems, instanceGroupElem)
			}

			utils.AddRelationship(&allInstanceGroupElems, zoneInternalName, instanceGroupInternalID, bloopi_agent.ParentChildTypeRelation, crawlTime)

			instanceGroupInstanceList, errInstanceGroupInstance := client.InstanceGroups.ListInstances(gcpCrawler.ConfiguredProjectID, zone, instanceGroup.Name, &compute.InstanceGroupsListInstancesRequest{}).Do()
			if errInstanceGroupInstance != nil {
				continue
			}

			for _, instanceGroupInstance := range instanceGroupInstanceList.Items {
				split := strings.Split(instanceGroupInstance.Instance, "/")
				instanceInternalID := cloudutils.CreateGCPInternalName(gcpCrawler.dataSource.DataSourceID, zone, gcpModel.TypeVMInstance, split[len(split)-1])

				rel, errRel := utils.CreateRelationship(instanceGroupInternalID, instanceInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allInstanceGroupElems = append(allInstanceGroupElems, rel)
				}
			}
		}
	}

	return allInstanceGroupElems, nil
}

func (gcp *gcpCrawler) getDisks(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allDisks := []*bloopi_agent.Element{}

	disksAggregatedList, errDisksAggList := client.Disks.AggregatedList(gcp.ConfiguredProjectID).Do()
	if errDisksAggList != nil {
		return allDisks, errDisksAggList
	}

	for scopedZone, diskList := range disksAggregatedList.Items {
		zone := getZoneFromScopedZone(scopedZone)
		zoneInternalName := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeZone, zone)

		for _, disk := range diskList.Disks {
			diskInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, zone, gcpModel.TypeDisk, disk.Name)
			diskElem, errDiskElem := utils.CreateElement(disk, disk.Name, diskInternalID, gcpModel.TypeDisk, getComputeStatus(disk.Status), "", crawlTime)
			if errDiskElem == nil {
				allDisks = append(allDisks, diskElem)
			}

			utils.AddRelationship(&allDisks, zoneInternalName, diskInternalID, bloopi_agent.ParentChildTypeRelation, crawlTime)
		}
	}

	return allDisks, nil
}

func (gcp *gcpCrawler) getNetworks(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allNetworkElems := []*bloopi_agent.Element{}

	networks, errNetworks := client.Networks.List(gcp.ConfiguredProjectID).Do()
	if errNetworks != nil {
		return allNetworkElems, errNetworks
	}

	for _, network := range networks.Items {
		networkInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeNetwork, network.Name)

		networkElem, errNetworkElem := utils.CreateElement(network, network.Name, networkInternalID, gcpModel.TypeNetwork, bloopi_agent.StatusNoStatus, "", crawlTime)
		if errNetworkElem == nil {
			allNetworkElems = append(allNetworkElems, networkElem)
		}

		for _, subNet := range network.Subnetworks {
			split := strings.Split(subNet, "/")
			subnetInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, split[len(split)-3], gcpModel.TypeSubnetwork, split[len(split)-1])

			rel, errRel := utils.CreateRelationship(networkInternalID, subnetInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				allNetworkElems = append(allNetworkElems, rel)
			}
		}
	}

	return allNetworkElems, nil
}

func (gcp *gcpCrawler) getSubNetworks(client *compute.Service, crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allSubnets := []*bloopi_agent.Element{}

	subnetworks, errSubnetworks := client.Subnetworks.AggregatedList(gcp.ConfiguredProjectID).Do()
	if errSubnetworks != nil {
		return allSubnets, errSubnetworks
	}

	for scopedZone, list := range subnetworks.Items {
		zone := getZoneFromScopedZone(scopedZone)
		zoneInternalName := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeZone, zone)

		for _, subnet := range list.Subnetworks {
			subnetInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, zone, gcpModel.TypeSubnetwork, subnet.Name)
			subnetElem, errSubnetElem := utils.CreateElement(subnet, subnet.Name, subnetInternalID, gcpModel.TypeSubnetwork, getComputeStatus(subnet.State), "", crawlTime)
			if errSubnetElem == nil {
				allSubnets = append(allSubnets, subnetElem)
			}

			utils.AddRelationship(&allSubnets, zoneInternalName, subnetInternalID, bloopi_agent.ParentChildTypeRelation, crawlTime)
		}
	}

	return allSubnets, nil
}

func (gcp *gcpCrawler) getGKEClusters(crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allGKEClusterElems := []*bloopi_agent.Element{}

	client, errClient := createContainerClient(gcp.clientOpts)
	if errClient != nil {
		return nil, errClient
	}

	clusters, errClusters := client.Projects.Locations.Clusters.List(fmt.Sprintf("projects/%s/locations/-", gcp.ConfiguredProjectID)).Do()
	if errClusters != nil {
		return nil, errClusters
	}

	for _, cluster := range clusters.Clusters {
		zoneInternalName := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeZone, cluster.Zone)
		clusterInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, cluster.Location, gcpModel.TypeGKE, cluster.Name)
		clusterElem, errClusterElem := utils.CreateElement(cluster, cluster.Name, clusterInternalID, gcpModel.TypeGKE, getComputeStatus(cluster.Status), "", crawlTime)
		if errClusterElem != nil {
			continue
		}

		utils.AddRelationship(&allGKEClusterElems, zoneInternalName, clusterInternalID, bloopi_agent.ParentChildTypeRelation, crawlTime)

		allGKEClusterElems = append(allGKEClusterElems, clusterElem)

		networkRel, errNetworkRel := utils.CreateRelationship(cluster.Network, clusterInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errNetworkRel == nil {
			allGKEClusterElems = append(allGKEClusterElems, networkRel)
		}

		subnetRel, errSubnetRel := utils.CreateRelationship(cluster.Subnetwork, clusterInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errSubnetRel == nil {
			allGKEClusterElems = append(allGKEClusterElems, subnetRel)
		}

		for _, nodePool := range cluster.NodePools {
			nodePoolInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeNodePool, nodePool.Name)
			nodePoolElem, errNodePoolElem := utils.CreateElement(nodePool, nodePool.Name, nodePoolInternalID, gcpModel.TypeNodePool, getComputeStatus(nodePool.Status), "", crawlTime)
			if errNodePoolElem != nil {
				continue
			}

			allGKEClusterElems = append(allGKEClusterElems, nodePoolElem)

			for _, instanceGroupUrl := range nodePool.InstanceGroupUrls {
				split := strings.Split(instanceGroupUrl, "/")

				instanceGroupInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, split[len(split)-3], gcpModel.TypeInstanceGroup, split[len(split)-1])
				rel, errRel := utils.CreateRelationship(nodePoolInternalID, instanceGroupInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allGKEClusterElems = append(allGKEClusterElems, rel)
				}
			}

			relNetwork, errRelNetwork := utils.CreateRelationship(cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeNetwork, cluster.Network), nodePoolInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRelNetwork == nil {
				allGKEClusterElems = append(allGKEClusterElems, relNetwork)
			}

			relSubnet, errRelSubnet := utils.CreateRelationship(cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, cluster.Zone, gcpModel.TypeSubnetwork, cluster.Subnetwork), nodePoolInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRelSubnet == nil {
				allGKEClusterElems = append(allGKEClusterElems, relSubnet)
			}
		}
	}

	return allGKEClusterElems, nil
}

func (gcp *gcpCrawler) getSqlInstances(crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allCrawledSqlInstances := []*bloopi_agent.Element{}

	client, errClient := sqladmin.NewService(context.Background(), gcp.clientOpts...)
	if errClient != nil {
		return nil, errClient
	}

	sqlInstancesList, errSqlInstancesList := client.Instances.List(gcp.ConfiguredProjectID).Do()
	if errSqlInstancesList != nil {
		return nil, errSqlInstancesList
	}

	for _, sqlInstance := range sqlInstancesList.Items {
		sqlInternalName := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, sqlInstance.GceZone, gcpModel.TypeCloudSQL, sqlInstance.Name)
		elem, errElem := utils.CreateElement(sqlInstance, sqlInstance.Name, sqlInternalName, gcpModel.TypeCloudSQL, getComputeStatus(sqlInstance.State), "", crawlTime)
		if errElem == nil {
			allCrawledSqlInstances = append(allCrawledSqlInstances, elem)
		}

		split := strings.Split(sqlInstance.Settings.IpConfiguration.PrivateNetwork, "/")
		vpcInternalID := cloudutils.CreateGCPInternalName(gcp.dataSource.DataSourceID, "", gcpModel.TypeNetwork, split[4])
		rel, errRel := utils.CreateRelationship(vpcInternalID, sqlInternalName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
		if errRel == nil {
			allCrawledSqlInstances = append(allCrawledSqlInstances, rel)
		}
	}

	return allCrawledSqlInstances, nil
}
