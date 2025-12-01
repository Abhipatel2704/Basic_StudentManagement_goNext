package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	// Driver for PostgreSQL
	_ "github.com/lib/pq"
)

// The Go struct that represents a Student record.
type Student struct {
	ID             int    `json:"id"`         // Unique ID from the database
	FirstName      string `json:"first_name"` // Maps to 'first_name' column
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	EnrollmentDate string `json:"enrollment_date"` // Date the student was added
}

var db *sql.DB // Global variable to hold the database connection

func init() {
	// 1. Database Connection Setup
	connStr := "user=postgres password=ljeng dbname=Student_management_system host=localhost sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err) // Stops the program if connection configuration fails
	}
	if err = db.Ping(); err != nil {
		log.Fatal(err) // Stops the program if the DB server is unreachable
	}
	fmt.Println("Successfully connected to PostgreSQL!")
	createTable()
}

// createTable checks and creates the students table if it doesn't exist.
func createTable() {
	const tableCreationQuery = `
	CREATE TABLE IF NOT EXISTS students (
		id SERIAL PRIMARY KEY,
		first_name VARCHAR(100) NOT NULL,
		last_name VARCHAR(100) NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		enrollment_date DATE DEFAULT CURRENT_DATE
	);`
	if _, err := db.Exec(tableCreationQuery); err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

func main() {
	// 2. Routing Setup
	http.HandleFunc("/api/students", studentsHandler)     // Handles /api/students (GET all, POST new)
	http.HandleFunc("/api/students/", studentByIDHandler) // Handles /api/students/{id} (GET one, PUT, DELETE)

	fmt.Println("Server listening on port 8080...")
	// Start the server
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handler function for requests without an ID (GET all, POST new)
func studentsHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for the frontend (Next.js) to access this API
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case "GET":
		getAllStudents(w, r)
	case "POST":
		createStudent(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler function for requests with an ID (GET one, PUT, DELETE)
func studentByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Simple way to extract ID from URL path (e.g., "/api/students/101" -> "101")
	path := r.URL.Path
	idStr := path[len("/api/students/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid student ID format", http.StatusBadRequest)
		return
	}

	switch r.Method {
	// The rest of the switch logic remains the same, calling getStudentByID, updateStudent, or deleteStudent
	case "GET":
		getStudentByID(w, r, id)
	case "PUT":
		updateStudent(w, r, id)
	case "DELETE":
		deleteStudent(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- CRUD Functions ---

// READ: Fetch all students
func getAllStudents(w http.ResponseWriter, r *http.Request) {
	// db.Query sends the SQL and returns rows
	rows, err := db.Query("SELECT id, first_name, last_name, email, enrollment_date FROM students ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close() // ALWAYS close rows to release database resources

	students := []Student{}
	for rows.Next() {
		var s Student
		// rows.Scan reads column values into the struct fields
		err := rows.Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.EnrollmentDate)
		if err != nil {
			log.Println("Error scanning student:", err)
			continue
		}
		students = append(students, s)
	}

	// json.NewEncoder converts the Go slice (array) of structs into a JSON array
	json.NewEncoder(w).Encode(students)
}

// CREATE: Add a new student
func createStudent(w http.ResponseWriter, r *http.Request) {
	var s Student
	// json.NewDecoder reads the JSON body from the Next.js request and maps fields to the Student struct
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		
		return
	}

	// $1, $2, $3 are placeholders for safe SQL injection prevention.
	// RETURNING id, enrollment_date sends the generated values back immediately.
	sqlStatement := `INSERT INTO students (first_name, last_name, email) VALUES ($1, $2, $3) RETURNING id, enrollment_date`

	var newID int
	var enrollmentDate string

	// db.QueryRow executes the statement and only expects one row back (the RETURNING values)
	err := db.QueryRow(sqlStatement, s.FirstName, s.LastName, s.Email).Scan(&newID, &enrollmentDate)

	if err != nil {
		// This handles database errors, like a unique constraint violation on the email field.
		http.Error(w, fmt.Sprintf("Error creating student: %v", err), http.StatusInternalServerError)
		fmt.Print(2)

		return
	}

	s.ID = newID
	s.EnrollmentDate = enrollmentDate
	w.WriteHeader(http.StatusCreated) // HTTP 201 Created status
	json.NewEncoder(w).Encode(s)
}

// DELETE: Remove a student
func deleteStudent(w http.ResponseWriter, r *http.Request, id int) {
	// db.Exec executes a command that doesn't return rows (like DELETE)
	result, err := db.Exec("DELETE FROM students WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent) // HTTP 204 No Content for a successful deletion
}

// (The updateStudent and getStudentByID functions are similar structural patterns to the above,
// using UPDATE and SELECT with WHERE clauses respectively.)
func getStudentByID(w http.ResponseWriter, r *http.Request, id int) {
	var s Student
	err := db.QueryRow("SELECT id, first_name, last_name, email, enrollment_date FROM students WHERE id = $1", id).
		Scan(&s.ID, &s.FirstName, &s.LastName, &s.Email, &s.EnrollmentDate)

	if err == sql.ErrNoRows {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(s)
}

func updateStudent(w http.ResponseWriter, r *http.Request, id int) {
	var s Student
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := db.Exec("UPDATE students SET first_name = $1, last_name = $2, email = $3 WHERE id = $4",
		s.FirstName, s.LastName, s.Email, id)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	s.ID = id
	json.NewEncoder(w).Encode(s)
}
