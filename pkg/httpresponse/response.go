package httpresponse

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	pkgerrors "github.com/iqbaljlldn/nexus/pkg/errors"
)

// Envelope Structs
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Success Helpers
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMeta(c *gin.Context, statusCode int, data interface{}, meta interface{}) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error Handler
func Error(c *gin.Context, err error) {
	var domainErr *pkgerrors.DomainError
	var valErr *pkgerrors.ValidationError
	var infraErr *pkgerrors.InfrastructureError

	// 1. Tangani Domain Error (Expected Business Errors)
	if errors.As(err, &domainErr) {
		status := mapDomainCodeToStatus(domainErr.Code)
		c.JSON(status, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    domainErr.Code,
				Message: domainErr.Message,
			},
		})
		return
	}

	// 2. Tangani Validation Error
	if errors.As(err, &valErr) {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "VALIDATION_ERROR",
				Message: "Validasi input gagal.",
				Details: valErr.Fields,
			},
		})
		return
	}

	// 3. Tangani Infrastructure Error (Unexpected System Errors)
	if errors.As(err, &infraErr) {
		// Logika tambahan: Anda bisa mengambil logger dari context di sini
		// untuk melog infraErr.Err secara internal, tapi jangan bocorkan ke client.
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    pkgerrors.CodeInternalServerError,
				Message: "Terjadi kesalahan pada sistem.",
			},
		})
		return
	}

	// 4. Default / Unknown Error
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Success: false,
		Error: ErrorDetail{
			Code:    pkgerrors.CodeUnexpectedError,
			Message: "Terjadi kesalahan yang tidak terduga.",
		},
	})
}

func mapDomainCodeToStatus(code string) int {
	switch code {
	case pkgerrors.CodeUserNotFound, pkgerrors.CodeRecordNotFound:
		return http.StatusNotFound
	case pkgerrors.CodeUserUnauthorized, pkgerrors.CodeTokenExpired, pkgerrors.CodeTokenInvalid:
		return http.StatusUnauthorized
	case pkgerrors.CodeUserAlreadyExists, pkgerrors.CodeDuplicateRecord:
		return http.StatusConflict
	case pkgerrors.CodeInvalidCredentials, pkgerrors.CodeInvalidPassword, pkgerrors.CodeMissingRequiredField, pkgerrors.CodeInvalidFieldFormat:
		return http.StatusBadRequest
	default:
		// Default untung domain error adalah 400 Bad Request
		return http.StatusBadRequest
	}
}
