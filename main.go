package main

import (
	"bytes"
	"embed"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/r3labs/sse/v2"
	"gopkg.in/yaml.v2"
)

// func newService(url string, frequency time.Duration) *Service {
// 	return &Service{
// 		URL:       url,
// 		frequency: frequency,
// 	}
// }

func checkURLResponse(url string) (bool, error) {
	// Send an HTTP GET request to the specified URL
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// println response statuc code
	fmt.Println(resp.Status)

	// Check the HTTP status code
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf(resp.Status)
	}
	return true, nil
}

func sendStream(server *sse.Server, up bool, err error) {

	// create message containing the error message
	text := ""
	if up {
		text = "<div class=\"up\">ServiceName</div>"
	} else {
		// text = "<div class=\"down\">ServiceName</div>"
		text = fmt.Sprintf("<div class=\"down\">%s</div>", err)
	}
	server.Publish("messages", &sse.Event{
		Data: []byte(text),
	})
}

func sendSlackNotification(server *sse.Server, message string, webhookURL string, up bool) {
	// get current time and format it as string
	return
	data := fmt.Sprintf(`{"text":"%s"}`, message)

	// Create a POST request with the JSON data
	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer([]byte(data)))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return
	}

	// Set the Content-Type header to application/json
	req.Header.Set("Content-type", "application/json")

	// Perform the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending Slack notification:", err)
		return
	}
	defer resp.Body.Close()
	// Check the response status
	fmt.Println(" sent successfully")
	if resp.Status == "200 OK" {
		fmt.Println("Slack notification sent successfully")
	} else {
		fmt.Println("Error sending Slack notification. Status:", resp.Status)
	}
}

//go:embed templates
var indexHTML embed.FS

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
		log.Fatalf("Error unmarshaling YAML: %v", err)
	}

	// Print the loaded services
	for _, service := range services {
		fmt.Printf("Name: %s\n", service.Name)
		fmt.Printf("Endpoint: %s\n", service.Endpoint)
		fmt.Printf("Frequency: %s\n", service.Frequency)
		fmt.Printf("ExpectedCode: %d\n", service.ExpectedCode)
		fmt.Printf("ExpectedBody: %s\n", service.ExpectedBody)
		fmt.Println()
	}

	// read from os environment variables for slackurl webhook
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	message := "Hello, World!"

	server := sse.New()
	server.CreateStream("messages")

	// Create a new Mux and set the handler
	// mux := http.NewServeMux()
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		go func() {
			// Received Browser Disconnection
			<-r.Context().Done()
			println("The client is disconnected here")
			return
		}()

		server.ServeHTTP(w, r)
	})

	// add an /up endpoint that returns a 200 OK
	http.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		// Return with 200 OK if the current time seconds are odd
		if time.Now().Unix()%2 == 1 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		// w.WriteHeader(http.StatusOK)

	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Serve the index.html file
		http.ServeFile(w, r, "templates/index.html")
	})

	// Start a goroutine that sends a Slack notification every minute
	go func() {
		for {
			up, err := checkURLResponse("http://localhost:8080/up")
			sendStream(server, up, err)
			if err != nil {
				sendSlackNotification(server, message, webhookURL, up)
			}

			time.Sleep(1 * time.Second)
		}
	}()

	http.ListenAndServe(":8080", nil)

	// Keep the main program running
	// fmt.Println("Press Ctrl+C to exit")
	// select {}
}
