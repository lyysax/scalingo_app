package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type Task struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Age       int    `json:"age"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
}

var (
	users   = []User{}
	tasks   = map[int][]Task{} // Tasks by user ID
	usersMu sync.Mutex
	tasksMu sync.Mutex
	nextID  = 1
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	})

	initDB()

	http.HandleFunc("/api/tasks", handleTasks)
	http.HandleFunc("/api/tasks/", handleTaskDetail)
	http.HandleFunc("/api/users", handleUsers)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3333"
	}

	log.Printf("Serveur lancé sur le port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

func handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		getTasks(w, r)
	} else if r.Method == http.MethodPost {
		createTask(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		createUser(w, r)
	} else if r.Method == http.MethodGet {
		getUser(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.URL.Path[len("/api/tasks/"):]
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodPut {
		updateTask(w, r, id)
	} else if r.Method == http.MethodDelete {
		deleteTask(w, r, id)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getTasks(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid userId", http.StatusBadRequest)
		return
	}

	tasksMu.Lock()
	defer tasksMu.Unlock()

	userTasks := tasks[id]
	if userTasks == nil {
		userTasks = []Task{}
	}
	json.NewEncoder(w).Encode(userTasks)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid userId", http.StatusBadRequest)
		return
	}

	var task Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tasksMu.Lock()
	task.ID = nextID
	nextID++
	tasks[id] = append(tasks[id], task)
	tasksMu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(task)
}

func updateTask(w http.ResponseWriter, r *http.Request, idStr string) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	uid, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid userId", http.StatusBadRequest)
		return
	}

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

	tasksMu.Lock()
	defer tasksMu.Unlock()

	for i, t := range tasks[uid] {
		if t.ID == id {
			tasks[uid][i].Title = task.Title
			tasks[uid][i].Completed = task.Completed
			json.NewEncoder(w).Encode(tasks[uid][i])
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

func deleteTask(w http.ResponseWriter, r *http.Request, idStr string) {
	userID := r.URL.Query().Get("userId")
	if userID == "" {
		http.Error(w, "userId required", http.StatusBadRequest)
		return
	}

	uid, err := strconv.Atoi(userID)
	if err != nil {
		http.Error(w, "Invalid userId", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	tasksMu.Lock()
	defer tasksMu.Unlock()

	for i, t := range tasks[uid] {
		if t.ID == id {
			tasks[uid] = append(tasks[uid][:i], tasks[uid][i+1:]...)
			json.NewEncoder(w).Encode(map[string]string{"message": "Task deleted"})
			return
		}
	}

	http.Error(w, "Task not found", http.StatusNotFound)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	usersMu.Lock()
	user.ID = nextID
	nextID++
	users = append(users, user)
	tasks[user.ID] = []Task{}
	usersMu.Unlock()

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		http.Error(w, "email required", http.StatusBadRequest)
		return
	}

	usersMu.Lock()
	defer usersMu.Unlock()

	for _, user := range users {
		if user.Email == email {
			json.NewEncoder(w).Encode(user)
			return
		}
	}

	http.Error(w, "User not found", http.StatusNotFound)
}
