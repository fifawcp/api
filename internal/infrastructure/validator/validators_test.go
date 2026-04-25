package validator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsValidGroupCode(t *testing.T) {
	assert.True(t, IsValidGroupCode("A"))
	assert.True(t, IsValidGroupCode("B"))
	assert.True(t, IsValidGroupCode("C"))
	assert.True(t, IsValidGroupCode("D"))
	assert.True(t, IsValidGroupCode("E"))
	assert.True(t, IsValidGroupCode("F"))
	assert.True(t, IsValidGroupCode("G"))
	assert.True(t, IsValidGroupCode("H"))
	assert.True(t, IsValidGroupCode("I"))
	assert.True(t, IsValidGroupCode("J"))
	assert.True(t, IsValidGroupCode("K"))
	assert.True(t, IsValidGroupCode("L"))
	assert.False(t, IsValidGroupCode("M"))
	assert.False(t, IsValidGroupCode("Z"))
}

func TestIsValidStageCode(t *testing.T) {
	assert.True(t, IsValidStageCode("group_stage"))
	assert.True(t, IsValidStageCode("round_of_32"))
	assert.True(t, IsValidStageCode("round_of_16"))
	assert.True(t, IsValidStageCode("quarterfinals"))
	assert.True(t, IsValidStageCode("semifinals"))
	assert.True(t, IsValidStageCode("third_place"))
	assert.True(t, IsValidStageCode("final"))
}

func TestIsValidStatus(t *testing.T) {
	assert.True(t, IsValidStatus("scheduled"))
	assert.True(t, IsValidStatus("finished"))
	assert.False(t, IsValidStatus("pending"))
}

func TestIsValidFifaCode(t *testing.T) {
	assert.True(t, IsValidFifaCode("MEX"))
	assert.False(t, IsValidFifaCode("ZZZ"))
}

func TestIsValidDateRange(t *testing.T) {
	now := time.Now().UTC()
	tomorrow := now.Add(24 * time.Hour)

	assert.True(t, IsValidDateRange(nil, nil))
	assert.True(t, IsValidDateRange(&now, &tomorrow))
	assert.False(t, IsValidDateRange(&tomorrow, &now))
	assert.True(t, IsValidDateRange(&now, nil))
	assert.True(t, IsValidDateRange(nil, &tomorrow))
}
