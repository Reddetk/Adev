package usecase

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

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

func TestStagePromptKeepsTheQuestionBranch(t *testing.T) {
	stage, err := domain.StageByID(domain.RouteGreenfield, "00")
	assert.NoError(t, err)

	_, user := stagePrompt(stage, domain.TypeBRD, "SHAPE", nil, nil)

	assert.Contains(t, user, "You fill the BRD form of stage 00.")
	assert.Contains(t, user, "openGaps")
	assert.NotContains(t, user, "This is a design stage")
}

func TestStagePromptKeepsTheDesignBranch(t *testing.T) {
	stage, err := domain.StageByID(domain.RouteGreenfield, "03")
	assert.NoError(t, err)

	_, user := stagePrompt(stage, domain.TypeHLD, "SHAPE", nil, nil)

	assert.Contains(t, user, "You fill the HLD form of stage 03.")
	assert.Contains(t, user, "This is a design stage")
}
