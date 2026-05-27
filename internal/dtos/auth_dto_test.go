package dtos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestOtpDto_Normalize(t *testing.T) {
	t.Parallel()

	t.Run("normalize email to lowercase", func(t *testing.T) {
		t.Parallel()

		dto := &RequestOtpDto{
			Identifier: "  JOHN.DOE@EXAMPLE.COM  ",
		}

		dto.Normalize()

		assert.Equal(t, "john.doe@example.com", dto.Identifier)
	})

	t.Run("normalize username by trimming whitespace", func(t *testing.T) {
		t.Parallel()

		dto := &RequestOtpDto{
			Identifier: "  john_doe  ",
		}

		dto.Normalize()

		assert.Equal(t, "john_doe", dto.Identifier)
	})
}

func TestAuthenticationInputDto_Normalize(t *testing.T) {
	t.Parallel()

	t.Run("normalize email to lowercase", func(t *testing.T) {
		t.Parallel()

		dto := &AuthenticationInputDto{
			Identifier: "  JOHN.DOE@EXAMPLE.COM  ",
		}

		dto.Normalize()

		assert.Equal(t, "john.doe@example.com", dto.Identifier)
	})

	t.Run("normalize username by trimming whitespace", func(t *testing.T) {
		t.Parallel()

		dto := &AuthenticationInputDto{
			Identifier: "  john_doe  ",
		}

		dto.Normalize()

		assert.Equal(t, "john_doe", dto.Identifier)
	})
}

func TestIsEmail(t *testing.T) {
	t.Parallel()

	t.Run("is email", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isEmail("john.doe@example.com"))
	})

	t.Run("is not email", func(t *testing.T) {
		t.Parallel()

		assert.False(t, isEmail("john_doe"))
	})
}
