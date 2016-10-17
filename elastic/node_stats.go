package elastic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type nodesStats struct {
	ClusterName string          `json:"cluster_name"`
	Nodes       json.RawMessage `json:"nodes"`
}

type elasticNode struct {
	Name      string
	Host      string
	Processed bool
}

type elasticNodeStat struct {
	Name            string          `json:"name"`
	Timestamp       int             `json:"timestamp"`
	Indices         json.RawMessage `json:"indices"`
	OperatingSystem json.RawMessage `json:"os"`
	FileSystem      json.RawMessage `json:"fs"`
	JVMStats        json.RawMessage `json:"jvm"`
	ProcessStats    json.RawMessage `json:"process"`
	ThreadStats     json.RawMessage `json:"thread_pool"`
}
type nodeSubStat struct {
	Timestamp   int    `json:"timestamp"`
	NodeName    string `json:"node_name"`
	ClusterName string `json:"cluster_name"`
}

type indexStats struct {
	nodeSubStat
	Stats *json.RawMessage `json:"index_stats"`
}

type operatingSystemStats struct {
	nodeSubStat
	Stats *json.RawMessage `json:"os_stats"`
}

type fileSystemStats struct {
	nodeSubStat
	Stats *json.RawMessage `json:"fs_stats"`
}

type jvmStats struct {
	nodeSubStat
	Stats *json.RawMessage `json:"jvm_stats"`
}

type processStats struct {
	nodeSubStat
	Stats *json.RawMessage `json:"process_stats"`
}

type threadStats struct {
	nodeSubStat
	Stats *json.RawMessage `json:"thread_stats"`
}

type clusterHealthStats struct {
	Timestamp int64         `json:"timestamp"`
	Stats     clusterHealth `json:"cluster_stats"`
}

type clusterHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               int     `json:"number_of_nodes"`
	NumberOfDataNodes           int     `json:"number_of_data_nodes"`
	ActivePrimaryShards         int     `json:"active_primary_shards"`
	ActiveShards                int     `json:"active_shards"`
	RelocatingShards            int     `json:"relocating_shards"`
	InitializingShards          int     `json:"initializing_shards"`
	UnassignedShards            int     `json:"unassigned_shards"`
	DelayedUnassignedShards     int     `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        int     `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       int     `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis int     `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float32 `json:"active_shards_percent_as_number"`
}

type clusterHealthMonitor struct {
	ClusterName                      string
	NumberOfNodes                    int
	ClusterState                     string
	LastGoodNodeCountDate            time.Time
	LastGoodClusterStatusDate        time.Time
	LastNodeCountNotificationTime    time.Time
	LastClusterStateNotificationDate time.Time
}

// Tracks the nodes that are being processed in this particular processing run
var nodeList []elasticNode
var clusterHealthTracking []clusterHealthMonitor

// Initialize clusterHealthTracking with the configuration data
func initializeClusterHealthTracking() {
	for _, cluster := range configuration.ElasticClientsFrom {
		var newMonitor clusterHealthMonitor
		newMonitor.ClusterName = cluster.ClusterName
		clusterHealthTracking = append(clusterHealthTracking, newMonitor)
	}
}

// Converts current time into epocmills
func getCurrentTimeInMills() int64 {
	timeinmills := time.Now().UnixNano() / 1000000
	return timeinmills
}

// Gets a list of the nodes that are currently running on the Elasticsearch cluster
func getNodeList(hosts []string) {
	body, err := failoverHTTPRequest(hosts, "GET", "_cat/nodes", nil)
	if err == nil {
		parseNodeList(string(body))
	}
}

// Parses the list of nodes from the _cat/nodes call
func parseNodeList(data string) {
	nodeLines := strings.Split(data, "\n")
	nodeList = []elasticNode{}
	for _, value := range nodeLines {
		nodeCols := strings.Split(strings.Trim(value, " "), " ")

		var node elasticNode
		node.Name = nodeCols[len(nodeCols)-1]
		node.Host = nodeCols[0]
		node.Processed = false
		if node.Name != "" {
			nodeList = append(nodeList, node)
		}
	}
}

// updates nodeList array with whether or not the node has been processed
func setNodeAsProcessed(nodeName string) {
	for i := 0; i < len(nodeList); i++ {
		nodeValue := &nodeList[i]
		if nodeValue.Name == nodeName {
			nodeValue.Processed = true
		}
	}
}

// checks the nodeList array to see if a given node has been processed yet
func hasNodeBeenProcessed(nodeName string) bool {
	hasBeenProcessed := false
	for i := 0; i < len(nodeList); i++ {
		nodeValue := &nodeList[i]
		if nodeValue.Name == nodeName {
			hasBeenProcessed = nodeValue.Processed
		}
	}
	return hasBeenProcessed
}

// Gets the details of node stats for the cluster
// Client only nodes other than the one being queried will not show up in this result set and
// must be processed after in the catchup node processing
func queryNodeStats(hosts []string) {
	body, err := failoverHTTPRequest(hosts, "GET", "_nodes/stats", nil)
	if err == nil {
		processNodeStatsBody(body)
	}
}

