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

	router.HandleFunc("/login", makeHTTPHandleFunc(s.handleLogin))
	router.HandleFunc("/account", makeHTTPHandleFunc(s.handleAccount))
	
	router.HandleFunc("/account/{id}", withJWTAuth(makeHTTPHandleFunc(s.handleGetAccountByID), s.store))
	router.HandleFunc("/transfer", makeHTTPHandleFunc(s.handleTransfer))
	
	log.Println("JSON API server running on port: ", s.listenAddr)

	http.ListenAndServe(s.listenAddr, router)

	

}

// 636918

func (s *APIServer) handleLogin(w http.ResponseWriter, r *http.Request )error {


	if r.Method != "POST" {
		return fmt.Errorf("method not allowed %s", r.Method)
	}
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}


	acc, err := s.store.GetAccountByNumber(int(req.Number))

	if err != nil {
		return err
	}


	fmt.Printf("%+v\n", acc)




	return WriteJSON(w, http.StatusOK, req)

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

	req  := new(CreateAccountRequest)
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		return err
	}

	account, err := NewAccount(req.FirstName, req.LastName, req.Password)
	if err != nil {
		return err
	}
	// create account in store
	if err := s.store.CreateAccount(account); err != nil {
		return err
	}

	



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


func permissionDenied(w http.ResponseWriter){
	WriteJSON(w, http.StatusForbidden, ApiError{Error: "permission denied"})
}

func withJWTAuth(handlerFunc http.HandlerFunc, s Storage) http.HandlerFunc{

	
	return func(w http.ResponseWriter, r *http.Request){
		fmt.Println("Calling JWT auth middleware")

		tokenString := r.Header.Get("x-jwt-token")
		// Validate here
		token, err := validateJWT(tokenString)


		if err != nil {
			permissionDenied(w)
			return 
		}


		if !token.Valid {
			permissionDenied(w)
			return 
		}

		userID, err := getID(r)
		if err != nil {
			permissionDenied(w)
			return
		}

		account, err := s.GetAccountByID(userID)
		if err != nil {
			permissionDenied(w)
			return
		}


		claims := token.Claims.(jwt.MapClaims)

		// if account is not correct , just deny the permission
		if account.Number != int64(claims["accountNumber"].(float64)){
			permissionDenied(w)
			return
		}

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