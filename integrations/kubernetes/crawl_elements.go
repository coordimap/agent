package kubernetes

import (
	"cleye/utils"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	kube_model "dev.azure.com/bloopi/bloopi/_git/shared_models.git/kubernetes"
	promV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	extensionsBeta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func (kubeCrawler *kubernetesCrawler) getNodes() ([]v1.Node, error) {
	list, err := kubeCrawler.kubeClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not load the kubernetes nodes. %w", err)
	}

	for _, item := range list.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listNamespaces() ([]v1.Namespace, error) {
	list, err := kubeCrawler.kubeClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not load the kubernetes namespaces. %w", err)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listPods(namespace string) ([]v1.Pod, error) {
	podList, errPods := kubeCrawler.kubeClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if errPods != nil {
		return nil, fmt.Errorf("could not list the pods for namespace: %s. %w", namespace, errPods)
	}

	for _, item := range podList.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return podList.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listDeplyments(namespace string) ([]appsv1.Deployment, error) {
	list, errPods := kubeCrawler.kubeClient.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if errPods != nil {
		return nil, fmt.Errorf("could not list the deployments for namespace: %s. %w", namespace, errPods)
	}

	for _, item := range list.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listDeplymentPods(deployment *appsv1.Deployment, namespace string) ([]bloopi_agent.RelationshipElement, error) {
	allDeploymentPodsRelationships := []bloopi_agent.RelationshipElement{}

	set := labels.Set(deployment.Spec.Selector.MatchLabels)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := kubeCrawler.kubeClient.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	for _, pod := range pods.Items {
		allDeploymentPodsRelationships = append(allDeploymentPodsRelationships, bloopi_agent.RelationshipElement{
			SourceID:         generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, deployment.Name),
			DestinationID:    generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, pod.Name),
			RelationshipType: kube_model.RelationshipTypeDeploymentPod,
			RelationType:     bloopi_agent.ParentChildTypeRelation,
		})
	}

	return allDeploymentPodsRelationships, err
}

func (kubeCrawler *kubernetesCrawler) listServices(namespace string) ([]v1.Service, error) {
	list, errPods := kubeCrawler.kubeClient.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{})
	if errPods != nil {
		return nil, fmt.Errorf("could not list the services for namespace: %s. %w", namespace, errPods)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listServicePods(service *v1.Service, namespace string) ([]bloopi_agent.RelationshipElement, error) {
	allServicePodsRelationships := []bloopi_agent.RelationshipElement{}

	set := labels.Set(service.Spec.Selector)
	listOptions := metav1.ListOptions{LabelSelector: set.AsSelector().String()}
	pods, err := kubeCrawler.kubeClient.CoreV1().Pods(namespace).List(context.Background(), listOptions)
	for _, pod := range pods.Items {
		allServicePodsRelationships = append(allServicePodsRelationships, bloopi_agent.RelationshipElement{
			SourceID:         generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, service.Name),
			DestinationID:    generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, pod.Name),
			RelationshipType: kube_model.RelationshipTypeServicePod,
			RelationType:     bloopi_agent.ParentChildTypeRelation,
		})
	}

	return allServicePodsRelationships, err
}

