package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetHome(t *testing.T) {
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal("Creating 'GET /' request failed!")
	}

	http.HandlerFunc(Home).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatal("Server error: Returned ", recorder.Code, " instead of ", http.StatusOK)
	}

	if s := strings.TrimSpace(recorder.Body.String()); s != "All routes lead to Gitaly" {
		t.Fatal("Expected GET / to return 'All routes lead to Gitaly', got '", s, "'")
	}
}
