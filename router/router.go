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
	r.HandleFunc("/projects/{id:[0-9]+}/git-http/info-refs/{service:(upload|receive)-pack}", GetInfoRefs)

	return handlers.LoggingHandler(os.Stdout, r)
}
