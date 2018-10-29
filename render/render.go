package render

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// JSON encodes body and sends it as a JSON document.
func JSON(w http.ResponseWriter, body interface{}, code int) {
	headers := w.Header()
	headers.Set("Content-Type", "application/json; charset=UTF-8")

	// Do not cache errors
	if code >= 400 {
		headers.Set("Cache-Control", "no-store")
		headers.Set("Pragma", "no-cache")
		headers.Set("Expires", "0")
	}

	w.WriteHeader(code)

	if body == nil {
		body = http.StatusText(code)
	}

	if err := json.NewEncoder(w).Encode(body); err != nil {
		http.Error(w, "failed encoding JSON structure", http.StatusInternalServerError)
	}
}

// Unauthorized returns a 401 Unauthorized response with the string representation of the error
// as part of the WWW-Authenticate header.
func Unauthorized(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusUnauthorized)
	headers := w.Header()
	headers.Set("WWW-Authenticate", fmt.Sprintf("Bearer %s", err))
	headers.Set("Cache-Control", "no-store")
	headers.Set("Pragma", "no-cache")
	w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
}
