package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
    mux := http.NewServeMux()
    s := NewTaskStore()
    cache := NewRedisCache()
    tasks := TaskResource{
        s:     s,
        cache: cache,
    }

    mux.HandleFunc("GET /tasks", tasks.GetAll)
    mux.HandleFunc("POST /tasks", tasks.CreateOne)
    mux.HandleFunc("GET /tasks/{id}", tasks.GetOne)
    mux.HandleFunc("PUT /tasks/{id}", tasks.UpdateOne)
    mux.HandleFunc("DELETE /tasks/{id}", tasks.DeleteOne)

    if err := http.ListenAndServe(":8080", mux); err != nil {
        fmt.Printf("Failed to listen and serve: %v\n", err)
    }
}

type Task struct {
    ID        int    `json:"id"`
    Title     string `json:"title"`
    Completed bool   `json:"completed"`
}

type TaskStore struct {
    sync.RWMutex
    tasks  map[int]Task
    nextID int
}

func NewTaskStore() *TaskStore {
    return &TaskStore{
        tasks:  make(map[int]Task),
        nextID: 1,
    }
}

func (ts *TaskStore) Add(task Task) int {
    ts.Lock()
    defer ts.Unlock()
    task.ID = ts.nextID
    ts.tasks[task.ID] = task
    ts.nextID++
    return task.ID
}

func (ts *TaskStore) Get(id int) (Task, bool) {
    ts.RLock()
    defer ts.RUnlock()
    task, ok := ts.tasks[id]
    return task, ok
}

func (ts *TaskStore) Update(id int, task Task) bool {
    ts.Lock()
    defer ts.Unlock()
    if _, ok := ts.tasks[id]; !ok {
        return false
    }
    task.ID = id
    ts.tasks[id] = task
    return true
}

func (ts *TaskStore) Delete(id int) bool {
    ts.Lock()
    defer ts.Unlock()
    if _, ok := ts.tasks[id]; !ok {
        return false
    }
    delete(ts.tasks, id)
    return true
}

func (ts *TaskStore) GetAll() []Task {
    ts.RLock()
    defer ts.RUnlock()
    tasks := make([]Task, 0, len(ts.tasks))
    for _, task := range ts.tasks {
        tasks = append(tasks, task)
    }
    return tasks
}

type TaskResource struct {
    s     *TaskStore
    cache *RedisCache
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

    tasks := t.s.GetAll()
    
    tasksJSON, err := json.Marshal(tasks)
    if err == nil {
        err = t.cache.Set(ctx, cacheKey, string(tasksJSON), time.Minute*5) // Кешуємо на 5 хвилин
        if err != nil {
            fmt.Printf("Failed to cache all tasks: %v\n", err)
        }
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(tasks)
}

func (t *TaskResource) CreateOne(w http.ResponseWriter, r *http.Request) {
    var task Task
    err := json.NewDecoder(r.Body).Decode(&task)
    if err != nil {
        fmt.Printf("Failed to decode: %v\n", err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    task.ID = t.s.Add(task)
    
    cacheKey := fmt.Sprintf("task:%d", task.ID)
    taskJSON, _ := json.Marshal(task)
    err = t.cache.Set(r.Context(), cacheKey, string(taskJSON), time.Hour)
    if err != nil {
        fmt.Printf("Failed to cache new task: %v\n", err)
    }

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

    task, ok := t.s.Get(id)
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
    var task Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    if !t.s.Update(id, task) {
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
    if !t.s.Delete(id) {
        http.NotFound(w, r)
        return
    }

    // Видаляємо з кешу
    cacheKey := fmt.Sprintf("task:%d", id)
    err = t.cache.Del(ctx, cacheKey)
    if err != nil {
        fmt.Printf("Failed to delete from cache: %v\n", err)
    }

    w.WriteHeader(http.StatusOK)
}

type RedisCache struct {
    client *redis.Client
}

func NewRedisCache() *RedisCache {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    return &RedisCache{client: client}
}

func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
    return rc.client.Get(ctx, key).Result()
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
    return rc.client.Set(ctx, key, value, expiration).Err()
}

func (rc *RedisCache) Del(ctx context.Context, key string) error {
    return rc.client.Del(ctx, key).Err()
}