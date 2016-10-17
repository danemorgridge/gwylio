package elastic

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/howeyc/fsnotify"
)

//StartMonitoring Kicks off the main process thread that starts monitoring
func StartMonitoring() {
	// Start a WaitGroup so that the process will run as a service
	var wg sync.WaitGroup
	wg.Add(1)
	loadConfiguration()
	initializeClusterHealthTracking()
	loadNotificationRules()
	setupRulesWatcher()
	startRetryQueue()

	ticker := time.NewTicker(time.Second * time.Duration(configuration.CollectInterval))
	go func() {
		for range ticker.C {
			doMonitor()
		}
	}()

	// Fire it off once before the ticker kicks in
	doMonitor()
	wg.Wait()
}

// Loop through each Elastic cluster and collect stats
func doMonitor() {
	for _, hostCollection := range configuration.ElasticClientsFrom {
		getNodeList(hostCollection.Hosts)
		queryClusterHealth(hostCollection.Hosts)
		queryNodeStats(hostCollection.Hosts)
		queryCatchupNodes(hostCollection.Hosts)
	}

	runNotificationRules()
}

func setupRulesWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// Process events
	go func() {
		for {
			select {
			case <-watcher.Event:
				reloadNotifications = true
			}
		}
	}()

	err = watcher.Watch("rules")
	if err != nil {
		log.Fatal(err)
	}
}

// A utility function to provide http failover
// If one http call fails (with a status other than 200), the next host in the line will be tried
func failoverHTTPRequest(hosts []string, method string, url string, requestBody io.Reader) (body []byte, err error) {
	foundGoodHost := false
	for _, host := range hosts {
		if !foundGoodHost {
			body, err = executeHTTPRequest(host, method, url, requestBody)
			if err == nil && len(body) > 0 {
				foundGoodHost = true
			}
		}
	}

	if !foundGoodHost {
		err = fmt.Errorf("No responding host was found for url: %v ", url)
	}

	return body, err
}

func executeHTTPRequest(host string, method string, url string, requestBody io.Reader) (body []byte, err error) {

	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered in f", r)
		}
	}()

	client := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}

	requestURL := fmt.Sprintf("%v/%v", host, url)
	req, reqErr := http.NewRequest(method, requestURL, requestBody)

	if reqErr != nil {
		log.Print(reqErr)
		err = reqErr
	} else {
		resp, reqErr := client.Do(req)
		if reqErr != nil {
			log.Print(reqErr)
			err = reqErr
		} else {
			responseBody, reqErr := ioutil.ReadAll(resp.Body)
			if reqErr != nil {
				log.Print(reqErr)
				err = reqErr
			}
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				body = responseBody
			} else {
				err = errors.New("response wasn't 200")
			}
		}
		if resp != nil && resp.Body != nil {
			defer resp.Body.Close()
		}
	}

	return body, err
}
