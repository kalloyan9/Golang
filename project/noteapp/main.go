package main

// imports
import (
    "crypto/sha256"
    "encoding/json"
    "errors"
    "fmt"
    "html/template"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

// consts used for fstream project structure
const dataFolderPath = "data/"
const userFilePath = "data/users.json"

// defined types
type User struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

type Note struct {
    Name    string `json:"name"`
    Content string `json:"content"`
}

// global variables
var currentUser *User

// Helper function for reading files
func readFile(filePath string, v interface{}) error {
    data, err := os.ReadFile(filePath)
    if err != nil {
        if errors.Is(err, os.ErrNotExist) {
            return nil
        }
        return fmt.Errorf("failed to read file %s: %w", filePath, err)
    }
    if err := json.Unmarshal(data, v); err != nil {
        return fmt.Errorf("failed to unmarshal file %s: %w", filePath, err)
    }
    return nil
}

// Helper function for writing files
func writeFile(filePath string, v interface{}) error {
    data, err := json.Marshal(v)
    if err != nil {
        return fmt.Errorf("failed to marshal data for file %s: %w", filePath, err)
    }
    return os.WriteFile(filePath, data, 0644)
}

// Helper function to respond with an error message in JSON
func respondWithError(w http.ResponseWriter, message string, status int) {
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}


// Load users from JSON
func loadUsers() ([]User, error) {
    var users []User
    err := readFile(userFilePath, &users)
    return users, err
}

// Save users to JSON
func saveUsers(users []User) error {
    return writeFile(userFilePath, users)
}

// Load notes for a user
func loadNotes(username string) ([]Note, error) {
    filePath := filepath.Join(dataFolderPath, username+".json")
    var notes []Note
    err := readFile(filePath, &notes)
    return notes, err
}

// Save notes for a user
func saveNotes(username string, notes []Note) error {
    filePath := filepath.Join(dataFolderPath, username+".json")
    return writeFile(filePath, notes)
}

// Hash password using SHA-256
func hashPassword(password string) string {
    hash := sha256.Sum256([]byte(password))
    return fmt.Sprintf("%x", hash)
}

// Register user
func registerHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        username := r.FormValue("username")
        password := r.FormValue("password")

        users, err := loadUsers()
        if err != nil {
            respondWithError(w, "Error loading users", http.StatusInternalServerError)
            return
        }

        for _, user := range users {
            if user.Username == username {
                respondWithError(w, "Username already exists", http.StatusBadRequest)
                return
            }
        }

        hashedPassword := hashPassword(password)
        newUser := User{Username: username, Password: hashedPassword}
        users = append(users, newUser)

        if err := saveUsers(users); err != nil {
            respondWithError(w, "Error saving user", http.StatusInternalServerError)
            return
        }

        http.Redirect(w, r, "/", http.StatusSeeOther)
    } else {
        tmpl, err := template.ParseFiles("templates/register.html")
        if err != nil {
            respondWithError(w, "Error loading template", http.StatusInternalServerError)
            return
        }
        tmpl.Execute(w, nil)
    }
}

// Login user
func loginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodPost {
        r.ParseForm()
        username := r.FormValue("username")
        password := hashPassword(r.FormValue("password"))

        users, err := loadUsers()
        if err != nil {
            respondWithError(w, "Error loading users", http.StatusInternalServerError)
            return
        }

        for _, user := range users {
            if user.Username == username && user.Password == password {
                currentUser = &user
                http.Redirect(w, r, "/notes", http.StatusSeeOther)
                return
            }
        }

        respondWithError(w, "Invalid username or password", http.StatusUnauthorized)
    } else {
        tmpl, err := template.ParseFiles("templates/login.html")
        if err != nil {
            respondWithError(w, "Error loading template", http.StatusInternalServerError)
            return
        }
        tmpl.Execute(w, nil)
    }
}

