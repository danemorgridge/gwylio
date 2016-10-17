package elastic

import "testing"

func TestParseNodeList(t *testing.T) {

	exepectedNodeCount := 3

	nodeListResponse := `10.0.0.1 10.0.0.1           18          97 0.07 d         *      node-1 
10.0.0.2 10.0.0.2           49          97 0.24 d         m      node-2 
10.0.0.3 10.0.0.3           18          93 0.39 d         m      node-3`

	parseNodeList(nodeListResponse)

	nodeCount := len(nodeList)
	if nodeCount != exepectedNodeCount {
		t.Fail()
		t.Logf("Node Count is incorrect. Should be %v, was %v", exepectedNodeCount, nodeCount)
	}

	if nodeList[0].Host != "10.0.0.1" {
		t.Fail()
		t.Logf("nodeList[0].Host is incorrect. Should be %v, was %v", "10.0.0.1", nodeList[0].Host)
	}

	if nodeList[0].Name != "node-1" {
		t.Fail()
		t.Logf("nodeList[0].Host is incorrect. Should be %v, was %v", "node-1", nodeList[0].Name)
	}

	if nodeList[0].Processed {
		t.Fail()
		t.Logf("nodeList[0].Processed should be false")
	}

	if nodeList[1].Host != "10.0.0.2" {
		t.Fail()
		t.Logf("nodeList[1].Host is incorrect. Should be %v, was %v", "10.0.0.2", nodeList[1].Host)
	}

	if nodeList[1].Name != "node-2" {
		t.Fail()
		t.Logf("nodeList[1].Host is incorrect. Should be %v, was %v", "node-2", nodeList[1].Name)
	}

	if nodeList[1].Processed {
		t.Fail()
		t.Logf("nodeList[1].Processed should be false")
	}

	if nodeList[2].Host != "10.0.0.3" {
		t.Fail()
		t.Logf("nodeList[2].Host is incorrect. Should be %v, was %v", "10.0.0.3", nodeList[2].Host)
	}

	if nodeList[2].Name != "node-3" {
		t.Fail()
		t.Logf("nodeList[2].Host is incorrect. Should be %v, was %v", "node-3", nodeList[2].Name)
	}

	if nodeList[2].Processed {
		t.Fail()
		t.Logf("nodeList[2].Processed should be false")
	}
}

func TestSetNodeAsProcessed(t *testing.T) {
	nodeListResponse := `10.0.0.1 10.0.0.1           18          97 0.07 d         *      node-1 
10.0.0.2 10.0.0.2           49          97 0.24 d         m      node-2 
10.0.0.3 10.0.0.3           18          93 0.39 d         m      node-3`

	parseNodeList(nodeListResponse)

	setNodeAsProcessed("node-1")

	if !nodeList[0].Processed {
		t.Fail()
		t.Logf("nodeList[0].Processed should be true")
	}

	if nodeList[1].Processed {
		t.Fail()
		t.Logf("nodeList[1].Processed should be false")
	}
}

func TestHasNodeBeenProcessed(t *testing.T) {
	nodeListResponse := `10.0.0.1 10.0.0.1           18          97 0.07 d         *      node-1 
10.0.0.2 10.0.0.2           49          97 0.24 d         m      node-2 
10.0.0.3 10.0.0.3           18          93 0.39 d         m      node-3`

	parseNodeList(nodeListResponse)

	setNodeAsProcessed("node-2")

	if hasNodeBeenProcessed("node-1") {
		t.Fail()
		t.Logf("node-1 processed flag should be false")
	}

	if !hasNodeBeenProcessed("node-2") {
		t.Fail()
		t.Logf("node-2 processed flag should be true")
	}
}
