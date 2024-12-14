package gcpflowlogs

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

const (
	gcpFlowLogsConfigInGoogleCloud   = "in_cloud"
	gcpFlowLogsConfigCredentialsFile = "credentials_file"
	gcpFlowLogsConfigCrawlInterval   = "crawl_interval"
	gcpFlowLogsProjectID             = "project_id"
)

type gcpFlowLogsCrawler struct {
	clientOpts          []option.ClientOption
	InGCPEnvironment    bool
	credentialsFile     string
	crawlInterval       time.Duration
	dataSource          bloopi_agent.DataSource
	outputChan          chan *bloopi_agent.CloudCrawlData
	ConfiguredProjectID string
	client              *logging.Service
}

type Crawler interface {
	Crawl()
}

type IpConnection struct {
	Protocol int    `json:"protocol"`
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dest_ip"`
	SrcPort  int    `json:"src_port"`
	DstPort  int    `json:"dest_port"`
}

type GatewayDetails struct {
	ProjectID string     `json:"project_id"`
	Location  string     `json:"location"`
	Name      string     `json:"name"`
	Type      string     `json:"type"`
	Vpc       VpcDetails `json:"vpc"`
}

type ClusterDetails struct {
	ClusterLocation string `json:"cluster_location"`
	ClusterName     string `json:"cluster_name"`
}

type PodDetails struct {
	Name      string          `json:"pod_name"`
	Namespace string          `json:"pod_namespace"`
	Workload  WorkloadDetails `json:"pod_workload"`
}

type WorkloadDetails struct {
	Name string `json:"workload_name"`
	Type string `json:"workload_type"`
}

type ServiceDetails struct {
	Name      string `json:"service_name"`
	Namespace string `json:"service_namespace"`
}

type GkeDetails struct {
	Cluster ClusterDetails `json:"cluster"`
	Pod     PodDetails     `json:"pod"`
	Service ServiceDetails `json:"service"`
}

type GoogleService struct {
	Type string `json:"type"`
}

type InstanceDetails struct {
	ProjectID string `json:"project_id"`
	Region    string `json:"region"`
	VmName    string `json:"vm_name"`
	Zone      string `json:"zone"`
}

type VpcDetails struct {
	ProjectID    string `json:"project_id"`
	SubnetName   string `json:"subnetwork_name"`
	SubnetRegion string `json:"subnetwork_region"`
	VpcName      string `json:"vpc_name"`
}

type GeographicalDetails struct {
	Asn       int    `json:"asn"`
	City      string `json:"city"`
	Continent string `json:"continent"`
	Country   string `json:"country"`
	Region    string `json:"region"`
}

type flowJSONStructure struct {
	BytesSent        string              `json:"bytes_sent,-,omitempty"`
	PacketsSent      string              `json:"packets_sent,-,omitempty"`
	Connection       IpConnection        `json:"connection,omitempty"`
	StartTime        string              `json:"start_time,omitempty"`
	EndTime          string              `json:"end_time,omitempty"`
	SrcGateway       GatewayDetails      `json:"src_gateway,omitempty"`
	DstGateway       GatewayDetails      `json:"dest_gateway,omitempty"`
	SrcGkeDetails    GkeDetails          `json:"src_gke_details,omitempty"`
	DstGkeDetails    GkeDetails          `json:"dest_gke_details,omitempty"`
	SrcGoogleService GoogleService       `json:"src_google_service,omitempty"`
	DstGoogleService GoogleService       `json:"dest_google_service,omitempty"`
	SrcInstance      InstanceDetails     `json:"src_instance,omitempty"`
	DstInstance      InstanceDetails     `json:"dest_instance,omitempty"`
	SrcLocation      GeographicalDetails `json:"src_location,omitempty"`
	DstLocation      GeographicalDetails `json:"dest_location,omitempty"`
	SrcVpc           VpcDetails          `json:"src_vpc,omitempty"`
	DstVpc           VpcDetails          `json:"dest_vpc,omitempty"`
}