// Logout user
func logoutHandler(w http.ResponseWriter, r *http.Request) {
    currentUser = nil // Clear the current user session

    // Clear any session cookies
    http.SetCookie(w, &http.Cookie{
        Name:    "session", // Name of the cookie
        Value:   "",        // Clear the value
        Expires: time.Now().Add(-1 * time.Hour), // Set the expiration to the past
        Path:    "/",      // Specify the path
    })

    http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Notes page handler
func notesHandler(w http.ResponseWriter, r *http.Request) {
    if currentUser == nil {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    // Add cache control headers to prevent caching
    w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
    w.Header().Set("Pragma", "no-cache")
    w.Header().Set("Expires", "0")

    notes, err := loadNotes(currentUser.Username)
    if err != nil {
        respondWithError(w, "Error loading notes", http.StatusInternalServerError)
        return
    }

    if r.Method == http.MethodPost {
        r.ParseForm()
        name := r.FormValue("name")
        content := r.FormValue("content")

        // Check if note name already exists
        for _, note := range notes {
            if note.Name == name {
                respondWithError(w, "Note with this name already exists. Please choose a unique name.", http.StatusBadRequest)
                return
            }
        }

        note := Note{Name: name, Content: content}
        notes = append(notes, note)

        if err := saveNotes(currentUser.Username, notes); err != nil {
            respondWithError(w, "Error saving notes", http.StatusInternalServerError)
            return
        }

        http.Redirect(w, r, "/notes", http.StatusSeeOther)
        return
    }

    tmpl, err := template.ParseFiles("templates/notes.html")
    if err != nil {
        respondWithError(w, "Error loading template", http.StatusInternalServerError)
        return
    }
    tmpl.Execute(w, notes)
}


// Edit note handler
func editNoteHandler(w http.ResponseWriter, r *http.Request) {
    notes, err := loadNotes(currentUser.Username)
    if err != nil {
        respondWithError(w, "Error loading notes", http.StatusInternalServerError)
        return
    }

    if r.Method == http.MethodPost {
        r.ParseForm()
        oldName := r.FormValue("old_name")
        newName := r.FormValue("name")
        newContent := r.FormValue("content")

        // Update the note
        for i, note := range notes {
            if note.Name == oldName {
                notes[i].Name = newName
                notes[i].Content = newContent
                break
            }
        }

        if err := saveNotes(currentUser.Username, notes); err != nil {
            respondWithError(w, "Error saving notes", http.StatusInternalServerError)
            return
        }

        http.Redirect(w, r, "/notes", http.StatusSeeOther)
        return
    }

    noteName := r.URL.Query().Get("name")
    var noteToEdit Note
    for _, note := range notes {
        if note.Name == noteName {
            noteToEdit = note
            break
        }
    }

    tmpl, err := template.ParseFiles("templates/edit_note.html")
    if err != nil {
        respondWithError(w, "Error loading template", http.StatusInternalServerError)
        return
    }
    tmpl.Execute(w, noteToEdit)
}

// Delete note handler
func deleteNoteHandler(w http.ResponseWriter, r *http.Request) {
    if currentUser == nil {
        http.Redirect(w, r, "/", http.StatusSeeOther)
        return
    }

    notes, err := loadNotes(currentUser.Username)
    if err != nil {
        respondWithError(w, "Error loading notes", http.StatusInternalServerError)
        return
    }

    noteName := r.URL.Query().Get("name")
    for i, note := range notes {
        if note.Name == noteName {
            notes = append(notes[:i], notes[i+1:]...) // Remove the note
            break
        }
    }

    if err := saveNotes(currentUser.Username, notes); err != nil {
        respondWithError(w, "Error saving notes", http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

// Static file handler
func staticFileHandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "static/"+r.URL.Path[1:])
}

// Main handler function with error handling
func main() {
    // Ensure the data folder exists
    if err := os.MkdirAll(dataFolderPath, 0755); err != nil {
        fmt.Println("Error creating data folder:", err)
        return
    }

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        tmpl, err := template.ParseFiles("templates/index.html")
        if err != nil {
            respondWithError(w, "Error loading template", http.StatusInternalServerError)
            return
        }
        if err := tmpl.Execute(w, nil); err != nil {
            respondWithError(w, "Error executing template", http.StatusInternalServerError)
            fmt.Println("Template execution error:", err)
        }
    })

    http.HandleFunc("/register", registerHandler)
    http.HandleFunc("/login", loginHandler)
    http.HandleFunc("/notes", notesHandler)
    http.HandleFunc("/edit", editNoteHandler)
    http.HandleFunc("/delete", deleteNoteHandler)
    http.HandleFunc("/logout", logoutHandler)
    http.HandleFunc("/static/", staticFileHandler)

    fmt.Println("Server started at http://localhost:8080")
    if err := http.ListenAndServe(":8080", nil); err != nil {
        fmt.Println("Server failed:", err)
    }
}
