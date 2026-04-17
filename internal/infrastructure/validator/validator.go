package validator

import (
	"reflect"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	v := &Validator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}

	// Use JSON field names in errors automatically
	v.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		// Omit fields with json:"-"
		if name == "-" {
			return ""
		}

		return name
	})

	// Register custom validators
	v.validate.RegisterStructValidation(
		validateAuthenticationInput,
		dtos.AuthenticationInputDto{},
	)

	return v
}

func (v *Validator) ValidateStruct(data any) map[string]string {
	err := v.validate.Struct(data)
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"_error": err.Error()}
	}

	errors := make(map[string]string)
	for _, fieldErr := range validationErrs {
		fieldName := fieldErr.Field()
		errors[fieldName] = formatFieldError(fieldErr)
	}

	return errors
}

func formatFieldError(err validator.FieldError) string {
	field := err.Field()
	tag := err.Tag()
	param := err.Param()

	messages := map[string]string{
		"required": field + " is required",
		"email":    field + " must be a valid email address",
		"max":      field + " must be at most " + param + " characters",
		"min":      field + " must be at least " + param + " characters",
	}

	message, exists := messages[tag]
	if exists {
		return message
	}

	return field + " is invalid"
}

func validateAuthenticationInput(sl validator.StructLevel) {
	input := sl.Current().Interface().(dtos.AuthenticationInputDto)

	// If purpose is registration, create user dto is required
	if input.Purpose == domain.OTPPurposeRegistration && input.User == nil {
		sl.ReportError(input.User, "User", "user", "required", "")
	}
}
