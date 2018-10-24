package render

import (
	"encoding/json"
	"net/http"
)

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
