package aws

type AWS struct {
	Version string
}

// VPC Generic structure of an AWS VPC
type VPC struct {
	VPCId     string
	IPv4      string
	IPv6      string
	IsDefault bool
}
