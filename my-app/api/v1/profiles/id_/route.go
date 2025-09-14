package id_

import (
	"fmt"
	"net/http"
)

func GET(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	// profile := profile_repo.FindProfile(id)
	// if profile == nil {
	// 	http.Error(w, "Profile not found", http.StatusNotFound)
	// 	return
	// }
	// data, err := json.Marshal(profile)
	// if err != nil {
	// 	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	// 	return
	// }
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(id))
}

func DELETE(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	fmt.Println(id)
	// profile := profile_repo.DeleteProfile(id)
	// if profile == -1 {
	// 	http.Error(w, "Profile not found", http.StatusNotFound)
	// 	return
	// }

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Successfully deleted profile"))
}
