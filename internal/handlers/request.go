package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ncondes/fifa-world-cup-pickems/internal/infrastructure/validator"
)

const maxBodySizeInBytes = 1_048_576 // 1 MB

func readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	// Limit the size of the request body to 1MB to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySizeInBytes)

	decoder := json.NewDecoder(r.Body)
	// Prevent injection of additional fields
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(data); err != nil {
		return parseJSONError(err)
	}

	return nil
}

func readAndValidateJSON(
	w http.ResponseWriter,
	r *http.Request,
	data any,
	v *validator.Validator,
) error {
	if err := readJSON(w, r, data); err != nil {
		respondWithError(w, http.StatusBadRequest, err)
		return err
	}

	if validationErrors := v.ValidateStruct(data); len(validationErrors) > 0 {
		respondWithValidationError(w, validationErrors)
		return errValidationFailed
	}

	return nil
}

func parseJSONError(err error) error {
	var syntaxError *json.SyntaxError
	var unmarshalTypeError *json.UnmarshalTypeError
	var maxBytesError *http.MaxBytesError

	switch {
	// Empty body
	case errors.Is(err, io.EOF):
		return errEmptyBody

	// Syntax error in JSON
	case errors.As(err, &syntaxError):
		return errMalformedJSON(syntaxError.Offset)

	// Wrong type for field
	case errors.As(err, &unmarshalTypeError):
		return errInvalidValueForField(
			unmarshalTypeError.Field,
			unmarshalTypeError.Type.String(),
		)

	// Unknown field
	case strings.HasPrefix(err.Error(), "json: unknown field"):
		// Extract field name from error message
		fieldName := extractFieldName(err.Error())
		return errUnknownField(fieldName)

	// Body too large
	case errors.As(err, &maxBytesError):
		return errBodyTooLarge

	default:
		return errInvalidRequestBody
	}
}

func extractFieldName(errMsg string) string {
	// "json: unknown field \"emal\"" -> "emal"
	parts := strings.Split(errMsg, "\"")
	if len(parts) >= 2 {
		return parts[1]
	}

	return "unknown"
}
