package usecase

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

func TestPromptMessagesCarryTheShapeContextAndProblems(t *testing.T) {
	system, user := promptMessages("Instruction.", "SHAPE", json.RawMessage(`{"a":1}`), []string{"a: b"})

	assert.Equal(t,
		"You fill one form of a software project. Answer with JSON only. Do not add fields. "+
			"Do not write code. Use short sentences. The payload shape is: SHAPE",
		system)
	assert.Equal(t,
		"Instruction.\nInput forms:\n{\"a\":1}\nThe previous answer failed these gates. Fix them:\n- a: b\n",
		user)
}

func TestStagePromptAsksForAnUncertainValueOnEveryStage(t *testing.T) {
	for _, id := range []string{"00", "03", "06"} {
		stage, err := domain.StageByID(domain.RouteGreenfield, id)
		require.NoError(t, err)

		_, user := stagePrompt(stage, domain.TypeHLD, "SHAPE", nil, nil)

		assert.Contains(t, user, "You fill the HLD form of stage "+id)
		assert.Contains(t, user, "put one openGaps entry", "stage %s must ask instead of inventing", id)
		assert.Contains(t, user, "the options when it is a choice", "stage %s must offer options", id)
	}
}
