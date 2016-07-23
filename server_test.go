package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotAuthorized(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")
	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status [%v] but got [%v]", http.StatusUnauthorized, res.Code)
	}
}

func TestAuthorizedWithoutCommand(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader("token=hospNwvYl5EdtWuoZvHiawfr"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; param=value")

	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("Expected status [%v] but got [%v]", http.StatusBadRequest, res.Code)
	}
}
