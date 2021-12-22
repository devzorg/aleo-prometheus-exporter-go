// A minimal example of how to include Prometheus instrumentation.
package main

import (
  "encoding/json"
	"crypto/tls"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "aleo"

var (
	tr = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}

	listenAddress = flag.String("web.listen-address", ":9200",
		"Address to listen on for telemetry")
	metricsPath = flag.String("web.telemetry-path", "/metrics",
		"Path under which to expose metrics")

	// Metrics
	nodeType = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "type"),
		"Type of node.",
		nil, nil,
	)
	nodeStatus = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "status"),
		"Node status.",
		[]string{"channel"}, nil,
	)
	connectedSyncNodes = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "connected_sync_nodes"),
		"Number of connected sync nodes.",
		[]string{"channel"}, nil,
	)
	connectedPeers = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "connected_peers"),
		"Number of connected peers.",
		[]string{"channel"}, nil,
	)
	candidatePeers = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "candidate_peers"),
		"Number of candidate peers",
		[]string{"channel"}, nil,
	)
	cumulativeWeight = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "cumulative_weight"),
		".",
		[]string{"channel"}, nil,
	)
  latestBlockHeight = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "latest_block_height"),
		"Latest block height of node.",
		[]string{"channel"}, nil,
	)
  blocksMined = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "blocks_mined"),
		"Blocks mined after 18000 block.",
		[]string{"channel"}, nil,
	)
  blocksMinedCalibrate = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "blocks_mined_calibrate"),
		"Blocks mined before 18000 block.",
		[]string{"channel"}, nil,
	)
)

type Exporter struct {
	aleorpcEndpoint, aleorpcUsername, aleorpcPassword string
}

func NewExporter(aleorpcEndpoint string, aleorpcUsername string, aleorpcPassword string) *Exporter {
	return &Exporter{
		aleorpcEndpoint: aleorpcEndpoint,
		aleorpcUsername: aleorpcUsername,
		aleorpcPassword: aleorpcPassword,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- nodeType
  ch <- nodeStatus
  ch <- connectedSyncNodes
  ch <- connectedPeers
  ch <- candidatePeers
	ch <- cumulativeWeight
	ch <- latestBlockHeight
	ch <- blocksMined
	ch <- blocksMinedCalibrate
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	channelIdNameMap, err := e.LoadChannelIdNameMap()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(
			up, prometheus.GaugeValue, 0,
		)
		log.Println(err)
		return
	}
	ch <- prometheus.MustNewConstMetric(
		up, prometheus.GaugeValue, 1,
	)

	e.HitMirthRestApisAndUpdateMetrics(channelIdNameMap, ch)
}

func (e *Exporter) LoadChannelIdNameMap() (map[string]string, error) {
	// Create the map of channel id to names
	channelIdNameMap := make(map[string]string)

	req, err := http.NewRequest("GET", e.aleorpcEndpoint+channelIdNameApi, nil)
	if err != nil {
		return nil, err
	}

	// This one line implements the authentication required for the task.
	req.SetBasicAuth(e.aleorpcUsername, e.aleorpcPassword)
	// Make request and show output.
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// fmt.Println(string(body))

	// we initialize our array
	var channelIdNameMapXML ChannelIdNameMap
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'users' which we defined above
	err = xml.Unmarshal(body, &channelIdNameMapXML)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(channelIdNameMapXML.Entries); i++ {
		channelIdNameMap[channelIdNameMapXML.Entries[i].Values[0]] = channelIdNameMapXML.Entries[i].Values[1]
	}

	return channelIdNameMap, nil
}

func (e *Exporter) HitMirthRestApisAndUpdateMetrics(channelIdNameMap map[string]string, ch chan<- prometheus.Metric) {
	// Load channel stats
	req, err := http.NewRequest("GET", e.aleorpcEndpoint, nil)
	if err != nil {
		log.Fatal(err)
	}

	// This one line implements the authentication required for the task.
	req.SetBasicAuth(e.aleorpcUsername, e.aleorpcPassword)
	// Make request and show output.
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	// fmt.Println(string(body))

	// we initialize our array
	var channelStatsList ChannelStatsList
	// we unmarshal our byteArray which contains our
	// xmlFiles content into 'users' which we defined above
	err = xml.Unmarshal(body, &channelStatsList)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(channelStatsList.Channels); i++ {
		channelName := channelIdNameMap[channelStatsList.Channels[i].ChannelId]

		channelReceived, _ := strconv.ParseFloat(channelStatsList.Channels[i].Received, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesReceived, prometheus.GaugeValue, channelReceived, channelName,
		)

		channelSent, _ := strconv.ParseFloat(channelStatsList.Channels[i].Sent, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesSent, prometheus.GaugeValue, channelSent, channelName,
		)

		channelError, _ := strconv.ParseFloat(channelStatsList.Channels[i].Error, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesErrored, prometheus.GaugeValue, channelError, channelName,
		)

		channelFiltered, _ := strconv.ParseFloat(channelStatsList.Channels[i].Filtered, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesFiltered, prometheus.GaugeValue, channelFiltered, channelName,
		)

		channelQueued, _ := strconv.ParseFloat(channelStatsList.Channels[i].Queued, 64)
		ch <- prometheus.MustNewConstMetric(
			messagesQueued, prometheus.GaugeValue, channelQueued, channelName,
		)
	}

	log.Println("Endpoint scraped")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file, assume env variables are set.")
	}

	flag.Parse()

	aleorpcEndpoint := os.Getenv("ALEO_RPC_ENDPOINT")
	aleorpcUsername := os.Getenv("ALEO_RPC_USERNAME")
	aleorpcPassword := os.Getenv("ALEO_RPC_PASSWORD")

	exporter := NewExporter(aleorpcEndpoint, aleorpcUsername, aleorpcPassword)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Aleo Exporter</title></head>
             <body>
             <h1>Aleo Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
