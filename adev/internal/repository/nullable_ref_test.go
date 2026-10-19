package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Reddetk/Adev/internal/domain"
)

// A nil reference must become SQL NULL. The typed nil pointer used to be encoded as
// the JSON literal null, which a read turned back into an empty reference. That is
// why a first form version showed "previousFormVersion" full of zero values.
func TestOptionalJSONEncodesANilReferenceAsNull(t *testing.T) {
	var absent *domain.Ref

	raw, err := optionalJSON(absent)
	require.NoError(t, err)
	assert.Nil(t, raw)

	raw, err = optionalJSON(nil)
	require.NoError(t, err)
	assert.Nil(t, raw)

	raw, err = optionalJSON(&domain.Ref{FormID: "FRM-1", FormVersion: 1})
	require.NoError(t, err)
	assert.JSONEq(t, `{"formId":"FRM-1","formType":"","formVersion":1}`, string(raw))
}

func TestDecodeOptionalRefTreatsNullAsAbsent(t *testing.T) {
	ref, err := decodeOptionalRef(nil)
	require.NoError(t, err)
	assert.Nil(t, ref)

	ref, err = decodeOptionalRef([]byte("null"))
	require.NoError(t, err)
	assert.Nil(t, ref, "rows written before the fix stored the literal null")

	ref, err = decodeOptionalRef([]byte(`{"formId":"FRM-1","formType":"BRD","formVersion":2}`))
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, 2, ref.FormVersion)

	_, err = decodeOptionalRef([]byte(`{`))
	assert.Error(t, err)
}
