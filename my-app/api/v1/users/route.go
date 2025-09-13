package users

import (
	"encoding/json"
	"my-app/api/v1/users/user_repo"
	"net/http"
)

func GET(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	users := user_repo.GetAllUsers()
	data, err := json.Marshal(users)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}
