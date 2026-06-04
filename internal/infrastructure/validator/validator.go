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
	v.validate.RegisterStructValidation(
		validateUpdateMatchResult,
		dtos.UpdateMatchResultDto{},
	)
	v.validate.RegisterStructValidation(
		validateCreateCompetition,
		dtos.CreateCompetitionDto{},
	)
	v.validate.RegisterStructValidation(
		validateUpdateUser,
		dtos.UpdateUserDto{},
	)
	v.validate.RegisterValidation("min_array_len", validateMinArrayLen)
	v.validate.RegisterValidation("fifa_code", func(fl validator.FieldLevel) bool {
		return IsValidFifaCode(fl.Field().String())
	})

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
	case "len":
		return ValidationField{Code: "LEN", Message: "must have " + param + " elements", Params: map[string]any{"len": parseParamInt(param)}}
	case "fifa_code":
		return ValidationField{Code: "INVALID_FIFA_CODE", Message: "must be a valid FIFA team code"}
	case "scope_forbidden":
		return ValidationField{Code: "SCOPE_FORBIDDEN", Message: "scope is not allowed for this competition type"}
	case "scope_required":
		return ValidationField{Code: "SCOPE_REQUIRED", Message: "scope with at least one stage is required for this competition type"}
	case "penalty_incomplete":
		return ValidationField{Code: "PENALTY_INCOMPLETE", Message: "penalty score must include both home and away values"}
	case "penalty_tied":
		return ValidationField{Code: "PENALTY_TIED", Message: "penalty score must be decisive: home and away values cannot be equal"}
	case "at_least_one_field":
		return ValidationField{Code: "AT_LEAST_ONE_FIELD", Message: "at least one field must be provided"}
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

func validateUpdateMatchResult(sl validator.StructLevel) {
	input := sl.Current().Interface().(dtos.UpdateMatchResultDto)

	homeSet := input.HomePenaltyScore != nil
	awaySet := input.AwayPenaltyScore != nil

	if homeSet != awaySet {
		sl.ReportError(input.HomePenaltyScore, "home_penalty_score", "HomePenaltyScore", "penalty_incomplete", "")
		sl.ReportError(input.AwayPenaltyScore, "away_penalty_score", "AwayPenaltyScore", "penalty_incomplete", "")
		return
	}

	if homeSet && awaySet && *input.HomePenaltyScore == *input.AwayPenaltyScore {
		sl.ReportError(input.HomePenaltyScore, "home_penalty_score", "HomePenaltyScore", "penalty_tied", "")
		sl.ReportError(input.AwayPenaltyScore, "away_penalty_score", "AwayPenaltyScore", "penalty_tied", "")
	}
}

func validateCreateCompetition(sl validator.StructLevel) {
	input := sl.Current().Interface().(dtos.CreateCompetitionDto)

	switch input.Type {
	case domain.CompetitionTypePickem:
		if input.Scope != nil {
			sl.ReportError(input.Scope, "scope", "Scope", "scope_forbidden", "")
		}
	case domain.CompetitionTypeMatch:
		if input.Scope == nil || len(input.Scope.Stages) == 0 {
			sl.ReportError(input.Scope, "scope", "Scope", "scope_required", "")
		}
	case domain.CompetitionTypePool:
		if input.Scope != nil {
			sl.ReportError(input.Scope, "scope", "Scope", "scope_forbidden", "")
		}
		if input.MatchID == nil {
			sl.ReportError(input.MatchID, "match_id", "MatchID", "required", "")
		}
	}
}

func validateUpdateUser(sl validator.StructLevel) {
	input := sl.Current().Interface().(dtos.UpdateUserDto)

	if input.FirstName == nil && input.LastName == nil && input.Username == nil {
		sl.ReportError(input, "_error", "", "at_least_one_field", "")
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
