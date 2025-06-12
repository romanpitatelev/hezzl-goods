package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/romanpitatelev/hezzl-goods/internal/entity"
	"github.com/rs/zerolog/log"
)

func ErrorResponse(w http.ResponseWriter, errorText string, err error) {
	statusCode := getStatusCode(err)

	errResp := fmt.Errorf("%s: %w", errorText, err).Error()
	if statusCode == http.StatusInternalServerError {
		errResp = http.StatusText(http.StatusInternalServerError)

		log.Warn().Err(err).Send()
	}

	response, err := json.Marshal(errResp)
	if err != nil {
		log.Warn().Msgf("error marshalling response: %v", err)
	}

	w.WriteHeader(statusCode)

	if _, err := w.Write(response); err != nil {
		log.Warn().Msgf("error writing response: %v", err)
	}
}

func OkResponse(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Warn().Msgf("error encoding response: %v", err)
	}
}

func getStatusCode(err error) int {
	switch {
	default:
		return http.StatusInternalServerError
	}
}

func GetListRequest(r *http.Request) entity.ListRequest {
	queryParams := r.URL.Query()

	parameters := entity.ListRequest{}

	parameters.Limit, _ = strconv.Atoi(queryParams.Get("limit"))
	parameters.Offset, _ = strconv.Atoi(queryParams.Get("offset"))

	return parameters
}
