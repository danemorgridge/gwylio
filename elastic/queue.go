package elastic

import (
	"bytes"
	"time"
)

var retryQueue chan indexQueueItem

type indexQueueItem struct {
	URL     string
	Payload string
}

func addToIndexQueue(URL string, payload string) {
	queueItem := indexQueueItem{URL, payload}

	select {
	case retryQueue <- queueItem:
	default:
		return
	}
}

func startRetryQueue() {
	retryQueue = make(chan indexQueueItem, 100000)

	go func() {
		for item := range retryQueue {
			success := false
			for !success {

				_, err := failoverHTTPRequest(configuration.ElasticClientsTo, "POST",
					item.URL, bytes.NewBuffer([]byte(item.Payload)))

				if err != nil {
					time.Sleep(30 * time.Second)
				} else {
					success = true
				}
			}
		}
	}()
}
