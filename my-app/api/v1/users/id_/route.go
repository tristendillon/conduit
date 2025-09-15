package id_

import (
	"encoding/json"
	"my-app/api/v1/users/user_repo"
	"net/http"
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func GET(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	user := user_repo.FindUser(id)
	if user == nil {
		http.Error(w, "The user you are looking for does not exist", http.StatusNotFound)
		return
	}
	data, err := json.Marshal(user)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func DELETE(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	user := user_repo.DeleteUser(id)
	if user == -1 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully deleted user"))
}
