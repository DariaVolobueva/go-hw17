package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"redis/internal/cache"
	"redis/internal/models"
	"redis/internal/store"
)

type TaskResource struct {
    store *store.TaskStore
    cache *cache.RedisCache
}

func NewTaskResource(s *store.TaskStore, c *cache.RedisCache) *TaskResource {
    return &TaskResource{
        store: s,
        cache: c,
    }
}

func (t *TaskResource) GetAll(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    cacheKey := "all_tasks"
    
    cachedTasks, err := t.cache.Get(ctx, cacheKey)
    if err == nil {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(cachedTasks))
        return
    }

    tasks := t.store.GetAll()
    
    tasksJSON, err := json.Marshal(tasks)
    if err == nil {
        err = t.cache.Set(ctx, cacheKey, string(tasksJSON), time.Minute*5) // Cache for 5 minutes
        if err != nil {
            fmt.Printf("Failed to cache all tasks: %v\n", err)
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

func (t *TaskResource) CreateOne(w http.ResponseWriter, r *http.Request) {
    var task models.Task
    err := json.NewDecoder(r.Body).Decode(&task)
    if err != nil {
        fmt.Printf("Failed to decode: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    task.ID = t.store.Add(task)
    
    err = json.NewEncoder(w).Encode(task)
    if err != nil {
        fmt.Printf("Failed to encode: %v\n", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}

func (t *TaskResource) GetOne(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) != 3 {
        http.Error(w, "Invalid URL", http.StatusBadRequest)
        return
    }
    idStr := parts[2]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }

    cacheKey := fmt.Sprintf("task:%d", id)
    cachedTask, err := t.cache.Get(ctx, cacheKey)
    if err == nil {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(cachedTask))
        return
    }

    task, ok := t.store.Get(id)
    if !ok {
        http.NotFound(w, r)
        return
    }

    taskJSON, _ := json.Marshal(task)
    err = t.cache.Set(ctx, cacheKey, string(taskJSON), time.Hour)
    if err != nil {
        fmt.Printf("Failed to set cache: %v\n", err)
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write(taskJSON)
}

func (t *TaskResource) UpdateOne(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) != 3 {
        http.Error(w, "Invalid URL", http.StatusBadRequest)
        return
    }
    idStr := parts[2]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }
    var task models.Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if !t.store.Update(id, task) {
        http.NotFound(w, r)
        return
    }

    cacheKey := fmt.Sprintf("task:%d", id)
    err = t.cache.Del(ctx, cacheKey)
    if err != nil {
        fmt.Printf("Failed to delete from cache: %v\n", err)
    }

    w.WriteHeader(http.StatusOK)
}

func (t *TaskResource) DeleteOne(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) != 3 {
        http.Error(w, "Invalid URL", http.StatusBadRequest)
        return
    }
    idStr := parts[2]
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid ID", http.StatusBadRequest)
        return
    }
    if !t.store.Delete(id) {
        http.NotFound(w, r)
        return
    }

    cacheKey := fmt.Sprintf("task:%d", id)
    err = t.cache.Del(ctx, cacheKey)
    if err != nil {
        fmt.Printf("Failed to delete from cache: %v\n", err)
    }

    w.WriteHeader(http.StatusOK)
}