func (kubeCrawler *kubernetesCrawler) listSecrets(namespace string) ([]v1.Secret, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the secrets for namespace: %s. %w", namespace, errList)
	}

	for _, item := range list.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listJobs(namespace string) ([]batchv1.Job, error) {
	list, errList := kubeCrawler.kubeClient.BatchV1().Jobs(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the jobs for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listCronJobs(namespace string) ([]batchv1.CronJob, error) {
	list, errList := kubeCrawler.kubeClient.BatchV1().CronJobs(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the cronjobs for namespace: %s. %w", namespace, errList)
	}

	for _, item := range list.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listEndpoints(namespace string) ([]v1.Endpoints, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().Endpoints(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the endpoints for namespace: %s. %w", namespace, errList)
	}

	for _, item := range list.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listConfigMaps(namespace string) ([]v1.ConfigMap, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().ConfigMaps(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the configmaps for namespace: %s. %w", namespace, errList)
	}

	// set managed fields
	for _, item := range list.Items {
		clearManagedFields(&item.ObjectMeta)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listStatefulSets(namespace string) ([]appsv1.StatefulSet, error) {
	list, errList := kubeCrawler.kubeClient.AppsV1().StatefulSets(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the configmaps for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listDaemonSets(namespace string) ([]appsv1.DaemonSet, error) {
	list, errList := kubeCrawler.kubeClient.AppsV1().DaemonSets(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the configmaps for namespace: %s. %w", namespace, errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listStorageClasses() ([]storagev1.StorageClass, error) {
	list, errList := kubeCrawler.kubeClient.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the storageclasses. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listPersistentVolumeClaims(namespace string) ([]v1.PersistentVolumeClaim, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the storageclasses. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listPersistentVolumes() ([]v1.PersistentVolume, error) {
	list, errList := kubeCrawler.kubeClient.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the storageclasses. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listIngressesExtensionsBeta1(namespace string) ([]extensionsBeta1.Ingress, error) {
	list, errList := kubeCrawler.kubeClient.ExtensionsV1beta1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the ingresses extensions beta1. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listIngressesNetworkingV1(namespace string) ([]networkingv1.Ingress, error) {
	list, errList := kubeCrawler.kubeClient.NetworkingV1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the ingresses networkingv1. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) listIngressesNetworkingV1Beta1(namespace string) ([]networkingv1beta1.Ingress, error) {
	list, errList := kubeCrawler.kubeClient.NetworkingV1beta1().Ingresses(namespace).List(context.Background(), metav1.ListOptions{})
	if errList != nil {
		return nil, fmt.Errorf("could not list the ingresses networkingv1 beta1. %w", errList)
	}

	return list.Items, nil
}

func (kubeCrawler *kubernetesCrawler) getLabelElementsAndRelationships(elemInternalID, namespace string, labelsToCheckIn map[string]string, createdElementsFromLabels []string, crawlTime time.Time) ([]*bloopi_agent.Element, []string) {
	allFoundElementsAndRelationships := []*bloopi_agent.Element{}
	createdElements := []string{}

	// check for helm chart
	helmChartLabel := "helm.sh/chart"
	if name, isHelmChart := labelsToCheckIn[helmChartLabel]; isHelmChart {
		chartInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, name)

		if !slices.Contains(createdElementsFromLabels, chartInternalID) {
			if elem, errElem := utils.CreateElement(kube_model.KubernetesChart{Name: name}, name, chartInternalID, kube_model.TypeHelmChart, crawlTime); errElem == nil {
				allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, elem)
				createdElements = append(createdElements, chartInternalID)
			}
		}

		if rel, errRel := utils.CreateRelationship(chartInternalID, elemInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
		}
	}

	// check for part-of
	partOfLabel := "app.kubernetes.io/part-of"
	partOfLabelValue, partOfLabelExists := labelsToCheckIn[partOfLabel]
	partOfLabelInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, partOfLabelValue)
	if partOfLabelExists {
		if elem, errElem := utils.CreateElement(kube_model.KubernetesLabelComponent{Name: partOfLabelValue}, partOfLabelValue, partOfLabelInternalID, kube_model.TypeLabelPartOf, crawlTime); errElem == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, elem)
			createdElements = append(createdElements, partOfLabelInternalID)
		}

		if rel, errRel := utils.CreateRelationship(partOfLabelInternalID, elemInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
		}
	}

	// check for component
	componentLabel := "app.kubernetes.io/component"
	componentLabelValue, componentLabelExists := labelsToCheckIn[componentLabel]
	componentLabelInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, componentLabelValue)
	if componentLabelExists {
		if elem, errElem := utils.CreateElement(kube_model.KubernetesLabelComponent{Name: componentLabelValue}, componentLabelValue, componentLabelInternalID, kube_model.TypeLabelName, crawlTime); errElem == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, elem)
			createdElements = append(createdElements, componentLabelInternalID)
		}

		if rel, errRel := utils.CreateRelationship(componentLabelInternalID, elemInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
		}

		if partOfLabelExists {
			if rel, errRel := utils.CreateRelationship(partOfLabelInternalID, componentLabelInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
				allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
			}
		}
	}

	nameLabel := "app.kubernetes.io/name"
	nameLabelValue, nameLabelExists := labelsToCheckIn[nameLabel]
	nameLabelInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, namespace, nameLabelValue)
	if nameLabelExists {
		if elem, errElem := utils.CreateElement(kube_model.KubernetesLabelName{Name: nameLabelValue}, nameLabelValue, nameLabelInternalID, kube_model.TypeLabelName, crawlTime); errElem == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, elem)
			createdElements = append(createdElements, nameLabelInternalID)
		}

		if rel, errRel := utils.CreateRelationship(nameLabelInternalID, elemInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
			allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
		}

		if partOfLabelExists {
			if rel, errRel := utils.CreateRelationship(partOfLabelInternalID, nameLabelInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
				allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
			}
		}

		if componentLabelExists {
			if rel, errRel := utils.CreateRelationship(componentLabelInternalID, nameLabelInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); errRel == nil {
				allFoundElementsAndRelationships = append(allFoundElementsAndRelationships, rel)
			}
		}
	}

	return allFoundElementsAndRelationships, createdElements
}

func (kubeCrawler *kubernetesCrawler) getRetinaFlowsRelationships(crawlTime time.Time) ([]*bloopi_agent.Element, error) {
	allFoundRelationships := []*bloopi_agent.Element{}
	sourcePromQuery := "networkobservability_adv_forward_bytes[5m]"

	v1api := promV1.NewAPI(kubeCrawler.retinaCrawler.promClient)
	ctx, cancelQuery := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelQuery()

	resultSourcePromQuery, _, errSourcePromQuery := v1api.Query(ctx, sourcePromQuery, time.Now(), promV1.WithTimeout(15*time.Second))
	if errSourcePromQuery != nil {
		log.Error().Msgf("Cannot query Istio sources because an error happened: %s", errSourcePromQuery.Error())
		return nil, fmt.Errorf("cannot query Istio sources because an error happened: %w", errSourcePromQuery)
	}

	for _, source := range resultSourcePromQuery.(model.Matrix) {

		isRageEqual := true
		for _, val := range source.Values {
			if val.Value != source.Values[0].Value {
				isRageEqual = false
				break
			}

			if isRageEqual {
				continue
			}
		}

		metric := source.Metric

		if (metric["destination_namespace"] == "unknown" && metric["destination_podname"] == "unknown") || (metric["source_namespace"] == "unknown" && metric["source_podname"] == "unknown") {
			continue
		}

		sourceInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(metric["source_namespace"]), string(metric["source_podname"]))
		destinationInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(metric["destination_namespace"]), string(metric["destination_podname"]))

		if rel, errRel := utils.CreateRelationship(sourceInternalID, destinationInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.FlowTypeRelation, crawlTime); errRel == nil {
			allFoundRelationships = append(allFoundRelationships, rel)
		}

		if metric["destination_workload_kind"] == "ReplicaSet" {
			hyphenIndex := strings.LastIndex(string(metric["destination_workload_name"]), "-")
			workloadName := string(metric["destination_workload_name"])[0:hyphenIndex]
			deploymentInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(metric["destination_namespace"]), workloadName)
			if rel, err := utils.CreateRelationship(sourceInternalID, deploymentInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.FlowTypeRelation, crawlTime); err == nil {
				allFoundRelationships = append(allFoundRelationships, rel)
			}
		}

		if metric["source_workload_kind"] == "ReplicaSet" {
			hyphenIndex := strings.LastIndex(string(metric["source_workload_name"]), "-")
			workloadName := string(metric["source_workload_name"])[0:hyphenIndex]
			deploymentInternalID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(metric["source_namespace"]), workloadName)
			if rel, err := utils.CreateRelationship(deploymentInternalID, destinationInternalID, bloopi_agent.RelationshipType, bloopi_agent.RelationshipType, bloopi_agent.ParentChildTypeRelation, crawlTime); err == nil {
				allFoundRelationships = append(allFoundRelationships, rel)
			}
		}
	}

	return allFoundRelationships, nil
}

