package router

import (
	"github.com/gorilla/mux"

	"gitlab.com/gitlab-org/gitaly/handler"
)

func NewRouter() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", handler.Home)

	return r
}
