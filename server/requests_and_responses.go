package server

import (
	"encoding/json"
	"net/http"
)

type ResponseData map[string]string

func WriteResponse(w http.ResponseWriter, statusCode int, response ResponseData) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")

	var responseBytes []byte
	var err error
	if responseBytes, err = json.Marshal(response); err != nil {
		w.Write([]byte("failed to write response"))
	}

	w.Write(responseBytes)
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	response := ResponseData{
		"error": message,
	}
	WriteResponse(w, statusCode, response)
}
