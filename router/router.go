package router

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func NewRouter() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/", Home)

	return handlers.LoggingHandler(os.Stdout, r)
}
