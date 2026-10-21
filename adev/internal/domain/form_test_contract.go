package domain

import (
	"errors"
	"fmt"
)

// TargetRef is the physical test file: a registry object or a new file.
type TargetRef struct {
	ObjectID string `json:"objectId,omitempty"`
	New      string `json:"new,omitempty"`
	Package  string `json:"package"`
}

// TestScenario is the given-when-then block of one test. Each step is a list of
// texts, so a step names every fact the test needs.
type TestScenario struct {
	Given []Text `json:"given"`
	When  []Text `json:"when"`
	Then  []Text `json:"then"`
}

// TestCase is one test of one task and one contract.
type TestCase struct {
	ID                string       `json:"id"`
	TaskRef           string       `json:"taskRef"`
	ContractRef       string       `json:"contractRef"`
	TargetTest        TargetRef    `json:"targetTest"`
	Kind              string       `json:"kind"` // unit | contract | integration
	Path              PathKind     `json:"path"`
	ExpectedErrorCode string       `json:"expectedErrorCode,omitempty"`
	Scenario          TestScenario `json:"scenario"`
}

// TestCoverage is what the test set covers and what it misses.
type TestCoverage struct {
	ContractsCovered []string `json:"contractsCovered"`
	PathsCovered     []string `json:"pathsCovered"`
	Uncovered        []string `json:"uncovered"`
}

// TestContract is the payload of a test contract form.
type TestContract struct {
	Tests    []TestCase   `json:"tests"`
	Coverage TestCoverage `json:"coverage"`
	Ready    bool         `json:"ready"`
}

// Texts returns the free texts the language checker reads.
func (TestContract) Texts() []Text { return nil }

var knownTestKinds = map[string]bool{"unit": true, "contract": true, "integration": true}

// ValidateTestContract checks the gates of the registry that one document can
// check. The cross-document gates of the registry (a contract or a path of the
// LLD, an error code of the contract) need both documents: they belong to the
// stage layer.
func ValidateTestContract(c TestContract) error {
	if len(c.Tests) == 0 {
		return invalid("tests", "must contain at least one entry")
	}

	var errs []error
	seen := map[string]bool{}
	for i, test := range c.Tests {
		field := fmt.Sprintf("tests[%d]", i)
		switch {
		case test.ID == "":
			errs = append(errs, invalid(field+".id", "is empty"))
		case seen[test.ID]:
			errs = append(errs, invalid(field+".id", fmt.Sprintf("duplicates %q", test.ID)))
		}
		seen[test.ID] = true
		if test.TaskRef == "" || test.ContractRef == "" {
			errs = append(errs, invalid(field, "taskRef and contractRef must be non-empty"))
		}
		if test.TargetTest.ObjectID == "" && test.TargetTest.New == "" {
			errs = append(errs, invalid(field+".targetTest", "needs objectId or new"))
		}
		if !knownTestKinds[test.Kind] {
			errs = append(errs, invalid(field+".kind", fmt.Sprintf("unknown value %q", test.Kind)))
		}
		if len(test.Scenario.Given) == 0 || len(test.Scenario.When) == 0 || len(test.Scenario.Then) == 0 {
			errs = append(errs, invalid(field+".scenario", "given, when and then must be non-empty"))
		}
		switch test.Path {
		case PathHappy:
		case PathNegative:
			if test.ExpectedErrorCode == "" {
				errs = append(errs, invalid(field+".expectedErrorCode", "must be non-empty on a negative path"))
			}
		default:
			errs = append(errs, invalid(field+".path", fmt.Sprintf("unknown value %q", test.Path)))
		}
	}

	// A ready test set has no uncovered path, and a set with no uncovered path is
	// ready: the two answers cannot disagree.
	if c.Ready && len(c.Coverage.Uncovered) > 0 {
		errs = append(errs, invalid("ready", "is true while coverage.uncovered is not empty"))
	}
	if !c.Ready && len(c.Coverage.Uncovered) == 0 {
		errs = append(errs, invalid("coverage.uncovered", "is empty while ready is false"))
	}
	return errors.Join(errs...)
}
