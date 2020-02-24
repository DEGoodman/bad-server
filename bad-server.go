package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Users struct {
	Users []User `json:"users"`
}

type User struct {
	UUID     string `json:"guid"` // github.com/google/uuid is more than we need now
	IsActive bool   `json:"isActive"`
	Age      uint8  `json:"age"`
	EyeColor string `json:"eyeColor"`
	Name     Name   `json:"name"`
	Company  string `json:"company"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	About    string `json:"about"`
}

type Name struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

// returns a pointer to the struct of all users
func loadUsers() *Users {
	jsonFile, err := os.Open("users.json")
	if err != nil {
		log.Fatal(err)
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)
	var users Users
	json.Unmarshal(byteValue, &users)

	return &users

}

func rootHandler(w http.ResponseWriter, r *http.Request) {

	userData := loadUsers()
	fmt.Printf("%+v", userData)

	query := r.URL.Query()
	fmt.Printf("Query params: %s", query)

	json.NewEncoder(w).Encode(userData)
}

func main() {
	// Create Server and Route Handlers
	r := mux.NewRouter()

	r.HandleFunc("/", rootHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Configure Logging
	logFileLocation := os.Getenv("LOG_FILE_LOCATION")
	if logFileLocation != "" {
		log.SetOutput(&lumberjack.Logger{
			Filename:   logFileLocation,
			MaxSize:    500, //megabytes
			MaxBackups: 3,
			MaxAge:     28,   //days
			Compress:   true, // disabled by default
		})
	}

	// Start Server
	go func() {
		log.Println("Starting Server")
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// Graceful Shutdown
	waitForShutdown(srv)
}

func waitForShutdown(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-interruptChan

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	srv.Shutdown(ctx)

	log.Println("Shutting down")
	os.Exit(0)
}
