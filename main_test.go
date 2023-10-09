package main

import (
	"fmt"
	"github.com/r3labs/sse/v2"
	"net/http"
	"testing"
	"time"
)

// TestCheckURLResponse tests the checkURLResponse function.
func TestCheckURLResponse(t *testing.T) {
	// Start a simple HTTP server for testing purposes
	go func() {
		http.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		http.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		http.ListenAndServe(":8081", nil)
	}()
	time.Sleep(1 * time.Second) // Wait for the server to start

	// Test the function with a URL that returns 200 OK
	urlOK := "http://localhost:8081/ok"
	resultOK, err := checkURLResponse(urlOK)
	if err != nil || !resultOK {
		t.Errorf("Expected checkURLResponse to return true and no error for URL %s, but got %v and %v", urlOK, resultOK, err)
	}

	// Test the function with a URL that returns an error status code
	urlError := "http://localhost:8081/error"
	resultError, err := checkURLResponse(urlError)
	if err == nil || resultError {
		t.Errorf("Expected checkURLResponse to return false and an error for URL %s, but got %v and %v", urlError, resultError, err)
	}
}

// TestSendStream tests the sendStream function.
func TestSendStream(t *testing.T) {
	server := sse.New()
	server.CreateStream("messages")

	testService := Service{
		Name:         "TestService",
		Endpoint:     "http://localhost:8081/test",
		Frequency:    1 * time.Second,
		ExpectedCode: http.StatusOK,
	}

	// Test the sendStream function with a test service and no error
	err := fmt.Errorf("No error")
	sendStream(server, testService, err)

	// Test the sendStream function with a test service and an error
	err = fmt.Errorf("Test error")
	sendStream(server, testService, err)

	// TODO
}
