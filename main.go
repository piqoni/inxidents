package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/r3labs/sse/v2"
	"gopkg.in/yaml.v2"
)

type Service struct {
	Name           string        `yaml:"name" json:"name"`
	Endpoint       string        `yaml:"endpoint"`
	Frequency      time.Duration `yaml:"frequency"`
	ExpectedCode   int           `yaml:"expectedCode"`
	ContainsString string        `yaml:"containsString"`
	HttpMethod     string        `yaml:"httpMethod"`
	DisableAlerts  bool          `yaml:"disableAlerts"`
	UserAgent      string        `yaml:"userAgent"`
	up             *bool
	ack            bool
}

var webhookSlackURL string = os.Getenv("SLACK_WEBHOOK_URL")
var dashboardEndpoint string = os.Getenv("DASHBOARD_ENDPOINT")

var services []*Service

//go:embed templates/* static/*
var templatesFS embed.FS

func checkService(s Service) (bool, error) {
	var req *http.Request
	var err error
	// Create an HTTP request based on the specified method
	switch s.HttpMethod {
	case "GET":
		req, err = http.NewRequest("GET", s.Endpoint, nil)
	case "POST":
		req, err = http.NewRequest("POST", s.Endpoint, nil)
	default:
		// Default to GET if the method is not specified or invalid
		req, err = http.NewRequest("GET", s.Endpoint, nil)
	}

	if err != nil {
		return false, err
	}

	// Set the User-Agent header if specified
	if s.UserAgent != "" {
		req.Header.Set("User-Agent", s.UserAgent)
	}

	// Send the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// If there is no ExpectedCode, default 200
	if s.ExpectedCode == 0 {
		s.ExpectedCode = 200
	}

	// Check the HTTP status code
	if resp.StatusCode != s.ExpectedCode {
		return false, fmt.Errorf(resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Check if the specified string exists in the response content
	if s.ContainsString != "" {
		if !bytes.Contains(body, []byte(s.ContainsString)) {
			return false, fmt.Errorf("response does not contain: %s", s.ContainsString)
		}
	}

	return true, nil
}

func sendStream(server *sse.Server, s Service, err error) {
	serviceData := map[string]any{
		"name":         s.Name,
		"endpoint":     s.Endpoint,
		"frequency":    s.Frequency.Seconds(),
		"expectedCode": s.ExpectedCode,
		"up":           s.up,
		"ack":          s.ack,
		"error":        "",
	}

	// If there's an error, set the error field in the map
	if err != nil {
		serviceData["error"] = err.Error()
	}

	// Serialize the map to JSON
	jsonData, err := json.Marshal(serviceData)
	if err != nil {
		// Handle the error
		slog.Error("Error marshaling JSON: %v\n", err)
		return
	}

	// Publish the JSON data as an SSE event
	server.Publish("messages", &sse.Event{
		Data: jsonData,
	})
}

func handleNotification(s *Service, up bool, err error) {
	if s.DisableAlerts {
		s.up = &up
		return
	}

	// Recovering Alert
	if up && s.up != nil && !*s.up {
		sendSlackNotification(fmt.Sprintf("🟩 *<%s|%s>* returning *%v*", s.Endpoint, s.Name, s.ExpectedCode))
		s.ack = false
	}
	s.up = &up // update s.up so its used for the recovering alert on next run in case is false

	// Down Alert
	if err != nil && !s.ack {
		sendSlackNotification(fmt.Sprintf("🟥 *<%s|%s>* returning *%s*", s.Endpoint, s.Name, err.Error()))
	}
}

func sendSlackNotification(message string) {
	if webhookSlackURL == "" {
		return
	}
	// disable notifications while developing with an early return TODO
	// return
	message = strconv.Quote(message)
	data := fmt.Sprintf(`{"text":%s}`, message)
	// Create a POST request with the JSON data
	req, err := http.NewRequest("POST", webhookSlackURL, bytes.NewBuffer([]byte(data)))
	if err != nil {
		slog.Error("Error creating HTTP request:", err)
		return
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-type", "application/json")

	// Perform the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Error sending Slack notification: %v", err)
		return
	}
	defer resp.Body.Close()
	// Check the response status
	if resp.Status == "200 OK" {
		slog.Info("Slack notification sent successfully")
	}
}

func updateAckStatus(services []*Service, serviceName string, ack bool) {
	for _, service := range services {
		if service.Name == serviceName {
			service.ack = ack
			fmt.Println("ack status updated for service:", serviceName)
			break
		}
	}
}

func main() {
	// Read the service.yaml file
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	// Unmarshal the YAML data into the services slice
	if err := yaml.Unmarshal(yamlFile, &services); err != nil {
		sendSlackNotification("❌ Error reading the config.yaml file, inxidents will exit and no services will be monitored. Please correct config.yaml and restart the app.")
		log.Fatalf("Error unmarshaling YAML: %v", err)
	}

	server := sse.New()
	server.AutoReplay = false
	server.CreateStream("messages")

	// Create a new Mux and set the handler
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		server.ServeHTTP(w, r)
	})

	http.HandleFunc("/ack", func(w http.ResponseWriter, r *http.Request) {
		// Check if the request method is POST
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Decode the JSON payload
		var requestBody Service
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&requestBody)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		updateAckStatus(services, requestBody.Name, true)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Request received successfully"}`)
	})

	http.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// add an /unstable endpoint that returns 200 and 503 - for testing
	http.HandleFunc("/unstable", func(w http.ResponseWriter, r *http.Request) {
		// Return with 200 OK if the current time seconds are odd
		if time.Now().Unix()%2 == 1 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	http.HandleFunc("/"+dashboardEndpoint, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && dashboardEndpoint == "" {
			http.NotFound(w, r)
			return
		}

		// Serve the index.html file from the embedded file system
		content, err := templatesFS.ReadFile("templates/index.html")
		if err != nil {
			http.Error(w, "Unable to read index.html", http.StatusInternalServerError)
			return
		}

		w.Write(content)
	})

	for _, service := range services {
		go func(s *Service) {
			for {
				up, err := checkService(*s)
				handleNotification(s, up, err)
				sendStream(server, *s, err)
				time.Sleep(s.Frequency)
			}
		}(service)
	}

	http.Handle("/static/", http.FileServer(http.FS(templatesFS)))
	http.ListenAndServe("0.0.0.0:8080", nil)

}
