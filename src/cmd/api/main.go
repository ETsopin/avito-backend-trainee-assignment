package main

import (
	"avitotask/internal/data"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type config struct {
	addr string
	db   struct {
		serverAddress    string
		postgresConn     string
		postgresJDBC     string
		postgresUsername string
		postgresPassword string
		postgresHost     string
		postgresPort     string
		postgresDB       string
	}
}

type application struct {
	config config
	models data.Models
}

func main() {

	cfg := config{}

	cfg.addr = os.Getenv("SERVER_ADDRESS")
	if cfg.addr == "" {
		cfg.addr = ":8080"
	}
	cfg.db.postgresConn = os.Getenv("POSTGRES_CONN")
	cfg.db.postgresJDBC = os.Getenv("POSTGRES_JDBC_URL")
	cfg.db.postgresUsername = os.Getenv("POSTGRES_USERNAME")
	cfg.db.postgresPassword = os.Getenv("POSTGRES_PASSWORD")
	cfg.db.postgresHost = os.Getenv("POSTGRES_HOST")
	cfg.db.postgresPort = os.Getenv("POSTGRES_PORT")
	cfg.db.postgresDB = os.Getenv("POSTGRES_DATABASE")
	db, err := openDB(cfg)
	if err != nil {
		cfg.db.postgresConn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", cfg.db.postgresUsername, cfg.db.postgresPassword, cfg.db.postgresHost, cfg.db.postgresPort, cfg.db.postgresDB)
		db, err = openDB(cfg)
	}
	//fmt.Println(cfg.db.postgresConn)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Started a db connection pool")
	defer db.Close()

	app := &application{
		config: cfg,
		models: data.NewModels(db),
	}

	err = app.models.Tables.CreateTables()

	if err != nil {
		log.Println(err)
	}

	srv := &http.Server{
		Addr:    cfg.addr,
		Handler: app.routes(),
	}

	err = srv.ListenAndServe()

	if err != nil {
		log.Fatal(err)
	}
}

func openDB(cfg config) (*sql.DB, error) {

	db, err := sql.Open("postgres", cfg.db.postgresConn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
