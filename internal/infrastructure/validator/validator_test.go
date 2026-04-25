package validator_test

import (
	"testing"

	"github.com/fifawcp/api/internal/domain"
	"github.com/fifawcp/api/internal/dtos"
	"github.com/fifawcp/api/internal/infrastructure/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var v = validator.NewValidator()

func TestValidateStruct_ReturnsNilOnValid(t *testing.T) {
	type testStruct struct {
		Email string   `json:"email" validate:"required,email"`
		Name  string   `json:"name" validate:"required,min=2,max=50"`
		Age   int      `json:"age" validate:"gte=18"`
		Tags  []string `json:"tags" validate:"min_array_len=2"`
	}

	result := v.ValidateStruct(testStruct{
		Email: "user@example.com",
		Name:  "Alice",
		Age:   20,
		Tags:  []string{"a", "b"},
	})

	assert.Nil(t, result)
}

func TestValidateStruct_OmitsFieldsWithDashJsonTag(t *testing.T) {
	type testStruct struct {
		Email string `json:"-"`
	}

	result := v.ValidateStruct(testStruct{Email: "user@example.com"})
	assert.Nil(t, result)
}

func TestValidateStruct_RequiredField(t *testing.T) {
	type testStruct struct {
		Email string `json:"email" validate:"required"`
	}

	result := v.ValidateStruct(testStruct{Email: ""})
	require.NotNil(t, result)
	field, ok := result["email"]

	require.True(t, ok)
	assert.Equal(t, "REQUIRED", field.Code)
	assert.Equal(t, "is required", field.Message)
	assert.Nil(t, field.Params)
}

func TestValidateStruct_InvalidEmail(t *testing.T) {
	type testStruct struct {
		Email string `json:"email" validate:"required,email"`
	}

	result := v.ValidateStruct(testStruct{Email: "not-an-email"})
	require.NotNil(t, result)
	field := result["email"]

	assert.Equal(t, "INVALID_EMAIL", field.Code)
	assert.Equal(t, "must be a valid email address", field.Message)
}

func TestValidateStruct_MinLength(t *testing.T) {
	type testStruct struct {
		Name string `json:"name" validate:"required,min=2"`
	}

	result := v.ValidateStruct(testStruct{Name: "A"})
	require.NotNil(t, result)
	field := result["name"]

	assert.Equal(t, "MIN_LENGTH", field.Code)
	assert.Equal(t, "must be at least 2 characters", field.Message)
	require.NotNil(t, field.Params)
	assert.Equal(t, 2, field.Params["min"])
}

func TestValidateStruct_MaxLength(t *testing.T) {
	type testStruct struct {
		Name string `json:"name" validate:"required,max=50"`
	}

	longName := make([]byte, 51)
	for i := range longName {
		longName[i] = 'a'
	}
	result := v.ValidateStruct(testStruct{Name: string(longName)})
	require.NotNil(t, result)
	field := result["name"]

	assert.Equal(t, "MAX_LENGTH", field.Code)
	assert.Equal(t, "must be at most 50 characters", field.Message)
	require.NotNil(t, field.Params)
	assert.Equal(t, 50, field.Params["max"])
}

func TestValidateStruct_MinArrayLen(t *testing.T) {
	t.Run("returns single element error message when min is 1", func(t *testing.T) {
		type testStruct struct {
			Tags []string `json:"tags" validate:"min_array_len=1"`
		}

		result := v.ValidateStruct(testStruct{Tags: []string{}})
		require.NotNil(t, result)
		field := result["tags"]

		assert.Equal(t, "MIN_ELEMENTS", field.Code)
		assert.Equal(t, "must have at least 1 element", field.Message)
		assert.Equal(t, 1, field.Params["min"])
	})

	t.Run("returns multiple elements error message when min is greater than 1", func(t *testing.T) {
		type testStruct struct {
			Tags []string `json:"tags" validate:"min_array_len=2"`
		}

		result := v.ValidateStruct(testStruct{Tags: []string{"a"}})
		require.NotNil(t, result)
		field := result["tags"]

		assert.Equal(t, "MIN_ELEMENTS", field.Code)
		assert.Equal(t, "must have at least 2 elements", field.Message)
		assert.Equal(t, 2, field.Params["min"])
	})
}

func TestValidateStruct_GT(t *testing.T) {
	type testStruct struct {
		Age int `json:"age" validate:"gt=18"`
	}

	result := v.ValidateStruct(testStruct{Age: 17})
	require.NotNil(t, result)
	field := result["age"]

	assert.Equal(t, "GT", field.Code)
	assert.Equal(t, "must be greater than 18", field.Message)
	assert.Equal(t, 18, field.Params["value"])
}

func TestValidateStruct_LT(t *testing.T) {
	type testStruct struct {
		Age int `json:"age" validate:"lt=20"`
	}

	result := v.ValidateStruct(testStruct{Age: 21})
	require.NotNil(t, result)
	field := result["age"]

	assert.Equal(t, "LT", field.Code)
	assert.Equal(t, "must be less than 20", field.Message)
	assert.Equal(t, 20, field.Params["value"])
}

func TestValidateStruct_GTE(t *testing.T) {
	type testStruct struct {
		Age int `json:"age" validate:"gte=18"`
	}

	result := v.ValidateStruct(testStruct{Age: 17})
	require.NotNil(t, result)
	field := result["age"]

	assert.Equal(t, "GTE", field.Code)
	assert.Equal(t, "must be greater than or equal to 18", field.Message)
	assert.Equal(t, 18, field.Params["value"])
}

func TestValidateStruct_LTE(t *testing.T) {
	type testStruct struct {
		Age int `json:"age" validate:"lte=20"`
	}

	result := v.ValidateStruct(testStruct{Age: 21})
	require.NotNil(t, result)
	field := result["age"]

	assert.Equal(t, "LTE", field.Code)
	assert.Equal(t, "must be less than or equal to 20", field.Message)
	assert.Equal(t, 20, field.Params["value"])
}

func TestValidateStruct_OneOf(t *testing.T) {
	type testStruct struct {
		Status string `json:"status" validate:"oneof=scheduled finished"`
	}

	result := v.ValidateStruct(testStruct{Status: "pending"})
	require.NotNil(t, result)
	field := result["status"]

	assert.Equal(t, "INVALID_OPTION", field.Code)
	assert.Equal(t, "must be one of: scheduled, finished", field.Message)
	assert.Equal(t, []string{"scheduled", "finished"}, field.Params["options"])
}

func TestValidateStruct_Invalid_DefaultCase(t *testing.T) {
	type testStruct struct {
		Name string `json:"name" validate:"alpha"` // built-in validator that exists
	}

	result := v.ValidateStruct(testStruct{Name: "123"}) // fails alpha
	require.NotNil(t, result)
	field, ok := result["name"]

	require.True(t, ok)
	assert.Equal(t, "INVALID", field.Code)
	assert.Equal(t, "is invalid", field.Message)
}

func TestValidateStruct_AuthenticationInput_RegistrationRequiresUser(t *testing.T) {
	dto := dtos.AuthenticationInputDto{
		Identifier: "john@example.com",
		Purpose:    domain.OTPPurposeRegistration,
		OTP:        "123456",
		User:       nil, // invalid for registration
	}

	result := v.ValidateStruct(dto)
	require.NotNil(t, result)
	field, ok := result["user"]
	require.True(t, ok)

	assert.Equal(t, "REQUIRED", field.Code)
	assert.Equal(t, "is required", field.Message)
}

func TestValidateStruct_AuthenticationInput_LoginDoesNotRequireUser(t *testing.T) {
	dto := dtos.AuthenticationInputDto{
		Identifier: "john@example.com",
		Purpose:    domain.OTPPurposeLogin,
		OTP:        "123456",
		User:       nil, // valid for login
	}

	result := v.ValidateStruct(dto)
	assert.Nil(t, result)
}
