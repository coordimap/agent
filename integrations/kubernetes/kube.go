package kubernetes

import (
	cloudutils "cleye/internal/cloud/utils"
	"cleye/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	gcpModel "dev.azure.com/bloopi/bloopi/_git/shared_models.git/gcp"
	kube_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
	"github.com/rs/zerolog/log"
)

func MakeKubernetesCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	clientInitialzed := false

	// Create initial kubernetesCrawler object
	crawler := &kubernetesCrawler{
		kubeClient:        nil,
		crawlInterval:     defaultCrawlTime,
		dataSource:        *dataSource,
		outputChannel:     outChannel,
		istioConfigured:   false,
		istioCrawler:      prometheusCrawler{},
		clusterName:       "",
		retinaCrawler:     nil,
		internalNodeNames: map[string]string{},
	}

	promQueryTime := ""

	// Assign values from the config
	for _, dsConfig := range dataSource.Config.ValuePairs {
		value, errLoadValue := utils.LoadValueFromEnvConfig(dsConfig.Value)
		if errLoadValue != nil {
			log.Info().Msgf("Error loading value of db_pass for value: %s. The error returned was: %s", dsConfig.Value, errLoadValue.Error())
			return crawler, errLoadValue
		}

		switch dsConfig.Key {

		case kubeConfigInCluster:
			if strings.Compare(value, "true") != 0 || clientInitialzed {
				continue
			}

			clientSet, errClientSet := connectoToK8sInCluster()
			if errClientSet != nil {
				return crawler, errClientSet
			}
			crawler.kubeClient = clientSet

			clientInitialzed = true

		case kubeConfigConfigFile:
			if clientInitialzed {
				continue
			}

			clientSet, errClientSet := connectToK8sFromConfigFile(value)
			if errClientSet != nil {
				return crawler, errClientSet
			}

			crawler.kubeClient = clientSet

			clientInitialzed = true

		case kubeConfigCloudDataSourceID:
			if dsConfig.Value == "" {
				continue
			}
			crawler.cloudDataSourceID = dsConfig.Value

		case kubeConfigIstioPrometheusHost:
			istioCrawler, err := makePrometheusCrawler(value)
			if err != nil {
				return crawler, err
			}

			crawler.istioCrawler = istioCrawler
			crawler.istioConfigured = true

		case kubeConfigRetinaPrometheusHost:
			retinaCrawler, err := makePrometheusCrawler(value)
			if err != nil {
				return crawler, err
			}

			crawler.retinaCrawler = &retinaCrawler

		case kubeConfigClusterName:
			crawler.clusterName = value

		case kubeConfigCrawlInterval:
			amountStr := string(dsConfig.Value[:len(dsConfig.Value)-1])
			durationStr := string(dsConfig.Value[len(dsConfig.Value)-1])
			promQueryTime = value

			amount, errConv := strconv.ParseInt(amountStr, 10, 32)
			if errConv != nil {
				return crawler, errConv
			}

			switch durationStr {
			case "s":
				crawler.crawlInterval = time.Duration(amount) * time.Second

			case "m":
				crawler.crawlInterval = time.Duration(amount) * time.Minute

			default:
				crawler.crawlInterval = defaultCrawlTime
			}
		}
	}

	crawler.istioCrawler.promQueryTime = promQueryTime

	return crawler, nil
}

