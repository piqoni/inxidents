package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/r3labs/sse/v2"
	"gopkg.in/yaml.v2"
)

type Service struct {
	Name         string        `yaml:"name"`
	Endpoint     string        `yaml:"endpoint"`
	Frequency    time.Duration `yaml:"frequency"`
	ExpectedCode int           `yaml:"expectedCode"`
	ExpectedBody string        `yaml:"expectedBody"`
	up           bool
	error        error
	ack          bool
}

var webhookSlackURL string = os.Getenv("SLACK_WEBHOOK_URL")

func checkURLResponse(url string) (bool, error) {
	// Send an HTTP GET request to the specified URL
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf(resp.Status)
	}
	return true, nil
}

func sendStream(server *sse.Server, s Service, err error) {
	serviceData := map[string]any{
		"name":         s.Name,
		"endpoint":     s.Endpoint,
		"frequency":    s.Frequency.Seconds(),
		"expectedCode": s.ExpectedCode,
		"expectedBody": s.ExpectedBody,
		"up":           s.up,
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

func sendSlackNotification(message string) {
	if webhookSlackURL == "" {
		return
	}
	// disable notifications while developing with an early return TODO
	// return
	data := fmt.Sprintf(`{"text":"%s"}`, message)
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

//go:embed templates/*
var templatesFS embed.FS

func main() {
	// Read the service.yaml file
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file: %v", err)
	}

	// Create a slice to store the loaded services
	var services []Service

	// Unmarshal the YAML data into the services slice
	if err := yaml.Unmarshal(yamlFile, &services); err != nil {
		sendSlackNotification("❌ Error reading the config.yaml file, inxidents will exit and no services will be monitored. Please correct config.yaml and restart the app.")
		log.Fatalf("Error unmarshaling YAML: %v", err)
	}

	server := sse.New()
	server.CreateStream("messages")

	// Create a new Mux and set the handler
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			// Received Browser Disconnection
			<-r.Context().Done()
			println("The client is disconnected here")
			return
		}()

		server.ServeHTTP(w, r)
	})

	http.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		// Return with 200 OK if the current time seconds are odd
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
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
		go func(s Service) {
			for {
				up, err := checkURLResponse(s.Endpoint)
				s.up = up
				sendStream(server, s, err)
				if err != nil {
					message := fmt.Sprintf("🟥 *<%s|%s>* returning *%s* instead of *%d*", s.Endpoint, s.Name, err, s.ExpectedCode)
					sendSlackNotification(message)
				}

				time.Sleep(s.Frequency)
			}
		}(service)
	}

	http.ListenAndServe("0.0.0.0:8080", nil)
}
