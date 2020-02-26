package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	ClientID string
	Secret   string
	Exp      time.Time
	Remain   int
}

var clientID string // only for /users now, but may be expanded in the future
var token Auth      // there is only one set of valid credentials at a time *scream*

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
		"and use the information provided in order to retrieve valid data from `/users`. " +
		"You should already have instructions for how to form a valid request. Good luck!")))
}

type HttpError struct {
	Code    int
	Message string
}

func (e *HttpError) Error() string {
	return fmt.Sprintf("%d - %s", e.Code, e.Message)
}

// auth token must have remaining attempts available
// Authorization header must
func credentialValidator(authString string) error {
	// new auth token required due to too many attempts
	if token.Remain == 0 {
		return &HttpError{401, "No login attempts remain, obtain new credentials and try again.\n"}
	} else if time.Now().After(token.Exp) { // new auth token required due to timeout
		return &HttpError{401, "Token has expired, obtain new credentials and try again.\n"}
	}
	credentials := strings.Split(authString, ":") // split credentials into clientId and checksum
	if len(credentials) != 2 {                    // verify number of params in Auth header
		return &HttpError{403, "Forbidden - verify Authorization token formatting.\n"}
	} else if credentials[0] != clientID || credentials[1] != token.Secret {
		// decrement access attempts remaining
		token.Remain--
		return &HttpError{403, fmt.Sprintf("Forbidden - could not authorize credentials."+
			"You have %d attempts remaining.", token.Remain)}
	}
	return nil
}

// This function returns a list of all user GUIDs
// if a user provides a properly-hashed secret key and client id, returns users from the API
// otherwise it returns a generic "Bad Request", because it is a bad server
// TODO: break out validations into a new function
func returnAllUsers(w http.ResponseWriter, r *http.Request) {
	// Request must contain a properly-formatted Authorization header as described in README
	err := credentialValidator(r.Header.Get("Authorization"))

	if err != nil {
		fmt.Printf("Error: %+v", err)
		w.WriteHeader(err.(*HttpError).Code)
		w.Write([]byte(err.(*HttpError).Message))
		return
	}

	fmt.Println("request validated, returning user list")
	userData := loadUsers()
	userLen := len(userData.Users)
	guids := make([]string, userLen+1) //here we can make slices of different length than users array, providing an extra line for the expected count
	guids[0] = fmt.Sprintf("Count: %d", userLen)
	for i, u := range userData.Users {
		guids[i+1] = u.GUID // include offset for count of users
	}
	w.Write([]byte(strings.Join(guids[:], "\n")))
}

// our simple auth is implemented as follows:
// client id key is generated so that this server knows what to validate against
// a secret key is provided that the consumer needs to hash properly to validate the API
// the credentials will expire after 5 minutes or 5 login failures, a new key must be generated
// if a user provides an md5 checksum of the secret key+api with a valid client id, then they may retrieve users from the API
func generateTokens(w http.ResponseWriter, r *http.Request) {
	// Generate secret key
	created := time.Now()
	rand.Seed(created.UnixNano())
	secretKey := rand.Intn(99999999)
	secretPrehash := strconv.Itoa(secretKey) + "/users" // leaving this here for clarity
	secret := fmt.Sprintf("%x", md5.Sum([]byte(secretPrehash)))

	token = Auth{ClientID: clientID,
		Secret: secret,
		Exp:    created.Add(time.Minute * 5),
		Remain: 5} // this disables auth lock

	fmt.Printf("Token: %+v\n", token)

	w.Header().Add("WWW-Authenticate", `Basic realm=users`)
	w.Header().Add("clientID", clientID)
	w.Header().Add("secret", strconv.Itoa(secretKey))
	w.Write([]byte("Congratulations, you are now authenticated! You can use the provided credentials to build a valid request and retrieve the list of users. Additional requests to `/auth` will invalidate these credentials and return a new set."))
}

func main() {
	// Create Server and Route Handlers
	r := mux.NewRouter()
	r.HandleFunc("/", rootHandler)
	r.HandleFunc("/users", returnAllUsers)
	r.HandleFunc("/auth", generateTokens)

	// Generate client key for /users
	created := time.Now()
	rand.Seed(created.UnixNano())
	clientSeed := rand.Intn(99999999)
	h := md5.New()
	io.WriteString(h, strconv.Itoa(clientSeed))
	clientID = fmt.Sprintf("%x", h.Sum(nil))

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
		log.Println("Server Started")
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