func (kubeCrawler *kubernetesCrawler) Crawl() {
	crawlTicker := time.NewTicker(kubeCrawler.crawlInterval)

	log.Info().Msgf("Starting ticker for: %s", kubeCrawler.dataSource.DataSourceID)
	for range crawlTicker.C {
		_, errCrawl := kubeCrawler.crawl()
		log.Info().Msgf("Crawling Kubernetes cluster for %s", kubeCrawler.dataSource.DataSourceID)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msg(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
	}
}

func (kubeCrawler *kubernetesCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	crawlTime := time.Now().UTC()
	globalCrawledElements := []*bloopi_agent.Element{}
	createdElementsFromLabels := []string{}

	nodes, errNodes := kubeCrawler.getNodes()
	if errNodes != nil {
		log.Warn().Msgf("Could not get the kubernetes nodes of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errNodes.Error())
	}

	for _, node := range nodes {
		cloudName, errCloudName := getNodeCloud(node.Labels, node.Annotations, node.Status.Addresses)
		if errCloudName == nil && kubeCrawler.cloudDataSourceID != "" {
			nodeInternalName := ""

			switch cloudName {
			case "aws":
				nodeInternalName = cloudutils.CreateAWSInternalID(kubeCrawler.cloudDataSourceID, node.Name)

			case "gcp":
				nodeInternalName = cloudutils.CreateGCPInternalName(kubeCrawler.cloudDataSourceID, node.Labels["topology.kubernetes.io/region"], gcpModel.TypeVMInstance, node.Name)
			}

			kubeCrawler.internalNodeNames[node.Name] = nodeInternalName
			continue
		}

		nodeElement, errNodeElement := utils.CreateElement(node, node.Name, node.Name, kube_model.TypeNode, bloopi_agent.StatusNoStatus, "", crawlTime)
		if errNodeElement != nil {
			continue
		}

		globalCrawledElements = append(globalCrawledElements, nodeElement)
	}

	pvs, errPvs := kubeCrawler.listPersistentVolumes()
	if errPvs != nil {
		log.Warn().Msgf("Could not get the kubernetes persistenvolumes of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errPvs.Error())
	} else {
		for _, pv := range pvs {
			pvInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, "", kube_model.TypePV, pv.Name)
			nodeElement, errNodeElement := utils.CreateElement(pv, pv.Name, pvInternalID, kube_model.TypePV, bloopi_agent.StatusNoStatus, "", crawlTime)
			if errNodeElement != nil {
				continue
			}

			elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(pvInternalID, "", pv.Labels, createdElementsFromLabels, crawlTime)
			createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
			globalCrawledElements = append(globalCrawledElements, elems...)

			rel, errRel := utils.CreateRelationship(pvInternalID, cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, "", kube_model.TypeStorageClass, pv.Spec.StorageClassName), bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
			if errRel == nil {
				globalCrawledElements = append(globalCrawledElements, rel)
			}

			globalCrawledElements = append(globalCrawledElements, nodeElement)
		}
	}

	storageClasses, errStorageClasses := kubeCrawler.listStorageClasses()
	if errStorageClasses != nil {
		log.Warn().Msgf("Could not get the kubernetes storageclasses of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errStorageClasses.Error())
	} else {
		for _, storageClass := range storageClasses {
			storageClassInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, "", kube_model.TypeStorageClass, storageClass.Name)
			nodeElement, errNodeElement := utils.CreateElement(storageClass, storageClass.Name, storageClassInternalID, kube_model.TypeStorageClass, bloopi_agent.StatusNoStatus, "", crawlTime)
			if errNodeElement != nil {
				continue
			}

			elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(storageClassInternalID, "", storageClass.Labels, createdElementsFromLabels, crawlTime)
			createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
			globalCrawledElements = append(globalCrawledElements, elems...)

			globalCrawledElements = append(globalCrawledElements, nodeElement)
		}
	}

	kubeNamespaces, errNamespaces := kubeCrawler.listNamespaces()
	if errNamespaces != nil {
		log.Warn().Msgf("Could not get the kubernetes namespaces of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errNamespaces.Error())
	}

	for _, namespace := range kubeNamespaces {
		namespaceInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeNamespace, "")
		allCrawledElements := []*bloopi_agent.Element{}
		allCrawledElements = append(allCrawledElements, globalCrawledElements...)

		nodeElement, errNodeElement := utils.CreateElement(namespace, namespace.Name, namespaceInternalID, kube_model.TypeNamespace, bloopi_agent.StatusNoStatus, "", crawlTime)
		if errNodeElement != nil {
			continue
		}

		elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(namespaceInternalID, namespace.Name, namespace.Labels, createdElementsFromLabels, crawlTime)
		createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
		allCrawledElements = append(allCrawledElements, elems...)

		allCrawledElements = append(allCrawledElements, nodeElement)

		// add the relevant namespace - storageClass relationship
		for _, storageClass := range storageClasses {
			if storageClass.Namespace == namespace.Name {
				storageClassInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeStorageClass, storageClass.Name)
				rel, errRel := utils.CreateRelationship(namespaceInternalID, storageClassInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				break
			}
		}

		// add the relevant namespace - persistenVolume relationship
		for _, pv := range pvs {
			if pv.Namespace == namespace.Name {
				pvInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypePV, pv.Name)
				rel, errRel := utils.CreateRelationship(namespaceInternalID, pvInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				break
			}
		}

		// get the deployments
		deployments, errDeployments := kubeCrawler.listDeplyments(namespace.Name)
		if errDeployments != nil {
			log.Warn().Msgf("Could not get the kubernetes deployments of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errDeployments.Error())
		} else {
			for _, deployment := range deployments {
				deploymentStatus := getDeploymentStatus(deployment.Status.Conditions)
				deploymentInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeDeployment, deployment.Name)
				nodeElement, errNodeElement := utils.CreateElement(deployment, deployment.Name, deploymentInternalID, kube_model.TypeDeployment, deploymentStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(deploymentInternalID, namespace.Name, deployment.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				// create namespace - deployment relationship
				rel, errRel := utils.CreateRelationship(namespaceInternalID, deploymentInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				// get deployment pods
				deploymentPods, errDeploymentPods := kubeCrawler.listDeplymentPods(&deployment, namespace.Name)
				if errDeploymentPods != nil {
					continue
				}

				for _, deploymentPodRelationship := range deploymentPods {
					nameAndID := fmt.Sprintf("%s.%s", deploymentPodRelationship.SourceID, deploymentPodRelationship.DestinationID)
					servicePodRelationshipElem, errSerservicePodRelationshipElem := utils.CreateElement(
						deploymentPodRelationship,
						nameAndID,
						nameAndID,
						bloopi_agent.RelationshipType,
						bloopi_agent.StatusNoStatus, "",
						crawlTime,
					)

					if errSerservicePodRelationshipElem != nil {
						continue
					}

					allCrawledElements = append(allCrawledElements, servicePodRelationshipElem)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// get the services
		services, errServices := kubeCrawler.listServices(namespace.Name)
		if errServices != nil {
			log.Warn().Msgf("Could not get the kubernetes services of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errServices.Error())
		} else {
			for _, service := range services {
				serviceInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeService, service.Name)
				nodeElement, errNodeElement := utils.CreateElement(service, service.Name, serviceInternalID, kube_model.TypeService, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(serviceInternalID, namespace.Name, service.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				// create namespace - service relationship
				rel, errRel := utils.CreateRelationship(namespaceInternalID, serviceInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				// get the service pods
				servicePods, errServicePods := kubeCrawler.listServicePods(&service, namespace.Name)
				if errServicePods != nil {
					continue
				}

				for _, servicePodRelationship := range servicePods {
					nameAndID := fmt.Sprintf("%s.%s", servicePodRelationship.SourceID, servicePodRelationship.DestinationID)
					servicePodRelationshipElem, errSerservicePodRelationshipElem := utils.CreateElement(
						servicePodRelationship,
						nameAndID,
						nameAndID,
						bloopi_agent.RelationshipType,
						bloopi_agent.StatusNoStatus, "",
						crawlTime,
					)

					if errSerservicePodRelationshipElem != nil {
						continue
					}

					allCrawledElements = append(allCrawledElements, servicePodRelationshipElem)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// get pods
		pods, errPods := kubeCrawler.listPods(namespace.Name)
		if errPods != nil {
			log.Warn().Msgf("Could not get the kubernetes pods of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errPods.Error())
		} else {
			for _, pod := range pods {
				podStatus := getPodStatus(pod.Status.Phase)
				podInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypePod, pod.Name)
				nodeElement, errNodeElement := utils.CreateElement(pod, pod.Name, podInternalID, kube_model.TypePod, podStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(podInternalID, namespace.Name, pod.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, podInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				if pod.Spec.NodeName != "" {
					if internalName, ok := kubeCrawler.internalNodeNames[pod.Spec.NodeName]; ok {
						rel, errRel := utils.CreateRelationship(internalName, podInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}
				}

				for _, podContainer := range pod.Spec.Containers {
					for _, containerEnv := range podContainer.Env {
						if containerEnv.ValueFrom != nil {
							if containerEnv.ValueFrom.ConfigMapKeyRef != nil {
								configMapInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeConfigMap, containerEnv.ValueFrom.ConfigMapKeyRef.Name)
								rel, errRel := utils.CreateRelationship(podInternalID, configMapInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
								if errRel == nil {
									allCrawledElements = append(allCrawledElements, rel)
								}
							}

							if containerEnv.ValueFrom.SecretKeyRef != nil {
								podSecretInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeSecret, containerEnv.ValueFrom.SecretKeyRef.Name)
								rel, errRel := utils.CreateRelationship(podInternalID, podSecretInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
								if errRel == nil {
									allCrawledElements = append(allCrawledElements, rel)
								}
							}
						}
					}
				}

				for _, podVolume := range pod.Spec.Volumes {
					podVolumeInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypePV, podVolume.Name)
					rel, errRel := utils.CreateRelationship(podInternalID, podVolumeInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
					if errRel == nil {
						allCrawledElements = append(allCrawledElements, rel)
					}

					if podVolume.ConfigMap != nil {
						podConfigMapInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeConfigMap, podVolume.ConfigMap.Name)
						rel, errRel := utils.CreateRelationship(podInternalID, podConfigMapInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}

					if podVolume.PersistentVolumeClaim != nil {
						podPersisternVolumeClaim := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypePVC, podVolume.PersistentVolumeClaim.ClaimName)
						rel, errRel := utils.CreateRelationship(podInternalID, podPersisternVolumeClaim, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}

					if podVolume.Secret != nil {
						podSecretInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeSecret, podVolume.Secret.SecretName)
						rel, errRel := utils.CreateRelationship(podInternalID, podSecretInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list secrets
		secrets, errSecrets := kubeCrawler.listSecrets(namespace.Name)
		if errSecrets != nil {
			log.Warn().Msgf("Could not get the kubernetes secrets of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errSecrets.Error())
		} else {
			for _, secret := range secrets {
				secretInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeSecret, secret.Name)
				nodeElement, errNodeElement := utils.CreateElement(secret, secret.Name, secretInternalID, kube_model.TypeSecret, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(secretInternalID, namespace.Name, secret.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, secretInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list endpoints
		endpoints, errEndpoints := kubeCrawler.listEndpoints(namespace.Name)
		if errEndpoints != nil {
			log.Warn().Msgf("Could not get the kubernetes endpoints of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errEndpoints.Error())
		} else {
			for _, endpoint := range endpoints {
				endpointInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeEndpoint, endpoint.Name)
				nodeElement, errNodeElement := utils.CreateElement(endpoint, endpoint.Name, endpointInternalID, kube_model.TypeEndpoint, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(endpointInternalID, namespace.Name, endpoint.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, endpointInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				// TODO: add relationship to the target
				// endpoint.Subsets[0].Addresses[0].TargetRef

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list jobs
		jobs, errJobs := kubeCrawler.listJobs(namespace.Name)
		if errJobs != nil {
			log.Warn().Msgf("Could not get the kubernetes jobs of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errJobs.Error())
		} else {
			for _, job := range jobs {
				jobInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeJob, job.Name)
				nodeElement, errNodeElement := utils.CreateElement(job, job.Name, jobInternalID, kube_model.TypeJob, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(jobInternalID, namespace.Name, job.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, jobInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				if job.Spec.Template.Spec.NodeName != "" {
					rel, errRel = utils.CreateRelationship(job.Spec.Template.Spec.NodeName, jobInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
					if errRel == nil {
						allCrawledElements = append(allCrawledElements, rel)
					}
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list cronjobs
		cronJobs, errCronJobs := kubeCrawler.listCronJobs(namespace.Name)
		if errEndpoints != nil {
			log.Warn().Msgf("Could not get the kubernetes cronjobs of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errCronJobs.Error())
		} else {
			for _, cronJob := range cronJobs {
				cronJobInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeCronJob, cronJob.Name)
				nodeElement, errNodeElement := utils.CreateElement(cronJob, cronJob.Name, cronJobInternalID, kube_model.TypeCronJob, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(cronJobInternalID, namespace.Name, cronJob.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, cronJobInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				if cronJob.Spec.JobTemplate.Spec.Template.Spec.NodeName != "" {
					rel, errRel = utils.CreateRelationship(cronJob.Spec.JobTemplate.Spec.Template.Spec.NodeName, cronJobInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
					if errRel == nil {
						allCrawledElements = append(allCrawledElements, rel)
					}
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list configmaps
		configMaps, errConfigMaps := kubeCrawler.listConfigMaps(namespace.Name)
		if errConfigMaps != nil {
			log.Warn().Msgf("Could not get the kubernetes configmaps of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errConfigMaps.Error())
		} else {
			for _, configMap := range configMaps {
				configMapInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeConfigMap, configMap.Name)
				nodeElement, errNodeElement := utils.CreateElement(configMap, configMap.Name, configMapInternalID, kube_model.TypeConfigMap, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(configMapInternalID, namespace.Name, configMap.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, configMapInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list statefulsets
		statefulSets, errStatefulSets := kubeCrawler.listStatefulSets(namespace.Name)
		if errStatefulSets != nil {
			log.Warn().Msgf("Could not get the kubernetes statefulsets of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errStatefulSets.Error())
		} else {
			for _, statefulSet := range statefulSets {
				statefulSetInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeStatefulSet, statefulSet.Name)
				nodeElement, errNodeElement := utils.CreateElement(statefulSet, statefulSet.Name, statefulSetInternalID, kube_model.TypeStatefulSet, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(statefulSetInternalID, namespace.Name, statefulSet.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, statefulSetInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				if statefulSet.Spec.Template.Spec.NodeName != "" {
					rel, errRel := utils.CreateRelationship(statefulSet.Spec.Template.Spec.NodeName, statefulSetInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
					if errRel == nil {
						allCrawledElements = append(allCrawledElements, rel)
					}
				}

				// add volume details
				for _, pvc := range statefulSet.Spec.VolumeClaimTemplates {
					volInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypePV, pvc.Spec.VolumeName)

					rel, errRel := utils.CreateRelationship(statefulSetInternalID, volInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
					if errRel == nil {
						allCrawledElements = append(allCrawledElements, rel)
					}
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list daemonsets
		daemonSets, errDaemonSets := kubeCrawler.listDaemonSets(namespace.Name)
		if errDaemonSets != nil {
			log.Warn().Msgf("Could not get the kubernetes daemonsets of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errDaemonSets.Error())
		} else {
			for _, daemonSet := range daemonSets {
				daemonSetInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeDaemonSet, daemonSet.Name)
				nodeElement, errNodeElement := utils.CreateElement(daemonSet, daemonSet.Name, daemonSetInternalID, kube_model.TypeDaemonSet, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(daemonSetInternalID, namespace.Name, daemonSet.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, daemonSetInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list pvcs
		pvcs, errPVCs := kubeCrawler.listPersistentVolumeClaims(namespace.Name)
		if errPVCs != nil {
			log.Warn().Msgf("Could not get the kubernetes persistenvolumeclaims of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errPVCs.Error())
		} else {
			for _, pvc := range pvcs {
				pvcInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypePVC, pvc.Name)
				nodeElement, errNodeElement := utils.CreateElement(pvc, pvc.Name, pvcInternalID, kube_model.TypePVC, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(pvcInternalID, namespace.Name, pvc.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, pvcInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list ingressesExtensionsBeta1
		ingressesExtensionsBeta1, errIngressesExtensionsBeta1 := kubeCrawler.listIngressesExtensionsBeta1(namespace.Name)
		if errIngressesExtensionsBeta1 != nil {
			log.Warn().Msgf("Could not get the kubernetes ingresses extensions beta1 of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errIngressesExtensionsBeta1.Error())
		} else {
			for _, ingress := range ingressesExtensionsBeta1 {
				ingressInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeIngressExtensionBeta1, ingress.Name)
				nodeElement, errNodeElement := utils.CreateElement(ingress, ingress.Name, ingressInternalID, kube_model.TypeIngressExtensionBeta1, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				for _, rules := range ingress.Spec.Rules {
					for _, path := range rules.HTTP.Paths {
						internalServiceName := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, ingress.Namespace, kube_model.TypeService, path.Backend.ServiceName)
						rel, errRel := utils.CreateRelationship(ingressInternalID, internalServiceName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(ingressInternalID, namespace.Name, ingress.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, ingressInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		ingressesNetworkingV1, errIngressesNetworkingV1 := kubeCrawler.listIngressesNetworkingV1(namespace.Name)
		if errIngressesNetworkingV1 != nil {
			log.Warn().Msgf("Could not get the kubernetes ingresses extensions beta1 of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errIngressesExtensionsBeta1.Error())
		} else {
			for _, ingress := range ingressesNetworkingV1 {
				ingressInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeIngressNetworkingV1, ingress.Name)
				nodeElement, errNodeElement := utils.CreateElement(ingress, ingress.Name, ingressInternalID, kube_model.TypeIngressNetworkingV1, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				for _, rules := range ingress.Spec.Rules {
					for _, path := range rules.HTTP.Paths {
						internalServiceName := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, ingress.Namespace, kube_model.TypeService, path.Backend.Service.Name)
						rel, errRel := utils.CreateRelationship(ingressInternalID, internalServiceName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(ingressInternalID, namespace.Name, ingress.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, ingressInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		ingressesNetworkingV1Beta1, errIngressesNetworkingV1Beta1 := kubeCrawler.listIngressesNetworkingV1Beta1(namespace.Name)
		if errIngressesNetworkingV1Beta1 != nil {
			log.Warn().Msgf("Could not get the kubernetes ingresses extensions beta1 of data source name: %s because %s", kubeCrawler.dataSource.DataSourceID, errIngressesExtensionsBeta1.Error())
		} else {
			for _, ingress := range ingressesNetworkingV1Beta1 {
				ingressInternalID := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, namespace.Name, kube_model.TypeIngressExtensionBeta1, ingress.Name)
				nodeElement, errNodeElement := utils.CreateElement(ingress, ingress.Name, ingressInternalID, kube_model.TypeIngressNetworkingV1Beta1, bloopi_agent.StatusNoStatus, "", crawlTime)
				if errNodeElement != nil {
					continue
				}

				for _, rules := range ingress.Spec.Rules {
					for _, path := range rules.HTTP.Paths {
						internalServiceName := cloudutils.CreateKubeInternalName(kubeCrawler.dataSource.DataSourceID, ingress.Namespace, kube_model.TypeService, path.Backend.ServiceName)
						rel, errRel := utils.CreateRelationship(ingressInternalID, internalServiceName, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
						if errRel == nil {
							allCrawledElements = append(allCrawledElements, rel)
						}
					}
				}

				elems, createdElems := kubeCrawler.getLabelElementsAndRelationships(ingressInternalID, namespace.Name, ingress.Labels, createdElementsFromLabels, crawlTime)
				createdElementsFromLabels = append(createdElementsFromLabels, createdElems...)
				allCrawledElements = append(allCrawledElements, elems...)

				rel, errRel := utils.CreateRelationship(namespaceInternalID, ingressInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		if kubeCrawler.retinaCrawler != nil {
			if retinaElems, errRetina := kubeCrawler.getRetinaFlowsRelationships(crawlTime); errRetina == nil {
				allCrawledElements = append(allCrawledElements, retinaElems...)
			}
		}

		crawledData := bloopi_agent.CrawledData{
			Data: allCrawledElements,
		}

		log.Info().Msgf("Crawled %d Kubernetes elements for connection %s and namespace %s", len(allCrawledElements), kubeCrawler.dataSource.DataSourceID, namespace.Name)

		kubeCrawler.outputChannel <- &bloopi_agent.CloudCrawlData{
			Timestamp:       crawlTime,
			DataSource:      kubeCrawler.dataSource,
			CrawledData:     crawledData,
			CrawlInternalID: fmt.Sprintf("%s.%s", namespace.Name, kube_model.TypeNamespace),
		}
	}

	if !kubeCrawler.istioConfigured {
		return nil, nil
	}

	istioRelationships, errIstioRelationships := kubeCrawler.getIstioRelationships()
	if errIstioRelationships != nil {
		log.Info().Msgf("There was an error finding the istio relationships for kubernetes connection %s because %s", kubeCrawler.dataSource.DataSourceID, errIstioRelationships.Error())
		return nil, errIstioRelationships
	}

	istioElements := []*bloopi_agent.Element{}
	for _, istioRelaitonship := range istioRelationships {
		istioElem, errIstioElem := utils.CreateElement(
			istioRelaitonship,
			fmt.Sprintf("%s-%s", istioRelaitonship.SourceID, istioRelaitonship.DestinationID),
			fmt.Sprintf("%s-%s", istioRelaitonship.SourceID, istioRelaitonship.DestinationID),
			kube_model.FlowIstioRelationshipSkipinsert,
			bloopi_agent.StatusNoStatus, "",
			crawlTime,
		)
		if errIstioElem != nil {
			continue
		}

		istioElements = append(istioElements, istioElem)
	}

	crawledData := bloopi_agent.CrawledData{
		Data: istioElements,
	}

	kubeCrawler.outputChannel <- &bloopi_agent.CloudCrawlData{
		Timestamp:   crawlTime,
		DataSource:  kubeCrawler.dataSource,
		CrawledData: crawledData,
	}

	return nil, nil
}
