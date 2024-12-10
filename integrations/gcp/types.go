package gcp

import (
	"time"

	"dev.azure.com/bloopi/bloopi/_git/shared_models.git/bloopi_agent"
	"google.golang.org/api/option"
)

const (
	gcpConfigInGoogleCloud   = "in_cloud"
	gcpConfigCredentialsFile = "credentials_file"
	gcpConfigCrawlInterval   = "crawl_interval"
	gcpProjectID             = "project_id"
)

// ServiceAccountKey represents the complete structure of a Google Cloud service account key JSON file
type ServiceAccountKey struct {
	// Required fields that are always present in service account keys
	Type         string `json:"type"`           // Always "service_account"
	ProjectID    string `json:"project_id"`     // The GCP project ID
	PrivateKeyID string `json:"private_key_id"` // Unique identifier for the private key
	PrivateKey   string `json:"private_key"`    // The PEM-encoded private key
	ClientEmail  string `json:"client_email"`   // Service account email address
	ClientID     string `json:"client_id"`      // Unique identifier for the service account

	// Optional fields that might be present
	AuthURI                 string `json:"auth_uri"`                    // OAuth2 auth URI (usually https://accounts.google.com/o/oauth2/auth)
	TokenURI                string `json:"token_uri"`                   // OAuth2 token URI (usually https://oauth2.googleapis.com/token)
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"` // X509 cert URL for auth provider
	ClientX509CertURL       string `json:"client_x509_cert_url"`        // X509 cert URL for this service account
}

type gcpCrawler struct {
	clientOpts          []option.ClientOption
	InGCPEnvironment    bool
	credentialsFile     string
	crawlInterval       time.Duration
	dataSource          bloopi_agent.DataSource
	outputChan          chan *bloopi_agent.CloudCrawlData
	ConfiguredProjectID string
}

type Crawler interface {
	Crawl()
}
