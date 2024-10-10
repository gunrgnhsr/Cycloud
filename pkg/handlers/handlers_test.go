package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gunrgnhsr/Cycloud/pkg/auth"
	_ "github.com/lib/pq"
)

func TestLogin(t *testing.T) {
	// Create a request with valid credentials
	credentials := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: "testuser",
		Password: "testpassword",
	}
	jsonCredentials, _ := json.Marshal(credentials)
	req, err := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonCredentials))
	if err != nil {
		t.Fatal(err)
	}

	// Create a response recorder
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(Login) // Create a handler for the Login function

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &response)
	token, ok := response["token"].(string)
	if !ok || token == "" {
		t.Error("handler did not return a valid token")
	}

	// Validate the token (optional, but recommended)
	_, err = auth.ValidateJWT(token)
	if err != nil {
		t.Errorf("handler returned an invalid token: %v", err)
	}
}
