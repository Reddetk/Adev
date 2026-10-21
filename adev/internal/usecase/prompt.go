package usecase

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

// promptMessages composes the two messages of one model call. The adapter only
// transports them; what an agent sees is decided here.
func promptMessages(instruction, shape string, context json.RawMessage, problems []string) (system, user string) {
	system = "You fill one form of a software project. Answer with JSON only. " +
		"Do not add fields. Do not write code. Use short sentences. " +
		"The payload shape is: " + shape

	var builder strings.Builder
	builder.WriteString(instruction)
	builder.WriteString("\nInput forms:\n")
	builder.Write(context)
	if len(problems) > 0 {
		builder.WriteString("\nThe previous answer failed these gates. Fix them:\n")
		for _, problem := range problems {
			builder.WriteString("- ")
			builder.WriteString(problem)
			builder.WriteString("\n")
		}
	}
	return system, builder.String()
}

// stagePrompt is the pair of messages of one stage output attempt.
func stagePrompt(stage domain.Stage, output domain.FormType, shape string, context json.RawMessage, problems []string) (system, user string) {
	return promptMessages(stageInstruction(stage, output), shape, context, problems)
}

func stageInstruction(stage domain.Stage, output domain.FormType) string {
	head := fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with the payload only. Do not add fields. ",
		output, stage.ID)
	// The question phase of the stages 00-02: the agent asks, the operator
	// decides, the answer compiles the document. The model never invents a
	// value the context does not give.
	if stage.ID == "00" || stage.ID == "01" || stage.ID == "02" {
		return head +
			"Fill every field that the given context answers. " +
			"For every question the context does not answer, put one openGaps entry: " +
			"the question, the pointer of the field it would fill, and the options when it is a choice. " +
			"Do not invent a value that the context does not give."
	}
	// The design stages 03-06: the design values are the model's own work, so it
	// fills the shape itself and keeps questions for an outside fact.
	return head +
		"This is a design stage: the design values are yours to choose, so fill every field of the shape. " +
		"Ask the operator only for an outside fact the context does not give: put one openGaps entry " +
		"with the question and the pointer of the field it would fill."
}
