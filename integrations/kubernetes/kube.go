package kubernetes

import (
	"cleye/utils"
	"fmt"
	"strconv"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	kube_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
	"github.com/rs/zerolog/log"
)

func MakeKubernetesCrawler(dataSource *bloopi_agent.DataSource, outChannel chan *bloopi_agent.CloudCrawlData) (Crawler, error) {
	clientInitialzed := false

	// Create initial kubernetesCrawler object
	crawler := &kubernetesCrawler{
		kubeClient:      nil,
		crawlInterval:   defaultCrawlTime,
		dataSource:      *dataSource,
		outputChannel:   outChannel,
		istioConfigured: false,
		istioCrawler:    istioCrawler{},
		clusterName:     "",
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

		case kubeConfigIstioPrometheusHost:
			istioCrawler, err := makeIstioCrawler(value)
			if err != nil {
				return crawler, err
			}

			crawler.istioCrawler = istioCrawler
			crawler.istioConfigured = true

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

	log.Info().Msgf("Starting ticker for: %s", kubeCrawler.dataSource.Info.Name)
	for range crawlTicker.C {
		_, errCrawl := kubeCrawler.crawl()
		log.Info().Msgf("Crawling Kubernetes cluster for %s-%s", kubeCrawler.dataSource.Info.Type, kubeCrawler.dataSource.Info.Name)
		if errCrawl != nil {
			// do not ship any data
			log.Info().Msgf(errCrawl.Error())
			continue
		}
		// ship the crawledData to the backend
	}
}

func (kubeCrawler *kubernetesCrawler) crawl() (*bloopi_agent.CloudCrawlData, error) {
	prefix := kubeCrawler.clusterName
	crawlTime := time.Now().UTC()
	globalCrawledElements := []*bloopi_agent.Element{}

	nodes, errNodes := kubeCrawler.getNodes()
	if errNodes != nil {
		log.Warn().Msgf("Could not get the kubernetes nodes of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errNodes.Error())
	}

	for _, node := range nodes {
		nodeElement, errNodeElement := utils.CreateElement(node, node.Name, node.Name, kube_model.TypeNode, crawlTime)
		if errNodeElement != nil {
			continue
		}

		globalCrawledElements = append(globalCrawledElements, nodeElement)
	}

	pvs, errPvs := kubeCrawler.listPersistentVolumes()
	if errPvs != nil {
		log.Warn().Msgf("Could not get the kubernetes persistenvolumes of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errPvs.Error())
	} else {
		for _, pv := range pvs {
			nodeElement, errNodeElement := utils.CreateElement(pv, pv.Name, fmt.Sprintf("%s%s", prefix, pv.Name), kube_model.TypePV, crawlTime)
			if errNodeElement != nil {
				continue
			}

			globalCrawledElements = append(globalCrawledElements, nodeElement)
		}
	}

	storageClasses, errStorageClasses := kubeCrawler.listStorageClasses()
	if errStorageClasses != nil {
		log.Warn().Msgf("Could not get the kubernetes storageclasses of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errStorageClasses.Error())
	} else {
		for _, storageClass := range storageClasses {
			nodeElement, errNodeElement := utils.CreateElement(storageClass, storageClass.Name, fmt.Sprintf("%s%s", prefix, storageClass.Name), kube_model.TypeStorageClass, crawlTime)
			if errNodeElement != nil {
				continue
			}

			globalCrawledElements = append(globalCrawledElements, nodeElement)
		}
	}

	kubeNamespaces, errNamespaces := kubeCrawler.listNamespaces()
	if errNamespaces != nil {
		log.Warn().Msgf("Could not get the kubernetes namespaces of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errNamespaces.Error())
	}

	for _, namespace := range kubeNamespaces {
		namespacePrefix := prefix + "." + namespace.Name
		allCrawledElements := []*bloopi_agent.Element{}
		allCrawledElements = append(allCrawledElements, globalCrawledElements...)

		nodeElement, errNodeElement := utils.CreateElement(namespace, namespace.Name, namespacePrefix, kube_model.TypeNamespace, crawlTime)
		if errNodeElement != nil {
			continue
		}

		allCrawledElements = append(allCrawledElements, nodeElement)

		// get the deployments
		deployments, errDeployments := kubeCrawler.listDeplyments(namespace.Name)
		if errDeployments != nil {
			log.Warn().Msgf("Could not get the kubernetes deployments of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errDeployments.Error())
		} else {
			for _, deployment := range deployments {
				deploymentInternalID := namespacePrefix + "." + kube_model.TypeDeployment + "." + deployment.Name
				nodeElement, errNodeElement := utils.CreateElement(deployment, deployment.Name, deploymentInternalID, kube_model.TypeDeployment, crawlTime)
				if errNodeElement != nil {
					continue
				}

				// create namespace - deployment relationship
				rel, errRel := utils.CreateRelationship(namespacePrefix, deploymentInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				// get deployment pods
				deploymentPods, errDeploymentPods := kubeCrawler.listDeplymentPods(&deployment, namespacePrefix, namespace.Name)
				if errDeploymentPods != nil {
					continue
				}

				for _, deploymentPodRelationship := range deploymentPods {
					nameAndID := fmt.Sprintf("%s.%s", deploymentPodRelationship.SourceID, deploymentPodRelationship.DestinationID)
					servicePodRelationshipElem, errSerservicePodRelationshipElem := utils.CreateElement(
						deploymentPodRelationship,
						nameAndID,
						nameAndID,
						kube_model.RelationshipSkipinsert,
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
			log.Warn().Msgf("Could not get the kubernetes services of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errServices.Error())
		} else {
			for _, service := range services {
				serviceInternalID := namespacePrefix + "." + kube_model.TypeService + "." + service.Name
				nodeElement, errNodeElement := utils.CreateElement(service, service.Name, serviceInternalID, kube_model.TypeService, crawlTime)
				if errNodeElement != nil {
					continue
				}

				// create namespace - service relationship
				rel, errRel := utils.CreateRelationship(namespacePrefix, serviceInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				// get the service pods
				servicePods, errServicePods := kubeCrawler.listServicePods(&service, namespacePrefix, namespace.Name)
				if errServicePods != nil {
					continue
				}

				for _, servicePodRelationship := range servicePods {
					nameAndID := fmt.Sprintf("%s.%s", servicePodRelationship.SourceID, servicePodRelationship.DestinationID)
					servicePodRelationshipElem, errSerservicePodRelationshipElem := utils.CreateElement(
						servicePodRelationship,
						nameAndID,
						nameAndID,
						kube_model.RelationshipSkipinsert,
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
			log.Warn().Msgf("Could not get the kubernetes pods of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errPods.Error())
		} else {
			for _, pod := range pods {
				podInternalID := namespacePrefix + "." + kube_model.TypePod + "." + pod.Name
				nodeElement, errNodeElement := utils.CreateElement(pod, pod.Name, podInternalID, kube_model.TypePod, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, podInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list secrets
		secrets, errSecrets := kubeCrawler.listSecrets(namespace.Name)
		if errSecrets != nil {
			log.Warn().Msgf("Could not get the kubernetes secrets of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errSecrets.Error())
		} else {
			for _, secret := range secrets {
				secretInternalID := namespacePrefix + "." + kube_model.TypeSecret + "." + secret.Name
				nodeElement, errNodeElement := utils.CreateElement(secret, secret.Name, secretInternalID, kube_model.TypeSecret, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, secretInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list endpoints
		endpoints, errEndpoints := kubeCrawler.listEndpoints(namespace.Name)
		if errEndpoints != nil {
			log.Warn().Msgf("Could not get the kubernetes endpoints of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errEndpoints.Error())
		} else {
			for _, endpoint := range endpoints {
				endpointInternalID := namespacePrefix + "." + kube_model.TypeEndpoint + "." + endpoint.Name
				nodeElement, errNodeElement := utils.CreateElement(endpoint, endpoint.Name, endpointInternalID, kube_model.TypeEndpoint, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, endpointInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list jobs
		jobs, errJobs := kubeCrawler.listJobs(namespace.Name)
		if errJobs != nil {
			log.Warn().Msgf("Could not get the kubernetes jobs of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errJobs.Error())
		} else {
			for _, job := range jobs {
				jobInternalID := namespacePrefix + "." + kube_model.TypeJob + "." + job.Name
				nodeElement, errNodeElement := utils.CreateElement(job, job.Name, jobInternalID, kube_model.TypeJob, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, jobInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list cronjobs
		cronJobs, errCronJobs := kubeCrawler.listCronJobs(namespace.Name)
		if errEndpoints != nil {
			log.Warn().Msgf("Could not get the kubernetes cronjobs of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errCronJobs.Error())
		} else {
			for _, cronJob := range cronJobs {
				cronJobInternalID := namespacePrefix + "." + kube_model.TypeCronJob + "." + cronJob.Name
				nodeElement, errNodeElement := utils.CreateElement(cronJob, cronJob.Name, cronJobInternalID, kube_model.TypeCronJob, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, cronJobInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list configmaps
		configMaps, errConfigMaps := kubeCrawler.listConfigMaps(namespace.Name)
		if errConfigMaps != nil {
			log.Warn().Msgf("Could not get the kubernetes configmaps of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errConfigMaps.Error())
		} else {
			for _, configMap := range configMaps {
				configMapInternalID := namespacePrefix + "." + kube_model.TypeConfigMap + "." + configMap.Name
				nodeElement, errNodeElement := utils.CreateElement(configMap, configMap.Name, configMapInternalID, kube_model.TypeConfigMap, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, configMapInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list statefulsets
		statefulSets, errStatefulSets := kubeCrawler.listStatefulSets(namespace.Name)
		if errStatefulSets != nil {
			log.Warn().Msgf("Could not get the kubernetes statefulsets of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errStatefulSets.Error())
		} else {
			for _, statefulSet := range statefulSets {
				statefulSetInternalID := namespacePrefix + "." + kube_model.TypeStatefulSet + "." + statefulSet.Name
				nodeElement, errNodeElement := utils.CreateElement(statefulSet, statefulSet.Name, statefulSetInternalID, kube_model.TypeStatefulSet, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, statefulSetInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list daemonsets
		daemonSets, errDaemonSets := kubeCrawler.listDaemonSets(namespace.Name)
		if errDaemonSets != nil {
			log.Warn().Msgf("Could not get the kubernetes daemonsets of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errDaemonSets.Error())
		} else {
			for _, daemonSet := range daemonSets {
				daemonSetInternalID := namespacePrefix + "." + kube_model.TypeDaemonSet + "." + daemonSet.Name
				nodeElement, errNodeElement := utils.CreateElement(daemonSet, daemonSet.Name, daemonSetInternalID, kube_model.TypeDaemonSet, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, daemonSetInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list pvcs
		pvcs, errPVCs := kubeCrawler.listPersistentVolumeClaims(namespace.Name)
		if errPVCs != nil {
			log.Warn().Msgf("Could not get the kubernetes persistenvolumeclaims of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errPVCs.Error())
		} else {
			for _, pvc := range pvcs {
				pvcInternalID := namespacePrefix + "." + kube_model.TypePVC + "." + pvc.Name
				nodeElement, errNodeElement := utils.CreateElement(pvc, pvc.Name, pvcInternalID, kube_model.TypePVC, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, pvcInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		// list ingressesExtensionsBeta1
		ingressesExtensionsBeta1, errIngressesExtensionsBeta1 := kubeCrawler.listIngressesExtensionsBeta1(namespace.Name)
		if errIngressesExtensionsBeta1 != nil {
			log.Warn().Msgf("Could not get the kubernetes ingresses extensions beta1 of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errIngressesExtensionsBeta1.Error())
		} else {
			for _, ingress := range ingressesExtensionsBeta1 {
				ingressInternalID := namespacePrefix + "." + kube_model.TypeIngressExtensionBeta1 + "." + ingress.Name
				nodeElement, errNodeElement := utils.CreateElement(ingress, ingress.Name, ingressInternalID, kube_model.TypeIngressExtensionBeta1, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, ingressInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		ingressesNetworkingV1, errIngressesNetworkingV1 := kubeCrawler.listIngressesNetworkingV1(namespace.Name)
		if errIngressesNetworkingV1 != nil {
			log.Warn().Msgf("Could not get the kubernetes ingresses extensions beta1 of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errIngressesExtensionsBeta1.Error())
		} else {
			for _, ingress := range ingressesNetworkingV1 {
				ingressInternalID := namespacePrefix + "." + kube_model.TypeIngressNetworkingV1 + "." + ingress.Name
				nodeElement, errNodeElement := utils.CreateElement(ingress, ingress.Name, ingressInternalID, kube_model.TypeIngressNetworkingV1, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, ingressInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		ingressesNetworkingV1Beta1, errIngressesNetworkingV1Beta1 := kubeCrawler.listIngressesNetworkingV1Beta1(namespace.Name)
		if errIngressesNetworkingV1Beta1 != nil {
			log.Warn().Msgf("Could not get the kubernetes ingresses extensions beta1 of data source name: %s because %s", kubeCrawler.dataSource.Info.Name, errIngressesExtensionsBeta1.Error())
		} else {
			for _, ingress := range ingressesNetworkingV1Beta1 {
				ingressInternalID := namespacePrefix + "." + kube_model.TypeIngressNetworkingV1Beta1 + "." + ingress.Name
				nodeElement, errNodeElement := utils.CreateElement(ingress, ingress.Name, ingressInternalID, kube_model.TypeIngressNetworkingV1Beta1, crawlTime)
				if errNodeElement != nil {
					continue
				}

				rel, errRel := utils.CreateRelationship(namespacePrefix, ingressInternalID, kube_model.RelationshipSkipinsert, kube_model.RelationshipSkipinsert, crawlTime)
				if errRel == nil {
					allCrawledElements = append(allCrawledElements, rel)
				}

				allCrawledElements = append(allCrawledElements, nodeElement)
			}
		}

		crawledData := bloopi_agent.CrawledData{
			Data: allCrawledElements,
		}

		log.Info().Msgf("Crawled %d Kubernetes elements for connection %s and namespace %s", len(allCrawledElements), kubeCrawler.dataSource.Info.Name, namespace.Name)

		kubeCrawler.outputChannel <- &bloopi_agent.CloudCrawlData{
			Timestamp:       time.Now().UTC(),
			DataSource:      kubeCrawler.dataSource,
			CrawledData:     crawledData,
			CrawlInternalID: fmt.Sprintf("%s.%s", namespace.Name, kube_model.TypeNamespace),
		}

	}

	if !kubeCrawler.istioConfigured {
		return nil, nil
	}

	istioRelationships, errIstioRelationships := kubeCrawler.getIstioRelationships(prefix)
	if errIstioRelationships != nil {
		log.Info().Msgf("There was an error finding the istio relationships for kubernetes connection %s because %s", kubeCrawler.dataSource.Info.Name, errIstioRelationships.Error())
		return nil, errIstioRelationships
	}

	istioElements := []*bloopi_agent.Element{}
	for _, istioRelaitonship := range istioRelationships {
		istioElem, errIstioElem := utils.CreateElement(
			istioRelaitonship,
			fmt.Sprintf("%s-%s", istioRelaitonship.SourceID, istioRelaitonship.DestinationID),
			fmt.Sprintf("%s-%s", istioRelaitonship.SourceID, istioRelaitonship.DestinationID),
			kube_model.FlowIstioRelationshipSkipinsert,
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
		Timestamp:   time.Now().UTC(),
		DataSource:  kubeCrawler.dataSource,
		CrawledData: crawledData,
	}

	return nil, nil
}
