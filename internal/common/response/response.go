package response

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func JSON(w http.ResponseWriter, httpStatus int, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	_ = json.NewEncoder(w).Encode(Response{
		Code:    code,
		Data:    data,
		Message: message,
	})
}

func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, 0, "success", data)
}

func Fail(w http.ResponseWriter, code int, message string) {
	httpStatus := http.StatusInternalServerError
	if code >= 400 && code <= 599 {
		httpStatus = code
	}
	JSON(w, httpStatus, code, message, nil)
}

func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, http.StatusBadRequest, message, nil)
}

func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, http.StatusUnauthorized, message, nil)
}

func InternalError(w http.ResponseWriter, message string) {
	if message == "" {
		message = "Internal Server Error"
	}
	JSON(w, http.StatusInternalServerError, http.StatusInternalServerError, message, nil)
}
