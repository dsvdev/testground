package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dsvdev/testground/example/simple_backend/handler"
	"github.com/dsvdev/testground/example/simple_backend/repository"
	"github.com/go-chi/chi/v5"
)

type UserDto struct {
	Name string `json:"name"`
}

func main() {
	r := chi.NewRouter()
	ctx := context.Background()
	connStr := os.Getenv("DATABASE_URL")
	repo, err := repository.NewUserRepo(ctx, connStr)
	if err != nil {
		panic(err)
	}

	userHandler := handler.NewUserHandler(repo)

	r.Get("/users/{userId}", func(w http.ResponseWriter, r *http.Request) {
		userIDStr := chi.URLParam(r, "userId")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		user, err := userHandler.GetUserById(r.Context(), userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		if user == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(user)
	})

	r.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		var user UserDto
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		newUser, err := userHandler.AddNewUser(ctx, user.Name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newUser)
	})

	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
