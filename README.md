# Cleye

Cleye is a data crawler that gathers information from various sources and generates a JSON graph of all the elements. It is written in Go and can be configured to crawl different data sources.

## Getting Started

To get started with Cleye, you\'ll need to have Go installed on your system. You can find the installation instructions for Go in the [official documentation](https://golang.org/doc/install).

### Dependencies

Cleye has the following dependencies:

- [cloud.google.com/go/compute/metadata](http://cloud.google.com/go/compute/metadata)
- [dev.azure.com/bloopi/bloopi/\_git/shared_models.git](http://dev.azure.com/bloopi/bloopi/_git/shared_models.git)
- [github.com/aws/aws-sdk-go](http://github.com/aws/aws-sdk-go)
- [github.com/gertd/go-pluralize](http://github.com/gertd/go-pluralize)
- [github.com/go-redis/redis/v8](http://github.com/go-redis/redis/v8)
- [github.com/go-sql-driver/mysql](http://github.com/go-sql-driver/mysql)
- [github.com/gorilla/mux](http://github.com/gorilla/mux)
- [github.com/lib/pq](http://github.com/lib/pq)
- [github.com/parnurzeal/gorequest](http://github.com/parnurzeal/gorequest)
- [github.com/prometheus/client_golang](http://github.com/prometheus/client_golang)
- [github.com/prometheus/common](http://github.com/prometheus/common)
- [github.com/rs/zerolog](http://github.com/rs/zerolog)
- [github.com/spf13/viper](http://github.com/spf13/viper)
- [go.mongodb.org/mongo-driver](http://go.mongodb.org/mongo-driver)
- [golang.org/x/oauth2](http://golang.org/x/oauth2)
- [google.golang.org/api](http://google.golang.org/api)
- [gopkg.in/alecthomas/kingpin.v2](http://gopkg.in/alecthomas/kingpin.v2)
- [gopkg.in/yaml.v3](http://gopkg.in/yaml.v3)
- [k8s.io/api](http://k8s.io/api)
- [k8s.io/apimachinery](http://k8s.io/apimachinery)
- [k8s.io/client-go](http://k8s.io/client-go)

These dependencies will be automatically downloaded when you build the project.

## Build and Test

To build and run Cleye, you can use the provided Dockerfile. You will need to have Docker installed on your system. You can find the installation instructions for Docker in the [official documentation](https://docs.docker.com/get-docker/).

To build the Docker image, run the following command from the root of the project:

```
docker build -t cleye .
```

Once the image is built, you can run the Cleye agent with the following command:

```
docker run cleye
```

## eBPF Flow Crawler

Cleye includes an eBPF-based flow crawler that can be used to monitor network traffic in a Kubernetes environment. This crawler uses eBPF to capture network flows and map the connections between services and pods.

### eBPF Dependencies

To use the eBPF flow crawler, you will need to have the following additional dependencies installed on your system:

- `clang`
- `llvm`
- `bpftool`

### eBPF Build Step

Before building the Cleye application, you will need to run the following command to generate the eBPF Go files:

```
go generate ./internal/cloud/flows
```

This command will compile the eBPF C code and generate the necessary Go files to interact with it.

### eBPF Configuration

To enable the eBPF flow crawler, you will need to add the following configuration to your `config.yaml` file:

```yaml
- type: flows
  id: "ebpf_flows"
  config:
    - name: crawl_interval
      value: 30s
    - name: deployedAt
      value: "kubernetes" # can be "kubernetes" or "server"
    - name: interface_name
      value: "all" # can be "all" or a specific interface like "eth0"
    - name: external_mappings
      value: "*@your_k8s_cluster_uid" # required when deployedAt is "kubernetes"
```

## Configuration

Cleye is configured using a YAML file. By default, the application looks for a `config.yaml` file in the same directory as the executable. You can specify a different configuration file using the `--config` flag.

The configuration file specifies the data sources to be crawled. Here is an example configuration:

```yaml
datasources:
  - name: "aws"
    type: "aws"
    # Add your AWS specific configuration here
  - name: "gcp"
    type: "gcp"
    # Add your GCP specific configuration here
```

## Supported Data Sources

Here are the supported data sources and their sample configurations:

### GCP

```yaml
- type: gcp
  id: gcp_id_123
  config:
    - name: in_cloud
      value: "false"
    - name: credentials_file
      value: "/path/to/your/credentials.json"
    - name: project_id
      value: "your-gcp-project-id"
    - name: crawl_interval
      value: 30s
    - name: gcp_flows
      value: "true"
    - name: external_mappings
      value: "europe-west3-your-gke-cluster@your_k8s_cluster_uid"
    - name: include_regions
      value: "your-gcp-region"
```

### GCP Flow Logs

```yaml
- type: gcp_flow_logs
  id: gcp_id_123
  config:
    - name: in_cloud
      value: "false"
    - name: credentials_file
      value: "/path/to/your/credentials.json"
    - name: project_id
      value: "your-gcp-project-id"
    - name: crawl_interval
      value: 30s
```

### AWS

```yaml
- type: aws
  id: awstestid
  config:
    - name: policy_config
      value: "true"
    - name: access_key_id
      value: "${AWS_ACCESS_KEY_ID}"
    - name: secret_access_key
      value: "${AWS_SECRET_ACCESS_KEY}"
    - name: crawl_interval
      value: 30s
```

### PostgreSQL

```yaml
- type: postgres
  id: postgres123
  name: "database-name"
  desc: "Description of the database."
  config:
    - name: db_name
      value: "your_db_name"
    - name: db_host
      value: "your_db_host"
    - name: db_user
      value: "your_db_user"
    - name: db_pass
      value: "your_db_password"
    - name: ssl_mode
      value: "require" # or disable, allow, prefer, verify-ca, verify-full
    - name: crawl_interval
      value: 30s
    - name: mapping_internal_id
      value: "your-internal-mapping-id"
```

### MariaDB

```yaml
- type: mariadb
  id: "data_source_123"
  config:
    - name: db_name
      value: "your_db_name"
    - name: db_host
      value: "your_db_host"
    - name: db_user
      value: "your_db_user"
    - name: db_pass
      value: "your_db_password"
    - name: crawl_interval
      value: 30s
```

### Kubernetes

```yaml
- type: kubernetes
  id: "kube_cluster_id"
  config:
    - name: in_cluster
      value: "false"
    - name: cluster_name
      value: "your_cluster_name"
    - name: cluster_uid
      value: "your_k8s_cluster_uid"
    - name: cloud_data_source_id
      value: "your_cloud_data_source_id"
    - name: config_file
      value: "/path/to/your/kube/config"
    - name: crawl_interval
      value: 30s
    - name: external_mappings
      value: "node-1@aws_data_source_id us-central1-a-node-2@gcp_data_source_id"
```

### Generate Kubernetes Cluster UID

The Kubernetes internal names are scoped by `cluster_uid` (not by data source id). You can retrieve the cluster UID from the `kube-system` namespace:

```bash
kubectl get namespace kube-system -o jsonpath='{.metadata.uid}'
```

Use this UID in:

- the Kubernetes data source as `cluster_uid`
- mappings that need to reference Kubernetes internal names (for example, GCP flow logs `external_mappings`)

### AWS Flow Logs

```yaml
- type: aws_flow_logs
  name: "flowlog-name"
  desc: "Description of the flow logs."
  config:
    - name: log_format
      value: "all"
    - name: log_type
      value: "S3"
    - name: account_id
      value: "your_aws_account_id"
    - name: bucket_name
      value: "your_s3_bucket_name"
    - name: region
      value: "your_aws_region"
    - name: access_key_id
      value: "${AWS_ACCESS_KEY_ID}"
    - name: secret_access_key
      value: "${AWS_SECRET_ACCESS_KEY}"
    - name: crawl_interval
      value: 30s
```

### MongoDB

```yaml
- type: mongodb
  name: "mongo-instance-name"
  desc: "Description of the mongo instance."
  config:
    - name: db_name
      value: "*" # or a specific database name
    - name: db_host
      value: "your_mongo_host"
    - name: db_user
      value: "your_mongo_user"
    - name: db_pass
      value: "your_mongo_password"
    - name: crawl_interval
      value: 30s
```

## Contribute

If you would like to contribute to Cleye, please fork the repository and submit a pull request. We welcome all contributions!
