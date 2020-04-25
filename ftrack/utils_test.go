package ftrack

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncodeUriParameters(t *testing.T) {
	assert.Equal(t,
		"id=_id_&username=_username_&apiKey=_apiKey_",
		encodeUriParameters(
			uriParameter{"id", "_id_"},
			uriParameter{"username", "_username_"},
			uriParameter{"apiKey", "_apiKey_"},
		),
	)
}

func TestNormalizeString(t *testing.T) {
	normalized := NormalizeString("Ra\u0308ksmo\u0308rga\u030as")
	assert.Equal(t, "Räksmörgås", normalized, "should normalize COMBINING DIAERESIS")

	normalized = NormalizeString("R\u00e4ksm\u00f6rg\u00e5s")
	assert.Equal(t, "Räksmörgås", normalized, "should not alter combined characters")

	normalized = NormalizeString("Ψ Ω Ϊ Ϋ ά έ ή")
	assert.Equal(t, "Ψ Ω Ϊ Ϋ ά έ ή", normalized, "Should not alter greek characters")
}
