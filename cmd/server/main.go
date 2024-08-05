package main

import (
	"fmt"
	"net/http"

	"redis/internal/api"
	"redis/internal/cache"
	"redis/internal/store"
)

func main() {
    mux := http.NewServeMux()
    s := store.NewTaskStore()
    cache := cache.NewRedisCache()
    handlers := api.NewTaskResource(s, cache)

    mux.HandleFunc("GET /tasks", handlers.GetAll)
    mux.HandleFunc("POST /tasks", handlers.CreateOne)
    mux.HandleFunc("GET /tasks/{id}", handlers.GetOne)
    mux.HandleFunc("PUT /tasks/{id}", handlers.UpdateOne)
    mux.HandleFunc("DELETE /tasks/{id}", handlers.DeleteOne)

    if err := http.ListenAndServe(":8080", mux); err != nil {
        fmt.Printf("Failed to listen and serve: %v\n", err)
    }
}