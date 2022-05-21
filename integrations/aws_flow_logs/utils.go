package awsflowlogs

import (
	"fmt"
	"net"
	"strings"

	awsflowlogs "dev.azure.com/bloopi/bloopi/_git/shared_models.git/aws_flow_logs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ipToCheck string) bool {
	ip := net.ParseIP(ipToCheck)
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func connectToAWS(region string) (*session.Session, error) {
	config := aws.Config{Region: aws.String(region)}
	sess, errSession := session.NewSession(&config)
	if errSession != nil {
		return nil, fmt.Errorf("problems with connection to AWS because %w", errSession)
	}
	return sess, nil
}

func getColumnIndexFromLogFormat(logFormat, columnName string) (int, error) {
	logFormatStripped := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(logFormat, "}", ""), "{", ""), "$", "")

	logFormatSlice := strings.Split(logFormatStripped, " ")

	for index, logFormatFieldName := range logFormatSlice {
		if logFormatFieldName == columnName {
			return index, nil
		}
	}

	return -1, fmt.Errorf("could not find %s in the log format %s", columnName, logFormat)
}

func getRowValue(row []string, logFormat, columnName string) string {
	colIndex, errColIndex := getColumnIndexFromLogFormat(logFormat, columnName)
	if errColIndex != nil {
		return ""
	}

	return row[colIndex]
}

func isInternalFlow(flow awsflowlogs.AWSFlowLog) bool {
	return isPrivateIP(flow.SrcAddr) && isPrivateIP(flow.DstAddr)
}
