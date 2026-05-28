package schemas

import (
	"errors"
	"honux-core/internal/domain/apperror"
	"honux-core/internal/utils"
	"net/http"
)

type HTTPErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func RespondError(w http.ResponseWriter, r *http.Request, err error) {

	if appErr, ok := errors.AsType[*apperror.AppError](err); ok {
		if appErr.HTTPStatus == http.StatusInternalServerError {
			// Internal logger
		}

		// ✅ Serializa la struct completa — Fields queda incluido
		utils.WriteJSON(w, appErr.HTTPStatus, HTTPErrorResponse{
			Code:    string(appErr.Code),
			Message: appErr.Message,
			Fields:  appErr.Fields,
		})
		return
	}

	// Error desconocido — nunca exponer detalles al cliente
	utils.WriteJSON(w, http.StatusInternalServerError, HTTPErrorResponse{
		Code:    string(apperror.CodeInternal),
		Message: "an unexpected error occurred",
	})
}
