package usecase

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Reddetk/Adev/internal/domain"
)

// registryDir is where the stage prompts take the form specs from. The
// repository root keeps the source of truth beside the Go module; make run
// starts in adev/, so ../form-registry is the mounted folder.
const registryDir = "../form-registry"

// maxFileBytes keeps one attachment inside a sane context budget. A bigger
// file still ships, but truncated, so the caller sees the cut in the prompt.
const maxFileBytes = 256 << 10

// registryFileName maps a form type onto its registry page name: the type
// constants in kebab-case, so CR_BRIEF becomes cr-brief.md.
func registryFileName(formType domain.FormType) string {
	return strings.ToLower(strings.ReplaceAll(string(formType), "_", "-")) + ".md"
}

// registryFile reads one source-of-truth markdown of the form registry. A
// missing file means a form without a registry page: the call ships without it.
func registryFile(formType domain.FormType) (LLMFile, bool) {
	name := registryFileName(formType)
	raw, err := os.ReadFile(filepath.Join(registryDir, name))
	if err != nil {
		log.Printf("form registry: %s is missing: the %s call ships without it", name, formType)
		return LLMFile{}, false
	}
	content := string(raw)
	if len(content) > maxFileBytes {
		content = content[:maxFileBytes] + "\n... (truncated at 256 KiB)"
	}
	return LLMFile{Name: name, Content: content}, true
}

// registryFiles returns the registry page of one form type, if it exists.
func registryFiles(formType domain.FormType) []LLMFile {
	file, ok := registryFile(formType)
	if !ok {
		return nil
	}
	return []LLMFile{file}
}

// registryParagraphs names the template paragraphs one detail call reads: the
// paragraphs of its own list and the gates. The other paragraphs describe lists
// that this call does not touch.
// shortcut: only the HLD and the LLD are mapped; map the other batched pages when
// their detail calls get slow.
var registryParagraphs = map[domain.FormType]map[string][]int{
	domain.TypeHLD: {
		"containers": {3, 4, 6},
	},
	domain.TypeLLD: {
		"containers": {1, 2},
		"layers":     {1, 2},
		"packages":   {1, 2},
		"modules":    {1, 2},
		"contracts":  {3, 4, 5, 6},
	},
}

// registryFileFor returns the registry page of one detail call: the paragraphs
// of its list and the gates. A page without the numbered template ships whole.
func registryFileFor(formType domain.FormType, kind string) []LLMFile {
	file, ok := registryFile(formType)
	if !ok {
		return nil
	}
	paragraphs, mapped := registryParagraphs[formType][kind]
	if !mapped {
		return []LLMFile{file}
	}
	focused, ok := keepSections(file.Content, paragraphs)
	if !ok {
		return []LLMFile{file}
	}
	return []LLMFile{{Name: file.Name, Content: focused}}
}

// keepSections keeps the numbered template paragraphs and the gates of a
// registry page: "### Paragraph 3 — Contract" stays for 3, "## Gates" stays
// always. A page whose paragraphs are not found ships whole.
func keepSections(page string, paragraphs []int) (string, bool) {
	keep := map[string]bool{"## Gates": true}
	for _, number := range paragraphs {
		keep[fmt.Sprintf("### Paragraph %d", number)] = true
	}
	var out, section strings.Builder
	kept, found := false, false
	flush := func() {
		if kept {
			out.WriteString(section.String())
		}
		section.Reset()
	}
	for _, line := range strings.Split(page, "\n") {
		if strings.HasPrefix(line, "### ") || strings.HasPrefix(line, "## ") {
			flush()
			key := headingKey(line)
			kept = keep[key]
			if kept && key != "## Gates" {
				found = true
			}
		}
		section.WriteString(line)
		section.WriteString("\n")
	}
	flush()
	if !found {
		return page, false
	}
	return out.String(), true
}

// headingKey drops the title of a heading: "### Paragraph 3 - Contract" becomes
// "### Paragraph 3".
func headingKey(line string) string {
	for _, separator := range []string{" - ", " — "} {
		if prefix, _, ok := strings.Cut(line, separator); ok {
			return prefix
		}
	}
	return line
}

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
	empty := strings.Join(collectEmptyLists(batchPlans[output], map[string]bool{}), ", ")
	return fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with the payload only. Do not add fields. "+
			"List every entry with its id and short fields. Leave the detailed lists empty ([]): detailed lists: %s. "+
			"Fill every field that the given context answers. "+
			"If the context does not give a concrete value or you are not sure, do not invent it: "+
			"put one openGaps entry with the question, the field it would fill (one value, never a whole list), "+
			"and the options when it is a choice.",
		output, stage.ID, empty)
}

// questionsInstruction asks the first call of the BRD for the operator questions
// only: the document is assembled later, per answer, and the model never invents
// a value to fill it.
func questionsInstruction(stage domain.Stage, output domain.FormType) string {
	return fmt.Sprintf(
		"You collect the operator questions for the %s form of stage %s. Answer with the payload only. Do not add fields. "+
			"Produce the openGaps entries only: one per value the context does not give. "+
			"Every question is about THIS system and its business: name the objects, actors and numbers of the request. "+
			"A question that fits any other system the same way is wrong: throw it away. "+
			"At most 8 questions. "+
			"Every entry carries the pointer of the field it would fill, "+
			"and the options when the answer is one of a few choices: "+
			"scope entries take the options in, out and deferred; yes and no questions take their options; "+
			"periods and units take their choices. "+
			"Write the lists that the context already answers fully. Leave every other list empty ([]). "+
			"Do not invent any value.",
		output, stage.ID)
}

// collectEmptyLists gathers the lists that later detail calls fill: the companion
// lists and, level by level, the children of every plan.
func collectEmptyLists(plans []batchPlan, seen map[string]bool) []string {
	var out []string
	for _, plan := range plans {
		if plan.with != "" && !seen[plan.with] {
			seen[plan.with] = true
			out = append(out, plan.with)
		}
		for _, child := range plan.children {
			if !seen[child.kind] {
				seen[child.kind] = true
				out = append(out, child.kind)
			}
			out = append(out, collectEmptyLists([]batchPlan{child}, seen)...)
		}
	}
	return out
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
			"put one openGaps entry with the question, the field it would fill (one value, never a whole list), "+
			"and the options when it is a choice.",
		output, stage.ID)
}
