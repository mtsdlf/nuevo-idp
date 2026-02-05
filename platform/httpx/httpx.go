package httpx

import (
	"encoding/json"
	"net/http"
)

// WriteJSON escribe una respuesta JSON con el status dado.
// Ignora errores de serialización para no romper el handler.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteText escribe una respuesta de texto plano con el status dado.
func WriteText(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

// RequireMethod valida que la request use el método esperado.
// Si no coincide, escribe 405 y devuelve false.
func RequireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		WriteText(w, http.StatusMethodNotAllowed, "method not allowed")
		return false
	}
	return true
}

// DecodeJSON decodifica el body como JSON en dst.
// Si falla, escribe un 400 con el mensaje dado (o "invalid json" si está vacío)
// y devuelve false.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any, msg string) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if msg == "" {
			msg = "invalid json"
		}
		WriteText(w, http.StatusBadRequest, msg)
		return false
	}
	return true
}
