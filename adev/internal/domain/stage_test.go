package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLastStageMarksTheEndOfTheRoute(t *testing.T) {
	assert.True(t, IsLastStage(RouteGreenfield, "09"), "stage 09 ends the greenfield route")
	assert.False(t, IsLastStage(RouteGreenfield, "08"))
	assert.False(t, IsLastStage(RouteGreenfield, "99"), "an unknown stage ends nothing")
	assert.False(t, IsLastStage(RouteChange, "06"), "the change head ends nothing")
	assert.True(t, IsLastStage(RouteChange, "01"), "the impact assessment closes the head route")
}
