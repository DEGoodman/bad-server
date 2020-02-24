package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Users struct {
	Users []User `json:"users"`
}

type User struct {
	GUID     string `json:"guid"` // github.com/google/uuid is more than we need now
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

type Auth struct {
	ClientID int
	Secret   hash.Hash
	Exp      time.Time
	Attempts int
}

var clientID map[string]int
var tokens []Auth

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
	w.Write([]byte(fmt.Sprintf("Welcome to the homepage! You must authenticate via `/auth` " +
		"and use the information provided in order to retrieve a valid reponse from `/users`." +
		"You should already have instructions for how to form a valid request. Good luck!")))
}

// This function returns a list of all user GUIDs
// The consumer should already be expecting a certain number of GUIDs
// TODO: manipulate response length to "break" api response
func returnAllUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Endpoint Hit: returnAllUsers")
	fmt.Println("validating request")
	if !validateChecksum(r.Header.Get("X-Request-Checksum")) {
		fmt.Println("validation failure")
		http.Error(w, "Not authorized", 401)
		return
	}
	fmt.Println("request validated")
	userData := loadUsers()
	userLen := len(userData.Users)
	guids := make([]string, userLen) //here we can make slices of different length than users array
	for i, u := range userData.Users {
		guids[i] = u.GUID
	}
	w.Write([]byte(strings.Join(guids[:], "\n")))
}

// our simple auth is implemented as follows:
// client id key is generated so that this server knows what to validate against
// a secret key is provided that the consumer needs to hash properly to validate the API
//   - the secret key will expire after 5 minutes or 10 login failures, requiring a new key to be generated
// if a user provides a properly-hashed secret key with a valid client id, then they may retrieve users from the API
func generateTokens(w http.ResponseWriter, r *http.Request) {
	// the secret key is a randomly generated 8-digit number
	created := time.Now()
	rand.Seed(created.UnixNano())
	secretKey := rand.Intn(99999999)
	secret := sha256.New()
	secret.Write([]byte(fmt.Sprintf("%d/users", secretKey)))

	tokens = append(tokens, Auth{ClientID: clientID["users"],
		Secret:   secret,
		Exp:      created.Add(time.Minute * 5),
		Attempts: 0})

	w.Write([]byte(fmt.Sprintf("ClientId: %d\nSecret Key: %d", clientID["users"], secretKey)))
}

func validateChecksum(checksum string) bool {
	return checksum == "12345/users"
}

func main() {
	// Create Server and Route Handlers
	r := mux.NewRouter()
	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/users", returnAllUsers)
	r.HandleFunc("/auth", generateTokens)

	// Set client key for /users
	clientID["users"] = rand.Int()

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
