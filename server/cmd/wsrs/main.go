package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/kevinfjq/go-react-ask-app/server/internal/api"
	"github.com/kevinfjq/go-react-ask-app/server/internal/store/pgstore"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		os.Getenv("WSRS_DATABASE_USER"),
		os.Getenv("WSRS_DATABASE_PASSWORD"),
		os.Getenv("WSRS_DATABASE_HOST"),
		os.Getenv("WSRS_DATABASE_PORT"),
		os.Getenv("WSRS_DATABASE_DB")))
	if err != nil {
		panic(err)
	}

	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		panic(err)
	}

	handler := api.NewHandler(pgstore.New(pool))
	go func() {
		if err := http.ListenAndServe(":8080", handler); err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				panic(err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
}
