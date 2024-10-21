package main

// needed libs for API functionality
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

// constants for DB
const userFilePath = "users.json"
const dataFolderPath = "data/"

// user defined types
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Note struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// local variables
var users []User
var currentUser *User
var notes []Note

// Function loading users from JSON file
func loadUsers() error {
	data, err := ioutil.ReadFile(userFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &users)
}

// Function saving users to JSON file
func saveUsers() error {
	data, err := json.Marshal(users)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(userFilePath, data, 0644)
}

// Function loading notes for a user from JSON file
func loadNotes(username string) error {
	filePath := dataFolderPath + username + ".json"
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &notes)
}

// Function saving notes for a user to JSON file
func saveNotes(username string) error {
	filePath := dataFolderPath + username + ".json"
	data, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, data, 0644)
}

// Function registering a new user
func register(username, password string) error {
	for _, user := range users {
		if user.Username == username {
			return errors.New("username already exists")
		}
	}
	newUser := User{Username: username, Password: password}
	users = append(users, newUser)
	return saveUsers()
}

// Function for logging a user
func login(username, password string) error {
	for _, user := range users {
		if user.Username == username && user.Password == password {
			currentUser = &user
			return loadNotes(username)
		}
	}
	return errors.New("invalid username or password")
}

// Function for adding a new note
func addNote() {
	var name, content string
	fmt.Println("Enter note name:")
	fmt.Scanln(&name)

	// Check if a note with the same name already exists
	for _, note := range notes {
		if note.Name == name {
			fmt.Println("Error: A note with this name already exists.")
			return
		}
	}

	fmt.Println("Enter note content:")
	fmt.Scanln(&content)

	note := Note{Name: name, Content: content}
	notes = append(notes, note)
	saveNotes(currentUser.Username)
}

// Function for editing an existing note
func editNote() {
	var name string
	fmt.Println("Enter note name to edit:")
	fmt.Scanln(&name)

	for i, note := range notes {
		if note.Name == name {
			var newContent string
			fmt.Println("Enter new content:")
			fmt.Scanln(&newContent)
			notes[i].Content = newContent
			saveNotes(currentUser.Username)
			fmt.Println("Note updated.")
			return
		}
	}
	fmt.Println("Error: Note not found.")
}

// Function deleting a note
func deleteNote() {
	var name string
	fmt.Println("Enter note name to delete:")
	fmt.Scanln(&name)

	for i, note := range notes {
		if note.Name == name {
			notes = append(notes[:i], notes[i+1:]...)
			saveNotes(currentUser.Username)
			fmt.Println("Note deleted.")
			return
		}
	}
	fmt.Println("Error: Note not found.")
}

// Function printing all notes
func printNotes() {
	if len(notes) == 0 {
		fmt.Println("No notes found.")
		return
	}
	for _, note := range notes {
		fmt.Printf("Name: %s, Content: %s\n", note.Name, note.Content)
	}
}

// Main menu for logged-in users
func loggedInMenu() {
	for {
		fmt.Println("\n1. Add Note")
		fmt.Println("2. Edit Note")
		fmt.Println("3. Delete Note")
		fmt.Println("4. Show Notes")
		fmt.Println("5. Logout")
		fmt.Print("Choose an option: ")
		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			addNote()
		case 2:
			editNote()
		case 3:
			deleteNote()
		case 4:
			printNotes()
		case 5:
			currentUser = nil
			notes = nil
			fmt.Println("Logged out.")
			return
		default:
			fmt.Println("Invalid option.")
		}
	}
}

// Main menu for the program - register/login
func startMenu() {
	err := loadUsers()
	if err != nil {
		fmt.Println("Error loading users:", err)
		return
	}

	for {
		fmt.Println("\n1. Register")
		fmt.Println("2. Login")
		fmt.Println("3. Exit")
		fmt.Print("Choose an option: ")
		var choice int
		fmt.Scanln(&choice)

		switch choice {
		case 1:
			var username, password string
			fmt.Println("Enter username:")
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			fmt.Scanln(&password)
			err := register(username, password)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Registration successful.")
			}
		case 2:
			var username, password string
			fmt.Println("Enter username:")
			fmt.Scanln(&username)
			fmt.Println("Enter password:")
			fmt.Scanln(&password)
			err := login(username, password)
			if err != nil {
				fmt.Println("Error:", err)
			} else {
				fmt.Println("Login successful.")
				loggedInMenu()
			}
		case 3:
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Invalid option.")
		}
	}
}

// main Go func
func main() {
	// Ensure the data directory exists
	if _, err := os.Stat(dataFolderPath); os.IsNotExist(err) {
		os.Mkdir(dataFolderPath, 0755)
	}

	startMenu()

}
