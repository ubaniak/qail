package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeJSON marshals body and writes it with the given status. Errors
// during marshal write a 500 with a plain text body so the client never
// sees a half-written JSON document.
func writeJSON(w http.ResponseWriter, status int, body any) {
	buf, err := json.Marshal(body)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf)
}

// writeError sends an `{ "error": "<msg>" }` JSON body with status. The
// msg is the err's Error string; the helper exists so handlers don't have
// to assemble the wrapper map every time.
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

// decodeJSON reads r.Body into out. Returns a 400-friendly error on
// malformed JSON; the caller is responsible for translating it into the
// HTTP response.
func decodeJSON(r *http.Request, out any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// pathParam returns the named path segment from r (Go 1.22 mux style) or
// an error if it is empty.
func pathParam(r *http.Request, name string) (string, error) {
	v := r.PathValue(name)
	if v == "" {
		return "", fmt.Errorf("missing path parameter %q", name)
	}
	return v, nil
}

