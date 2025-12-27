package main

import (
    "encoding/json"
    "log"
    "net/http"
    "time"
)

type Response struct {
    Message   string              `json:"message"`
    Method    string              `json:"method"`
    Path      string              `json:"path"`
    Headers   map[string][]string `json:"headers,omitempty"`
    Timestamp string              `json:"timestamp"`
}

func handler(w http.ResponseWriter, r *http.Request) {
    response := Response{
        Message:   "Hello from Backend Server!",
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
        "status": "healthy",
        "time":   time.Now().Format(time.RFC3339),
    })
}

func main() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/health", healthHandler)
    
    log.Println("==========================================")
    log.Println("🚀 Backend Server Starting...")
    log.Println("==========================================")
    log.Println("Server: http://localhost:3000")
    log.Println("Health: http://localhost:3000/health")
    log.Println("==========================================")
    
    if err := http.ListenAndServe(":3000", nil); err != nil {
        log.Fatal(err)
    }
}