package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLastStageMarksTheEndOfTheRoute(t *testing.T) {
	assert.True(t, IsLastStage(RouteGreenfield, "06"), "stage 06 ends the greenfield route")
	assert.False(t, IsLastStage(RouteGreenfield, "05"))
	assert.False(t, IsLastStage(RouteGreenfield, "99"), "an unknown stage ends nothing")
	assert.False(t, IsLastStage(RouteChange, "06"), "a route without stages has no last stage")
}
