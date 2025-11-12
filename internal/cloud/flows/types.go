package flows

import (
	"net"
	"sync"
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"k8s.io/client-go/kubernetes"
)

// Crawler is the interface for all the crawlers

type Crawler interface {
	Crawl()
}

type flowsCrawler struct {
	outputChannel     chan *bloopi_agent.CloudCrawlData
	dataSource        *bloopi_agent.DataSource
	kubeClientset     *kubernetes.Clientset
	podCache          *PodCache
	crawlInterval     time.Duration
	externalMappingID string
}

type ConnectionData struct {
	SrcIP   net.IP
	DstIP   net.IP
	SrcPort uint16
	DstPort uint16
	Proto   uint8
}

type PodInfo struct {
	Name      string
	Namespace string
	Workload  string
	IP        string
}

type PodCache struct {
	mu    sync.Mutex
	cache map[string]PodInfo
}

func NewPodCache() *PodCache {
	return &PodCache{
		cache: make(map[string]PodInfo),
	}
}

func (c *PodCache) Get(ip string) (PodInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	info, found := c.cache[ip]
	return info, found
}

func (c *PodCache) Set(ip string, info PodInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[ip] = info
}
