package handlers

import (
	"encoding/json"
	"net/http"

	"support_bot/internal/http/errorz"
)

func HandleRawJSON(w http.ResponseWriter, code int, data []byte) {
	w.WriteHeader(code)
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

func HandleJSON(w http.ResponseWriter, code int, data any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(err.Error()))
		return err
	}
	w.WriteHeader(code)
	w.Header().Add("Content-Type", "application/json")
	w.Write(raw)
	return nil
}

func AnswerErrJSON(w http.ResponseWriter, err errorz.ClientErr) {
	HandleRawJSON(w, err.Code, []byte(err.Desc))
}
