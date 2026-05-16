package eureka

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const registrationRetryInterval = 10 * time.Second

type eurekaRegistration struct {
	Instance eurekaInstance `json:"instance"`
}

type eurekaInstance struct {
	HostName       string         `json:"hostName"`
	App            string         `json:"app"`
	IPAddr         string         `json:"ipAddr"`
	Status         string         `json:"status"`
	Port           eurekaPort     `json:"port"`
	DataCenterInfo dataCenterInfo `json:"dataCenterInfo"`
	InstanceID     string         `json:"instanceId"`
	HealthCheckUrl string         `json:"healthCheckUrl"`
	StatusPageUrl  string         `json:"statusPageUrl"`
	HomePageUrl    string         `json:"homePageUrl"`
}

type eurekaPort struct {
	Value   int    `json:"$"`
	Enabled string `json:"@enabled"`
}

type dataCenterInfo struct {
	Class string `json:"@class"`
	Name  string `json:"name"`
}

// Register spawns a background goroutine that retries the Eureka handshake
// every 10 seconds until it succeeds, then starts the 30-second heartbeat
// loop. It returns immediately so a missing or late-starting Eureka server
// never blocks startup or aborts the application.
//
// Close the returned channel to trigger graceful deregistration. The channel
// is always non-nil; closing it before a successful registration cancels the
// retry loop.
func Register(eurekaURL, appName, instanceID, serverPort, appBaseURL string) (chan struct{}, error) {
	parsed, err := url.Parse(eurekaURL)
	if err != nil {
		return nil, fmt.Errorf("parse eureka URL: %w", err)
	}

	var username, password string
	if parsed.User != nil {
		username = parsed.User.Username()
		password, _ = parsed.User.Password()
	}

	baseURL := strings.TrimRight(
		fmt.Sprintf("%s://%s%s", parsed.Scheme, parsed.Host, parsed.Path),
		"/",
	)

	port, _ := strconv.Atoi(serverPort)
	upperApp := strings.ToUpper(appName)

	body := eurekaRegistration{
		Instance: eurekaInstance{
			HostName: "localhost",
			App:      upperApp,
			IPAddr:   "127.0.0.1",
			Status:   "UP",
			Port: eurekaPort{
				Value:   port,
				Enabled: "true",
			},
			DataCenterInfo: dataCenterInfo{
				Class: "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
				Name:  "MyOwn",
			},
			InstanceID:     instanceID,
			HealthCheckUrl: appBaseURL + "/actuator/health",
			StatusPageUrl:  appBaseURL + "/actuator/health",
			HomePageUrl:    appBaseURL,
		},
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal eureka body: %w", err)
	}

	registerURL := fmt.Sprintf("%s/apps/%s", baseURL, upperApp)
	heartbeatURL := fmt.Sprintf("%s/apps/%s/%s", baseURL, upperApp, instanceID)

	stopCh := make(chan struct{})

	go registerWithRetry(registerURL, heartbeatURL, payload, username, password, instanceID, upperApp, stopCh)

	return stopCh, nil
}

// registerWithRetry loops until Eureka accepts the registration or stopCh is
// closed. Once registered, it transitions into the heartbeat loop. It never
// returns an error to the caller — every transient failure is logged and
// retried.
func registerWithRetry(registerURL, heartbeatURL string, payload []byte, username, password, instanceID, upperApp string, stopCh chan struct{}) {
	for {
		err := doRequest(http.MethodPost, registerURL, payload, username, password, http.StatusNoContent)
		if err == nil {
			log.Println("[INFO] Successfully registered with Eureka Service Registry!")
			log.Printf("[eureka] registered as %s (instanceId: %s)", upperApp, instanceID)
			heartbeatLoop(heartbeatURL, username, password, instanceID, stopCh)
			return
		}

		log.Printf("[WARN] Eureka registration failed. Retrying in 10 seconds... (cause: %v)", err)

		select {
		case <-stopCh:
			log.Println("[eureka] retry loop cancelled before successful registration")
			return
		case <-time.After(registrationRetryInterval):
		}
	}
}

func heartbeatLoop(heartbeatURL, username, password, instanceID string, stopCh chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			if err := doRequest(http.MethodDelete, heartbeatURL, nil, username, password, http.StatusOK); err != nil {
				log.Printf("[eureka] deregister error: %v", err)
			} else {
				log.Printf("[eureka] deregistered %s", instanceID)
			}
			return
		case <-ticker.C:
			if err := doRequest(http.MethodPut, heartbeatURL, nil, username, password, http.StatusOK); err != nil {
				log.Printf("[eureka] heartbeat error: %v", err)
			}
		}
	}
}

func doRequest(method, url string, body []byte, username, password string, expectedStatus int) error {
	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	// Eureka returns 204 for register, 200 for heartbeat/deregister
	if resp.StatusCode != expectedStatus && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s %s", resp.StatusCode, method, url)
	}
	return nil
}
