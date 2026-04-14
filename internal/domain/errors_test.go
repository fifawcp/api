package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOtpCooldownError(t *testing.T) {
	t.Parallel()

	t.Run("return dynamic error message when cooldown is set", func(t *testing.T) {
		t.Parallel()

		cooldown := time.Duration(5) * time.Minute
		err := ErrOtpCooldown(cooldown)

		assert.Equal(t, "please wait 5m0s before requesting a new code", err.Error())
	})
}
