package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)




type APIServer struct {
	listenAddr string
	store Storage
}

func NewAPIServer(listenAddr string, store Storage) *APIServer {
	return &APIServer{
		listenAddr: listenAddr,
		store: store,
	}
}

func (s *APIServer) Run(){
	router := mux.NewRouter()

	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID)))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))
	
	log.Println("JSON API server running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)

	

}

func (s *APIServer) handleAccount(w http.ResponseWriter, r *http.Request )error {

	if r.Method == "GET" {
		return s.handleGetAccount(w, r)
	}


	if(r.Method == "POST"){
		return s.handleCreateAccount(w, r)
	}





	return fmt.Errorf("methods not allowed: %s", r.Method)
}


// Get / account
func (s *APIServer) handleGetAccount(w http.ResponseWriter, r *http.Request )error {
	
	accounts, err := s.store.GetAccounts()
	if err != nil {
		return err
	}

	return WriteJSON(w, http.StatusOK, accounts)
}




func (s *APIServer) handleGetAccountByID(w http.ResponseWriter, r *http.Request )error {
	
	// The id can be of string to .... so for checking convert it into the int 

	if r.Method == "GET"{

		
		id, err := getID(r)
		if err != nil {
			return err
		}
		

		account, err := s.store.GetAccountByID(id)

		if err != nil {
			return err
		}

		return WriteJSON(w, http.StatusOK, account)
	}

	if r.Method == "DELETE"{
		return s.handleDeleteAccount(w, r)
	}
	return fmt.Errorf("method not allowed %s", r.Method)
}




func (s *APIServer) handleCreateAccount(w http.ResponseWriter, r *http.Request )error {

	createAccountReq  := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(createAccountReq); err != nil {
		return err
	}

	account := NewAccount(createAccountReq.FirstName, createAccountReq.LastName)

	// create account in store
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	tokenString, err := createJWT(account)
	if err != nil {
		return err
	}

	fmt.Println("JWT token: ", tokenString)


	return WriteJSON(w, http.StatusOK, account)
}


func (s *APIServer) handleDeleteAccount(w http.ResponseWriter, r *http.Request )error {
	
	id, err := getID(r)
	if err != nil {
		return err
	}
	
	if err := s.store.DeleteAccount(id); err != nil {
		return err
	}
	return WriteJSON(w, http.StatusOK, map[string]int{"deleted": id})
}

func (s *APIServer) handleTransfer(w http.ResponseWriter, r *http.Request )error {
	transferReq := new(TransferRequest)
	if err := json.NewDecoder(r.Body).Decode(transferReq); err != nil {
		return err
	}

	defer r.Body.Close()


	
	return WriteJSON(w, http.StatusOK, transferReq)
}

// HELPER FUNCTIONS
func WriteJSON(w http.ResponseWriter, status int , v any) error{
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	
	
	return json.NewEncoder(w).Encode(v)
}

// eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJhY2NvdW50TnVtYmVyIjo0OTgzOCwiZXhwaXJlc0F0IjoxNTAwMH0.mMB0bPOu84h9gDN9JUG4sa75xSC_-OkbII0EVOLlhHA

func createJWT(account *Account)(string, error){
	claims := &jwt.MapClaims{
		"expiresAt": 15000,
		"accountNumber": account.Number,
	}

	secret := jwtSecret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	
	return token.SignedString([]byte(secret))
}

func withJWTAuth(handlerFunc http.HandlerFunc) http.HandlerFunc{

	
	return func(w http.ResponseWriter, r *http.Request){
		fmt.Println("Calling JWT auth middleware")

		tokenString := r.Header.Get("x-jwt-token")
		// Validate here
		_, err := validateJWT(tokenString)


		if err != nil {
			WriteJSON(w, http.StatusForbidden, ApiError{Error: "invalid token"})
			return 
		}
		handlerFunc(w, r)
	}
}

const jwtSecret = "hunter9999"

func validateJWT(tokenString string)(*jwt.Token, error){
	
	secret := jwtSecret
	return jwt.Parse(tokenString, func(token *jwt.Token)(interface{}, error){
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}



		// HmacSampleSecret is containing a []byte secret e.g []byte("my_secret_key")

		return []byte(secret), nil
	})
}

type apiFunc func(http.ResponseWriter, *http.Request) error 

type ApiError struct {
	Error string `json:"error"`
}

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request){
		if err := f(w, r); err != nil {
			WriteJSON(w, http.StatusBadRequest, ApiError{Error: err.Error()})
		}
	}
}


func getID(r *http.Request)(int , error){
	idStr := mux.Vars(r)["id"]

	id, err := strconv.Atoi(idStr)

	if err != nil {
		return id, fmt.Errorf("invalid id given %s", idStr)
	}

	return id, nil
}