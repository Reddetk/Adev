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
	instruction := stageInstruction(stage, output)
	if _, batched := batchPlans[output]; batched {
		instruction = skeletonInstruction(stage, output)
	}
	return promptMessages(instruction, shape, context, problems)
}

// skeletonInstruction asks the first call of a batched document for its outline:
// every entry with its id and short fields, the detailed lists empty. Each entry
// gets its own call afterwards, so no single answer carries the whole document.
func skeletonInstruction(stage domain.Stage, output domain.FormType) string {
	return fmt.Sprintf(
		"You fill the outline of the %s form of stage %s. Answer with the payload only. "+
			"List every entry with its id and short fields. Leave the detailed lists empty ([]): "+
			"each entry is detailed by its own call. "+
			"If the context does not give a concrete value, put one openGaps entry with the question and the pointer.",
		output, stage.ID)
}

// stageInstruction is one rule for every stage: fill what the context answers,
// and turn every uncertain or missing value into an operator question with
// options. A design stage differs in what the context gives, not in whether it
// may ask.
func stageInstruction(stage domain.Stage, output domain.FormType) string {
	return fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with the payload only. Do not add fields. "+
			"Fill every field that the given context answers. "+
			"If the context does not give a concrete value or you are not sure, do not invent it: "+
			"put one openGaps entry with the question, the pointer of the field it would fill, "+
			"and the options when it is a choice.",
		output, stage.ID)
}
