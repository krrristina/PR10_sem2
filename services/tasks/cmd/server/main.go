package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type Task struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

func main() {
	instanceID := os.Getenv("INSTANCE_ID")
	if instanceID == "" {
		instanceID = "tasks-unknown"
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8082"
	}

	tasks := []Task{
		{ID: 1, Title: "Изучить NGINX"},
		{ID: 2, Title: "Понять load balancing"},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Instance-ID", instanceID)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":   "ok",
			"instance": instanceID,
		})
	})

	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Instance-ID", instanceID)
		_ = json.NewEncoder(w).Encode(tasks)
	})

	addr := ":" + port
	log.Println("tasks service started on", addr, "instance =", instanceID)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
