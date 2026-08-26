package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"subsurface-survey-gate/internal/application"
)

const maxBodyBytes = 1 << 20

func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("请求体只能包含一个 JSON 对象")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeResult(w http.ResponseWriter, result application.Result) {
	if result.Replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	w.WriteHeader(result.Status)
	_, _ = w.Write(append(result.Body, '\n'))
}
