package store

import (
	"sync"

	"redis/internal/models"
)

type TaskStore struct {
    sync.RWMutex
    tasks  map[int]models.Task
    nextID int
}

func NewTaskStore() *TaskStore {
    return &TaskStore{
        tasks:  make(map[int]models.Task),
        nextID: 1,
    }
}

func (ts *TaskStore) Add(task models.Task) int {
    ts.Lock()
    defer ts.Unlock()
    task.ID = ts.nextID
    ts.tasks[task.ID] = task
    ts.nextID++
    return task.ID
}

func (ts *TaskStore) Get(id int) (models.Task, bool) {
    ts.RLock()
    defer ts.RUnlock()
    task, ok := ts.tasks[id]
    return task, ok
}

func (ts *TaskStore) Update(id int, task models.Task) bool {
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

func (ts *TaskStore) GetAll() []models.Task {
    ts.RLock()
    defer ts.RUnlock()
    tasks := make([]models.Task, 0, len(ts.tasks))
    for _, task := range ts.tasks {
        tasks = append(tasks, task)
    }
    return tasks
}