package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/snikch/goodman/hooks"
	trans "github.com/snikch/goodman/transaction"
)

var (
	db sql.DB
)

func main() {
	h := hooks.NewHooks()
	server := hooks.NewServer(hooks.NewHooksRunner(h))

	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	h.BeforeAll(func(t []*trans.Transaction) {
		_, err = db.Exec(`create table users (id integer not null primary key, name text, email text);`)
		if err != nil {
			log.Fatal(err)
		}
	})
	h.Before("Users > Endpoint > Getting a single user", func(t *trans.Transaction) {
		_, err = db.Exec(`insert into users (id, name, email) values (1, 'Dom', 'ddelnano@gmail.com');`)
		if err != nil {
			log.Fatal(err)
		}
	})
	h.Before("Users > Endpoint > Retrieve all users", func(t *trans.Transaction) {
		t.Skip = true
	})
	h.AfterAll(func(t []*trans.Transaction) {
		os.Remove("./foo.db")
	})
	server.Serve()
	defer server.Listener.Close()
}
