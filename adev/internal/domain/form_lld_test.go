package domain

import (
	"strings"
	"testing"
)

// TestLLDPointsTheMissingPathAtTheEndOfTheList: an error code without a negative
// path is a question for the operator, and the finding must name the slot the
// answer lands in. A finding against the contract entry left the operator nowhere
// to write the path.
func TestLLDPointsTheMissingPathAtTheEndOfTheList(t *testing.T) {
	lld := LLD{
		Contracts: []Contract{{
			ID: "contract-1", Kind: "http", InputType: "Book", OutputType: "Book",
			Errors: []ContractError{{Code: "E-1", Condition: "the book is missing"}},
		}},
		Paths: []Path{{
			ID: "path-1", ContractRef: "contract-1", Kind: PathHappy,
			Given: "a book", When: "it is lent", Then: "the loan is stored",
		}},
	}

	err := ValidateLLD(lld)
	if err == nil || !strings.Contains(err.Error(), "paths/-") {
		t.Fatalf("want the finding to name paths/-, got %v", err)
	}
	if !Patchable(TypeLLD, []byte(`{"paths":[]}`), "/paths/-") {
		t.Fatal("the answer must land past the end of the list")
	}
}

// TestLLDRefusesAnEmptyContainerAndAnEmptyLayer: a container without a layer and a
// layer without a package carry no code, so the document is a structural hole and
// the next run fills it. The gate must name the hole, or the stage accepts a layout
// that has nothing in it.
func TestLLDRefusesAnEmptyContainerAndAnEmptyLayer(t *testing.T) {
	lld := LLD{Containers: []LLDContainer{
		{ContainerID: "C-1"},
		{ContainerID: "C-2", Layers: []Layer{{Name: LayerApp}}},
	}}

	err := ValidateLLD(lld)
	if err == nil {
		t.Fatal("want the empty container and the empty layer to fail the gate")
	}
	for _, field := range []string{"containers[0].layers", "containers[1].layers[0].packages"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("want the finding to name %s, got %v", field, err)
		}
	}
}
