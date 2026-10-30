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
		"containers": {1, 2, 9},
		"layers":     {1, 2, 9},
		"packages":   {1, 2, 9},
		"modules":    {1, 2, 9},
		"contracts":  {3, 4, 5, 6, 9},
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
		line = strings.TrimSuffix(line, "\r")
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

// questionsMessages is the call of the operator interview: the model collects
// questions, it does not write the document. The answer shape is the small
// openGaps object, not the whole form.
func questionsMessages(instruction string, context json.RawMessage) (system, user string) {
	system = "You collect the operator questions for one form of a software project. " +
		"Answer with JSON only. Do not write code. Do not fill the document. " +
		`The answer shape is: {"openGaps":[{"id":"...","pointer":"/field/0",` +
		`"question":{"lang":"STE100","text":"..."},"options":["..."],"blocking":false}]}`
	return system, instruction + "\nInput forms:\n" + string(context)
}

// detailMessages is the call of one detail entry: the model answers the meaning
// of one entry and its questions. It reads the whole form but never writes it, so
// the full payload shape stays out of the system prompt.
func detailMessages(formType domain.FormType, instruction string, context, document json.RawMessage) (system, user string) {
	system = fmt.Sprintf(
		"You detail one entry of the %s form of a software project. Answer with JSON only. Do not write code. "+
			"Answer with one object: the key \"entry\" holds the fields of the entry you are given (the same keys), "+
			"and the key \"openGaps\" holds the questions: "+
			`{"entry":{...},"openGaps":[{"id":"...","pointer":"...","question":{"lang":"STE100","text":"..."},"options":["..."],"blocking":false}]}. `+
			"Do not repeat the whole document and do not add fields the entry does not have.",
		formType)
	var builder strings.Builder
	builder.WriteString(instruction)
	builder.WriteString("\nInput forms:\n")
	builder.Write(context)
	if len(document) > 0 {
		builder.WriteString("\nThe form so far (read only, do not repeat it):\n")
		builder.Write(document)
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
	kinds := make([]string, 0, len(batchPlans[output]))
	for _, plan := range batchPlans[output] {
		kinds = append(kinds, plan.kind)
	}
	empty := strings.Join(collectEmptyLists(batchPlans[output], map[string]bool{}), ", ")
	detail := "Leave every detailed list empty ([])."
	if empty != "" {
		detail = fmt.Sprintf("Leave the detailed lists empty ([]): %s.", empty)
	}
	base := fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with the payload only. Do not add fields. "+
			"List every entry of %s with its id and short fields: one entry for each item the input names. "+
			"An empty %s list is wrong when the input names items. "+
			"%s "+
			"Fill every field that the given context answers. "+
			"If the context does not give a concrete value or you are not sure, do not invent it: "+
			"put one openGaps entry with the question, the field it would fill (one value, never a whole list), "+
			"and the options when it is a choice.",
		output, stage.ID, strings.Join(kinds, " and "), strings.Join(kinds, " and "), detail)
	if output == domain.TypeSRS {
		base += " Write one fr entry for EVERY step of the real user flow of EVERY actor in the BRD: the sign-up or the sign-in from actors[].tasks[1] (registration, authentication, password reset), the cabinet action from actors[].tasks[2], one fr for each remaining task in actors[].tasks, one fr for the payment the BRD assumptions name (charge, refund, receipt), and one fr for the notification the BRD assumptions name. " +
			"The statement of such an fr names that actor as the subject and the business object as the object. " +
			"Write one nfr entry for every measurable quality the BRD constraints or the context name. " +
			"Do not return empty fr and nfr lists. outOfScope lists only behaviour this SRS excludes: leave it empty when nothing is excluded."
	}
	if output == domain.TypeHLD {
		base += " For HLD: fill ubiquitousLanguage with every glossary term of the BRD (term, definition, sourceRef). " +
			"Fill semanticAnalysis with every BRD term that carries a different meaning in two parts of the business (a shipment for the client is a delivery for the courier): name the contexts, the difference in attributes and lifecycle, and let the result choose the boundedContexts. " +
			"List the decisions this design makes, one entry per fork: the storage product, the message bus, the consistency model, the failure handling, the edge and the identity provider. Give each decision its id and title now and leave alternatives and consequences to the detail calls. A decision names in objectRefs the containers it fixes, and in techs the products it chooses: a product must be the tech of a container of this design, so a database or a cache the system needs becomes a container of its own. " +
			"List every external system or physical device the BRD assumptions name (payment terminal, card reader, email, accounting) as a container with tech 'external', and connect it to the containers it serves with a contextRelation. " +
			"Add the infrastructure containers the distributed system needs, each as its own container with its own boundedContext and its own purpose: exactly ONE reverse proxy or API Gateway container for the whole system at the edge, in front of every client container, and never one gateway per role; the identity provider (Keycloak) when the BRD has a sign-up or a sign-in; the message bus (Kafka) when an operation runs asynchronously; exactly ONE load balancer container in front of the API gateway, so no single instance is a point of failure; and a scheduler for the notifications and the timed status changes. " +
			"Do not fold an infrastructure component into the application container: the gateway, the provider and the bus are deployable containers with their own tech and their own degradation rule. " +
			"Wire every one of them, because a container without a relation is a box the system does not use: a sync relation from each client container to the identity provider for the sign-in and the token, and for every asynchronous step a sync relation from the publisher to the message bus and a sync relation from the bus to the consumer, with eventRef that names the event. " +
			"Fill boundedContexts (one entry per container), the contextRelations between them, and a degradationStrategy with one entry for every infrastructure container {failure, degradation, detection, timeoutMs}: the name of what fails, what the system does instead, how it detects the failure, and the timeout in milliseconds. A system with an availability NFR needs a degradation rule for the identity provider, the message bus and the gateway, not only for the external devices."
	}
	if output == domain.TypeFeasibility {
		base += " For FEASIBILITY: list at least two options with an id and a short summary. Leave cost, scheduleDays and confidence to the detail calls. At the root set recommendedOption to the id of the option you choose, verdict to go, revise or no_go, and reason to one sentence. Do not ask the operator for a currency, a cost or a confidence. Fill risks with at least one technical or economic risk. scopeDelta records only the scope an option changes: leave it empty when the option adds or removes nothing."
	}
	if output == domain.TypeADR {
		base += " For ADR: list one decision for EVERY architecture area, in this order, because a later decision rests on an earlier one: " +
			"1. architecture style (layered, modular monolith, hexagonal, microservices, event-driven, serverless, cell-based); " +
			"2. communication (REST, gRPC, GraphQL, message queue, pub/sub topic, event stream, webhook, WebSocket, server-sent event, long polling); " +
			"3. data storage (relational, key-value, document, wide-column, graph, object storage); " +
			"4. scaling (load balancing, caching layer, replication, sharding, autoscaling); " +
			"5. consistency (the CAP/PACELC trade-off and strong versus eventual consistency, decided after the storage and the scaling); " +
			"6. reliability (rate limit, circuit breaker, retry with idempotency); " +
			"7. build stack (database, cache, message bus, gateway, search, runtime); " +
			"8. operations and security (observability, load and chaos test, delivery, OAuth 2.0/OIDC, TLS, WAF, secrets vault); " +
			"9. design process (the requirement and the NFR the decision serves, the traffic and storage estimate); " +
			"10. success metrics (p99 latency, availability, throughput, error rate). " +
			"Each decision names the option this system takes and at least two options it rejects; the rejection cites the SRS requirement, the SRS NFR or the constraint the HLD names, and names it by its text or its id. " +
			"The ADR reads the HLD and the SRS, not the BRD: never cite a BRD goal or a BRD constraint, and never cite a rule the input forms do not hold. " +
			"The decision must match the containers the HLD lists: a modular monolith with separate deployed services is a contradiction, so state the style the HLD shows and name the deployment it needs. " +
			"Leave context, decision, alternatives and consequences to the detail calls."
	}
	if output == domain.TypeTestContract {
		base += " For TEST_CONTRACT: list every test with id and kind (unit, contract or integration). Leave the scenario to the detail calls. Fill coverage {contractsCovered, pathsCovered, uncovered} and ready at the root."
	}
	if output == domain.TypeBRD {
		base += " For BRD: metrics[] is optional. goals[].metricId and successMetrics[].metricId may be empty; when given, they MUST reference an ID that exists in metrics[]. Do not invent metric IDs, and do not invent a baseline for a system that does not run yet. scope.out and scope.deferred stay empty unless the request or an answer excludes or defers something."
	}
	if policy := stageQuestionPolicy(output); policy != "" {
		base += " " + policy
	}
	return base
}

