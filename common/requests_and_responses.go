package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type ResponseData map[string]string

func UnmarshalJSONRequestBody(w http.ResponseWriter, r *http.Request, dst interface{}) (statusCode int, err error) {
	r.Body = http.MaxBytesReader(w, r.Body, 524288)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err = decoder.Decode(&dst); err != nil {
		var unmarshalTypeError *json.UnmarshalTypeError
		var syntaxError *json.SyntaxError
		switch {
		case errors.As(err, &unmarshalTypeError):
			fallthrough
		case errors.As(err, &syntaxError):
			fallthrough
		case errors.Is(err, io.ErrUnexpectedEOF):
			err = fmt.Errorf("request body contains invalid JSON")
			statusCode = http.StatusBadRequest
			return
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			err = fmt.Errorf("request body contains unknown field")
			statusCode = http.StatusBadRequest
			return
		case errors.Is(err, io.EOF):
			err = fmt.Errorf("request body is empty")
			statusCode = http.StatusBadRequest
			return
		case err.Error() == "http: request body too large":
			err = fmt.Errorf("request body too large")
			statusCode = http.StatusBadRequest
			return
		default:
			statusCode = http.StatusInternalServerError
			return
		}
	}
	return
}

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