// Parse and process the json from the _nodes/stats call
func processNodeStatsBody(body []byte) {
	var nodesStats nodesStats
	json.Unmarshal(body, &nodesStats)
	var dat map[string]json.RawMessage
	json.Unmarshal(nodesStats.Nodes, &dat)
	for _, value := range dat {
		var node elasticNodeStat
		json.Unmarshal(value, &node)

		subStat := nodeSubStat{}
		subStat.Timestamp = node.Timestamp
		subStat.NodeName = node.Name
		subStat.ClusterName = nodesStats.ClusterName

		// Process each individual sub section
		indexStatData(indexStats{subStat, &node.Indices}, "index_stats")
		indexStatData(operatingSystemStats{subStat, &node.OperatingSystem}, "os_stats")
		indexStatData(fileSystemStats{subStat, &node.FileSystem}, "fs_stats")
		indexStatData(jvmStats{subStat, &node.JVMStats}, "jvm_stats")
		indexStatData(processStats{subStat, &node.ProcessStats}, "process_stats")
		indexStatData(threadStats{subStat, &node.ThreadStats}, "thread_stats")

		setNodeAsProcessed(node.Name)
	}
}

// Send the document to Elasticsearch for indexing
func indexStatData(stats interface{}, docType string) {
	indexBytes, _ := json.Marshal(stats)
	indexDocument(string(indexBytes), docType)
}

// query the Elasticsearch cluster for overall cluster health

func queryClusterHealth(hosts []string) {
	body, err := failoverHTTPRequest(hosts, "GET", "_cluster/health", nil)
	if err == nil {
		processClusterHealthBody(body)
	}
}

// Parse and process JSON returned from the _cluster/health call.
func processClusterHealthBody(body []byte) {
	var rawmsg clusterHealth
	json.Unmarshal(body, &rawmsg)

	clusterhealth := clusterHealthStats{getCurrentTimeInMills(), rawmsg}
	indexStatData(clusterhealth, "cluster_stats")
	checkClusterHealth(clusterhealth.Stats)

}

// Query and process client only nodes that were missed due to Elasticsearch
// not returning results for client only nodes different than the one being
// queried directly.
func queryCatchupNodes(hosts []string) {
	for _, nodeValue := range nodeList {
		if !nodeValue.Processed {
			processCatchupNode(nodeValue.Name, hosts)
		}
	}
}

// Process each catchup node
func processCatchupNode(nodeName string, hosts []string) {

	// We have to loop through each host and check for an actual results since the
	// http call will return success, but without data.
	for _, host := range hosts {
		var singleHostAsArray []string
		singleHostAsArray = append(singleHostAsArray, host)
		body, err := failoverHTTPRequest(singleHostAsArray, "GET",
			fmt.Sprintf("_nodes/%v/stats", nodeName), nil)
		if err == nil && !hasNodeBeenProcessed(nodeName) {
			processNodeStatsBody(body)
		}
	}
}

// Indexes the requested document.
func indexDocument(document string, docType string) {

	// Build out the index name with a daily suffix
	var urlBuffer bytes.Buffer
	urlBuffer.WriteString(configuration.IndexPrefix)
	urlBuffer.WriteString(time.Now().Format("-2006.01.02/"))
	urlBuffer.WriteString(docType)

	addToIndexQueue(urlBuffer.String(), document)
}

// we already have this data, so no need to do a separate query to get it
func checkClusterHealth(cluster clusterHealth) {

	var clusterMonitor *clusterHealthMonitor

	for i := 0; i < len(clusterHealthTracking); i++ {
		if clusterHealthTracking[i].ClusterName == cluster.ClusterName {
			clusterMonitor = &clusterHealthTracking[i]
		}
	}

	if configuration.NotifyOnNodeCountChange {
		expectedNodeCount := 0

		for _, clusterConfig := range configuration.ElasticClientsFrom {
			if clusterConfig.ClusterName == cluster.ClusterName {
				expectedNodeCount = clusterConfig.ExpectedNodeCount
			}
		}

		clusterMonitor.NumberOfNodes = cluster.NumberOfNodes

		if expectedNodeCount == cluster.NumberOfNodes {
			clusterMonitor.LastGoodNodeCountDate = time.Now()
		} else {
			// Only notify if the last known state was more than a minute ago.
			if clusterMonitor.LastGoodNodeCountDate.Before(time.Now().Add(time.Minute * -1)) {
				// Only notify once an hour
				if clusterMonitor.LastNodeCountNotificationTime.Before(time.Now().Add(time.Hour * -1)) {

					sendNotification(fmt.Sprintf("Node count changed for %v. Expected %v, found %v",
						cluster.ClusterName, expectedNodeCount, cluster.NumberOfNodes),
						nil, notificationOverrides{})

					clusterMonitor.LastNodeCountNotificationTime = time.Now()
				}
			}
		}
	}

	if cluster.Status == "green" {
		clusterMonitor.LastGoodClusterStatusDate = time.Now()
	} else {
		notify := false
		if configuration.NotifyOnClusterRed &&
			cluster.Status == "red" &&
			clusterMonitor.LastClusterStateNotificationDate.Before(time.Now().Add(time.Hour*-1)) &&
			clusterMonitor.LastGoodClusterStatusDate.Before(time.Now().Add(time.Minute*-1)) {

			notify = true
		}

		if configuration.NotifyOnClusterYellow &&
			cluster.Status == "yellow" &&
			clusterMonitor.LastClusterStateNotificationDate.Before(time.Now().Add(time.Hour*-1)) &&
			clusterMonitor.LastGoodClusterStatusDate.Before(time.Now().Add(time.Minute*-1)) {

			notify = true
		}

		if notify {
			sendNotification(fmt.Sprintf("Cluster state is %v for %v", cluster.Status, cluster.ClusterName),
				nil, notificationOverrides{})
			clusterMonitor.LastClusterStateNotificationDate = time.Now()
		}

	}

}