// questionsInstruction asks the first call of the BRD for the operator questions
// only: the document is assembled later, per answer, and the model never invents
// a value to fill it. The interview follows the business of the request, not the
// fields of the form: every role, capability and limit becomes one question, so
// the answers describe the subject instead of a skeleton.
func questionsInstruction(stage domain.Stage, output domain.FormType) string {
	if output != domain.TypeBRD {
		return fmt.Sprintf(
			"You collect the operator questions for the %s form of stage %s. Answer with the payload only. Do not add fields. "+
				"Put one openGaps entry at the answer root for each value the context does not give. "+
				"Every question is about THIS system and its business: name the objects, actors and numbers of the request. "+
				"A question that fits any other system the same way is wrong: throw it away. "+
				"At most 8 questions. "+
				"Every entry carries the pointer of the field it would fill, "+
				"and the options when the answer is one of a few choices. "+
				"Do not invent any value.",
			output, stage.ID)
	}
	return fmt.Sprintf(
		"You interview the operator about the business of stage %s before the %s is written. "+
			"Answer with the payload only and put openGaps entries only. "+
			"The interview follows the subject of the request, not the fields of the form. "+
			"An answer with an empty openGaps list is always wrong: the request never states the purpose in BRD words, never states the success value, and never describes the user flow of a role. "+
			"Keep the skeleton lists of the shape in the answer: an empty list stays [] so an answer can append to it. "+
			"Ask these questions, in this order, one gap for each value: "+
			"1. The purpose of the business: /goals/0/outcome. "+
			"2. One gap for EVERY role the request names: /actors/0/name, /actors/1/name, and so on. "+
			"Never merge two roles into one answer and never ask for a role the request already names. "+
			"3. For every role, its main task: /actors/<i>/tasks/0, and the business object it changes: /actors/<i>/businessObject. "+
			"4. For every role, its user flow in the real world: how the role gets into the system, registers or signs in, and what the role presents to prove who it is: /actors/<i>/tasks/1. "+
			"What the role does in its personal cabinet after the sign-in, which records it sees there and which it changes: /actors/<i>/tasks/2. "+
			"Ask these two gaps for EVERY role, because the role of the request is not a database table: it is a person who signs in and works in the system. "+
			"5. The payment the business takes: /assumptions/0 — who pays, when the money moves, and how the money arrives (a card at the door, a card in the cabinet, an invoice, cash). "+
			"6. The notification the business sends: /assumptions/1 — which event the role must learn about, on which channel it arrives (email, SMS, push, telephone call) and who receives it. "+
			"7. How the business does this work today, before the system: /assumptions/2 — who does what now, on paper, by telephone or in a spreadsheet, and where the work gets lost. "+
			"8. One gap for EVERY business word the request uses without defining it: /glossary/0, /glossary/1, and so on. "+
			"Do not ask for a word the request defines. The question names that word of the request and asks what it means in this business, in the real world. "+
			"9. One gap for EVERY external system or physical device the business uses, such as a payment terminal, a card reader, an email service or accounting: /assumptions/3, /assumptions/4, and so on. "+
			"The question names the device the request mentions, for example the card the guest presents at the reception, and asks what the system may read from it. "+
			"10. One gap for EVERY capability inside the project boundary that the request does not already name as an operation: /scope/in/0, /scope/in/1, and so on. "+
			"One capability per entry, never a list in one entry. A capability the request already names as an operation needs no question. "+
			"11. One gap for a law, regulation, budget or deadline that limits the solution: /constraints/0. "+
			"Ask only for a limit the request and the answers do not give; a business with no limit gets no question. "+
			"Do not put a gap under /metrics or /successMetrics: a system that does not run yet has no measured value. "+
			"Keep both lists empty. A success measure the operator names goes into the goal outcome text. "+
			"Use the nouns of the request: a question that names no word of the request is wrong, throw it away. "+
			"Every question repeats the word of the request it asks about, so the question for one role never reads like the question for the next one: "+
			"\"Which name does the receptionist have in this business?\" asks about a receptionist, not about a field. "+
			"At most 24 gaps in this answer; the rest follow after the answers. Answer the whole list above for the roles first, then the payment, the notification, the words, the devices, the capabilities and the limit. "+
			"Every gap names its concrete pointer, gives options when the answer is one of a few choices, and sets blocking false. "+
			"One gap looks like this: {\"id\":\"GAP-1\",\"pointer\":\"/actors/0/name\",\"blocking\":false,\"options\":[],\"question\":{\"lang\":\"STE100\",\"text\":\"Which name does the receptionist have in this business?\"}}. "+
			"Do not ask for an id, an internal name, a unit or a baseline: the operator cannot decide those. "+
			"A question never contains the characters < or >: they mark a template, not a question. "+
			"Replace every placeholder with the real word of the request: the request says the client, the courier and the administrator, so the question says the client, the courier and the administrator. "+
			"Do not invent any value and do not ask the same datum twice.",
		stage.ID, output) + " " + stageQuestionPolicy(output)
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
	base := fmt.Sprintf(
		"You fill the %s form of stage %s. Answer with the payload only. Do not add fields. "+
			"Fill every field that the given context answers. "+
			"If the context does not give a concrete value or you are not sure, do not invent it: "+
			"put one openGaps entry with the question, the field it would fill (one value, never a whole list), "+
			"and the options when it is a choice.",
		output, stage.ID)
	if output == domain.TypeBRD {
		base += " For BRD: one actor entry per role. Split a comma or \"and\" list into separate entries; never put two roles in one name. " +
			"One capability per scope.in entry, split a list the same way. " +
			"A constraint is a Statement with text and modality \"shall\"; when the operator says no or none, keep the list empty instead of writing the word. " +
			"metrics[] is optional. goals[].metricId and successMetrics[].metricId may be empty; when given, they MUST reference an ID that exists in metrics[]. " +
			"Do not invent metric IDs, and do not invent a baseline for a system that does not run yet. " +
			"Keep metrics[] and successMetrics[] empty unless the context or the operator gives a measured value."
	}
	if policy := stageQuestionPolicy(output); policy != "" {
		base += " " + policy
	}
	if output == domain.TypeFeasibility {
		base += " For FEASIBILITY you write a techno-economic justification, not a questionnaire. " +
			"Propose at least two options that satisfy the BRD goals: for example build in-house, buy a ready product, extend the current system. " +
			"Each option has an id, a summary, scheduleDays, a cost {currency, amount} and a confidence of low, medium or high. " +
			"Estimate the cost and the schedule yourself from the scope of the BRD; use low confidence when the data is thin. " +
			"Set recommendedOption to the id of the option you choose, and set verdict to go when it is feasible, revise only when you also give scopeDelta, no_go only when no option can satisfy the goals. " +
			"Do not ask the operator for an option id, a currency, a cost amount, a schedule, a confidence, a recommendedOption or a verdict: those are your decision. " +
			"Ask nothing unless the BRD is missing a limit that changes the recommendation."
	}
	return base
}
