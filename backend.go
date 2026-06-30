package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type Response struct {
	Message   string              `json:"message"`
	Service   string              `json:"service"`
	Method    string              `json:"method"`
	Path      string              `json:"path"`
	Headers   map[string][]string `json:"headers,omitempty"`
	Timestamp string              `json:"timestamp"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	serviceName := getEnv("BACKEND_NAME", "gonk-demo-backend")
	response := Response{
		Message:   "Hello from GONK demo backend",
		Service:   serviceName,
		Method:    r.Method,
		Path:      r.URL.Path,
		Headers:   r.Header,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)

	log.Printf("[%s] %s %s", time.Now().Format("15:04:05"), r.Method, r.URL.Path)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": getEnv("BACKEND_NAME", "gonk-demo-backend"),
		"time":    time.Now().Format(time.RFC3339),
	})
}

func main() {
	port := getEnv("PORT", "3000")

	http.HandleFunc("/", handler)
	http.HandleFunc("/health", healthHandler)

	log.Println("==========================================")
	log.Println("🚀 Backend Server Starting...")
	log.Println("==========================================")
	log.Printf("Service: %s", getEnv("BACKEND_NAME", "gonk-demo-backend"))
	log.Printf("Server: http://localhost:%s", port)
	log.Printf("Health: http://localhost:%s/health", port)
	log.Println("==========================================")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
