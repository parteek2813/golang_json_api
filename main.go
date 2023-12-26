package main

import (
	"flag"
	"fmt"
	"log"
)


func seedAccount(store Storage, fname, lname, pw string) *Account{
	acc, err := NewAccount(fname, lname, pw)
	if err != nil{
		log.Fatal(err)
	}

	if err:= store.CreateAccount(acc); err != nil{
		log.Fatal(err)
	}
	
	fmt.Println("new account =>", acc.Number)

	return acc
}


func seeAccounts(s Storage){
	seedAccount(s, "parteeek", "kumar", "hunter88888")
}

func main() {

	seed := flag.Bool("seed", false, "seed the db")
	flag.Parse()


	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	if *seed {

		fmt.Println("Seeding the database")
	// seed stuff
	seeAccounts(store)
	}



	server := NewAPIServer(":3000", store)
	server.Run()

}