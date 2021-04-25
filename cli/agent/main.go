package main

import (
	"bytes"
	"cleye/integrations/aws"
	"cleye/integrations/clouds"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	crawlInterval = kingpin.Flag("crawl-interval", "The amount of time in seconds when to trigger a crawl of the infrastructure.").Default("30").OverrideDefaultFromEnvar("CLEYE_CRAWLS_INTERVAL").Int()
	cloudToCrawl  = kingpin.Flag("cloud", "What type of cloud infrastructure are we crawling.").HintOptions("aws", "azure").Default("aws").OverrideDefaultFromEnvar("CLEYE_CLOUD_TYPE").String()
	endpoint      = kingpin.Flag("endpoint", "The server URL where to send data.").Default("http://localhost:8000/crawlers/infra/aws").OverrideDefaultFromEnvar("CLEYE_ENDPOINT").String()
	serverKey     = kingpin.Flag("key", "The authentication API key.").Default("bloopie-test-key").OverrideDefaultFromEnvar("BLOOPIE_KEY").String()
)

func main() {
	kingpin.Version("0.1.0")
	kingpin.Parse()

	sender := make(chan *clouds.CloudCrawlData, 5000)

	go func() {
		for range time.Tick(time.Second * time.Duration(*crawlInterval)) {
			switch *cloudToCrawl {
			case "aws":
				// crawl AWS and send json back to the backend
				awsData, err := aws.Crawl()
				if err != nil {
					fmt.Println(err.Error())
					return
				}

				sender <- awsData
			}
		}
	}()

	for crawledData := range sender {
		// call the endpoint
		httpClient := http.Client{
			Timeout: 15 * time.Second,
		}

		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(crawledData)

		req, err := http.NewRequest("POST", *endpoint, b)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		req.Header.Add("BLOOPIE_KEY", *serverKey)
		resp, err := httpClient.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		defer resp.Body.Close()
	}

	fmt.Println("Goodbye!!!")
}
