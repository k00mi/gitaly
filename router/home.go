package router

import (
	"net/http"
)

func Home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("All routes lead to Gitaly\n"))
}