// crawl, queries the prometheus endpoint to get the data regarding the istio relationships
func (kubeCrawler *kubernetesCrawler) getIstioRelationships() ([]bloopi_agent.RelationshipElement, error) {
	istioMappingFromQueries := map[string]bloopi_agent.RelationshipElement{}
	allFoundRelationships := []bloopi_agent.RelationshipElement{}
	if !kubeCrawler.istioConfigured {
		return allFoundRelationships, nil
	}

	promBaseQuery := `sum(rate(istio_requests_total{reporter="%s"}[%s])) by (source_workload_namespace, destination_workload_namespace, source_app, destination_app, source_canonical_service, destination_canonical_service, source_workload, destination_workload, pod)`
	v1api := promV1.NewAPI(kubeCrawler.istioCrawler.promClient)
	ctx, cancelQuery := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelQuery()

	// get the source
	sourcePromQuery := fmt.Sprintf(promBaseQuery, "source", kubeCrawler.istioCrawler.promQueryTime)
	resultSourcePromQuery, warningsSourcePromQuery, errSourcePromQuery := v1api.Query(ctx, sourcePromQuery, time.Now(), promV1.WithTimeout(5*time.Second))
	if errSourcePromQuery != nil {
		log.Error().Msgf("Cannot query Istio sources because an error happened: %s", errSourcePromQuery.Error())
		return nil, fmt.Errorf("cannot query Istio sources because an error happened: %w", errSourcePromQuery)
	}

	if len(warningsSourcePromQuery) > 0 {
		log.Warn().Strs("Istio Prometheus Warnings", warningsSourcePromQuery).Msg("Source Warnings")
	}

	// generate key from the labels. Keep in mind that the kubernetes service internal id is: <namespace name>.TypeService.<service name>
	// There are three types of relationships:
	// 1. service to service
	// 2. pod to pod
	// 3. workload to workload (mainly deployment)
	for _, source := range resultSourcePromQuery.(model.Vector) {
		if source.Value == 0 {
			// nothing happened during the queried time
			continue
		}

		sourceCanonicalService := source.Metric["source_canonical_service"]
		sourceWorkload := source.Metric["source_workload"]
		sourceWorkloadNamespace := source.Metric["source_workload_namespace"]
		destinationCanonicalService := source.Metric["destination_canonical_service"]
		destinationWorkload := source.Metric["destination_workload"]
		destinationWorkloadNamespace := source.Metric["destination_workload_namespace"]
		pod := source.Metric["pod"]

		if sourceCanonicalService == "unknown" || sourceWorkload == "unknown" || sourceWorkloadNamespace == "unknown" || destinationCanonicalService == "unknown" || destinationWorkload == "unknown" || destinationWorkloadNamespace == "unknown" {
			continue
		}

		if sourceCanonicalService != "unknown" && sourceWorkloadNamespace != "unknown" && destinationCanonicalService != "unknown" && destinationWorkloadNamespace != "unknown" {
			// create a relationship between the services and create a ISTIO_RELATIONSHIP_TYPE_SERVICE relationship
			sourceID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(sourceWorkloadNamespace), string(sourceCanonicalService))
			destinationID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(destinationWorkloadNamespace), string(destinationCanonicalService))

			allFoundRelationships = append(allFoundRelationships, bloopi_agent.RelationshipElement{
				SourceID:         sourceID,
				DestinationID:    destinationID,
				RelationshipType: kube_model.FlowIstioRelationshipTypeService,
				RelationType:     bloopi_agent.FlowTypeRelation,
			})

			istioMappingFromQueries[fmt.Sprintf("%s@%s", sourceID, destinationID)] = bloopi_agent.RelationshipElement{}
		}

		if sourceWorkload != "unknown" && sourceWorkloadNamespace != "unknown" && destinationWorkload != "unknown" && destinationWorkloadNamespace != "unknown" {
			// create a relationship between the deployments and create a ISTIO_RELATIONSHIP_TYPE_DEPLOYMENT relationship
			sourceID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(sourceWorkloadNamespace), string(sourceWorkload))
			destinationID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(destinationWorkloadNamespace), string(destinationWorkload))

			allFoundRelationships = append(allFoundRelationships, bloopi_agent.RelationshipElement{
				SourceID:         sourceID,
				DestinationID:    destinationID,
				RelationshipType: kube_model.FlowIstioRelationshipTypeDeployment,
				RelationType:     bloopi_agent.FlowTypeRelation,
			})

			istioMappingFromQueries[fmt.Sprintf("%s@%s", sourceID, destinationID)] = bloopi_agent.RelationshipElement{}
		}

		istioMappingFromQueries[fmt.Sprintf("%s.%s.%s-%s.%s.%s", sourceWorkload, sourceCanonicalService, sourceWorkloadNamespace, destinationWorkload, destinationCanonicalService, destinationWorkloadNamespace)] = bloopi_agent.RelationshipElement{
			SourceID:         string(pod),
			DestinationID:    "",
			RelationshipType: kube_model.FlowIstioRelationshipTypePod,
			RelationType:     bloopi_agent.FlowTypeRelation,
		}
	}

	// get the source
	destinationPromQuery := fmt.Sprintf(promBaseQuery, "destination", kubeCrawler.istioCrawler.promQueryTime)
	resultDestinationPromQuery, warningsDestinationPromQuery, errDestinationPromQuery := v1api.Query(ctx, destinationPromQuery, time.Now(), promV1.WithTimeout(5*time.Second))
	if errDestinationPromQuery != nil {
		log.Error().Msgf("Cannot query Istio destinations because an error happened: %s", errDestinationPromQuery.Error())
		return nil, fmt.Errorf("cannot query Istio destinations because an error happened: %w", errDestinationPromQuery)
	}

	if len(warningsDestinationPromQuery) > 0 {
		log.Warn().Strs("Istio Prometheus Warnings", warningsDestinationPromQuery).Msg("Destination Warnings")
	}

	for _, destination := range resultDestinationPromQuery.(model.Vector) {
		if destination.Value == 0 {
			// nothing happened during the queried time
			continue
		}

		sourceCanonicalService := destination.Metric["source_canonical_service"]
		sourceWorkload := destination.Metric["source_workload"]
		sourceWorkloadNamespace := destination.Metric["source_workload_namespace"]
		destinationCanonicalService := destination.Metric["destination_canonical_service"]
		destinationWorkload := destination.Metric["destination_workload"]
		destinationWorkloadNamespace := destination.Metric["destination_workload_namespace"]
		pod := destination.Metric["pod"]

		if sourceCanonicalService == "unknown" || sourceWorkload == "unknown" || sourceWorkloadNamespace == "unknown" || destinationCanonicalService == "unknown" || destinationWorkload == "unknown" || destinationWorkloadNamespace == "unknown" {
			continue
		}

		if sourceCanonicalService != "unknown" && sourceWorkloadNamespace != "unknown" && destinationCanonicalService != "unknown" && destinationWorkloadNamespace != "unknown" {
			// create a relationship between the services and create a ISTIO_RELATIONSHIP_TYPE_SERVICE relationship
			sourceID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(sourceWorkloadNamespace), string(sourceCanonicalService))
			destinationID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(destinationWorkloadNamespace), string(destinationCanonicalService))
			relationshipKey := fmt.Sprintf("%s@%s", sourceID, destinationID)

			if _, ok := istioMappingFromQueries[relationshipKey]; !ok {
				allFoundRelationships = append(allFoundRelationships, bloopi_agent.RelationshipElement{
					SourceID:         sourceID,
					DestinationID:    destinationID,
					RelationshipType: kube_model.FlowIstioRelationshipTypeService,
					RelationType:     bloopi_agent.FlowTypeRelation,
				})

				istioMappingFromQueries[relationshipKey] = bloopi_agent.RelationshipElement{}
			}
		}

		if sourceWorkload != "unknown" && sourceWorkloadNamespace != "unknown" && destinationWorkload != "unknown" && destinationWorkloadNamespace != "unknown" {
			// create a relationship between the deployments and create a ISTIO_RELATIONSHIP_TYPE_DEPLOYMENT relationship
			sourceID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(sourceWorkloadNamespace), string(sourceWorkload))
			destinationID := generateInternalName(kubeCrawler.dataSource.DataSourceID, string(destinationWorkloadNamespace), string(destinationWorkload))
			relationshipKey := fmt.Sprintf("%s@%s", sourceID, destinationID)

			if _, ok := istioMappingFromQueries[relationshipKey]; !ok {
				allFoundRelationships = append(allFoundRelationships, bloopi_agent.RelationshipElement{
					SourceID:         sourceID,
					DestinationID:    destinationID,
					RelationshipType: kube_model.FlowIstioRelationshipTypeDeployment,
					RelationType:     bloopi_agent.FlowTypeRelation,
				})

				istioMappingFromQueries[relationshipKey] = bloopi_agent.RelationshipElement{}
			}
		}

		// find the correct key and fill in the destinationID
		key := fmt.Sprintf("%s.%s.%s-%s.%s.%s", sourceWorkload, sourceCanonicalService, sourceWorkloadNamespace, destinationWorkload, destinationCanonicalService, destinationWorkloadNamespace)

		if entry, ok := istioMappingFromQueries[key]; ok {
			entry.DestinationID = string(pod)
			allFoundRelationships = append(allFoundRelationships, entry)
		}

	}
	return allFoundRelationships, nil
}
