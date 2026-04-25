package validator

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/go-playground/validator/v10"
)

type ValidationField struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params,omitempty"`
}

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	v := &Validator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}

	v.validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		// Omit fields with "-" json tag
		if name == "-" {
			return ""
		}

		return name
	})

	v.validate.RegisterStructValidation(
		validateAuthenticationInput,
		dtos.AuthenticationInputDto{},
	)
	v.validate.RegisterValidation("min_array_len", validateMinArrayLen)

	return v
}

func (v *Validator) ValidateStruct(data any) map[string]ValidationField {
	err := v.validate.Struct(data)
	if err == nil {
		return nil
	}

	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]ValidationField{"_error": {Code: "INVALID", Message: err.Error()}}
	}

	fields := make(map[string]ValidationField)
	for _, fieldErr := range validationErrs {
		fields[fieldErr.Field()] = formatFieldError(fieldErr)
	}

	return fields
}

func formatFieldError(err validator.FieldError) ValidationField {
	tag := err.Tag()
	param := err.Param()

	switch tag {
	case "required":
		return ValidationField{Code: "REQUIRED", Message: "is required"}
	case "email":
		return ValidationField{Code: "INVALID_EMAIL", Message: "must be a valid email address"}
	case "min":
		return ValidationField{Code: "MIN_LENGTH", Message: "must be at least " + param + " characters", Params: map[string]any{"min": parseParamInt(param)}}
	case "max":
		return ValidationField{Code: "MAX_LENGTH", Message: "must be at most " + param + " characters", Params: map[string]any{"max": parseParamInt(param)}}
	case "min_array_len":
		n := parseParamInt(param)
		suffix := "elements"
		if param == "1" {
			suffix = "element"
		}
		return ValidationField{Code: "MIN_ELEMENTS", Message: "must have at least " + param + " " + suffix, Params: map[string]any{"min": n}}
	case "gt":
		return ValidationField{Code: "GT", Message: "must be greater than " + param, Params: map[string]any{"value": parseParamInt(param)}}
	case "lt":
		return ValidationField{Code: "LT", Message: "must be less than " + param, Params: map[string]any{"value": parseParamInt(param)}}
	case "gte":
		return ValidationField{Code: "GTE", Message: "must be greater than or equal to " + param, Params: map[string]any{"value": parseParamInt(param)}}
	case "lte":
		return ValidationField{Code: "LTE", Message: "must be less than or equal to " + param, Params: map[string]any{"value": parseParamInt(param)}}
	case "oneof":
		options := strings.Fields(param)
		return ValidationField{
			Code:    "INVALID_OPTION",
			Message: "must be one of: " + strings.Join(options, ", "),
			Params:  map[string]any{"options": options},
		}
	default:
		return ValidationField{Code: "INVALID", Message: "is invalid"}
	}
}

func parseParamInt(param string) any {
	n, err := strconv.Atoi(param)
	if err != nil {
		return param
	}

	return n
}

func validateAuthenticationInput(sl validator.StructLevel) {
	input := sl.Current().Interface().(dtos.AuthenticationInputDto)
	if input.Purpose == domain.OTPPurposeRegistration && input.User == nil {
		sl.ReportError(input.User, "user", "user", "required", "")
	}
}

func validateMinArrayLen(fl validator.FieldLevel) bool {
	field := fl.Field()
	if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
		return false
	}

	param := fl.Param()
	minLen, err := strconv.Atoi(param)
	if err != nil {
		return false
	}

	return field.Len() >= minLen
}
