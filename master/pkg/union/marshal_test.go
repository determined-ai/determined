package union

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

func TestMarshalOmitEmpty(t *testing.T) {
	type union struct {
		OptionA         *struct{} `union:"type,a" json:"-"`
		OptionB         *struct{} `union:"type,b" json:"-"`
		Regular         *string   `json:"regular"`
		ShouldBeOmitted *string   `json:"shouldBeOmitted,omitempty"`
		DontBeOmitted   *string   `json:"dontBeOmitted,omitempty"`
	}

	out, err := Marshal(union{OptionA: &struct{}{}, DontBeOmitted: ptrs.Ptr("test")})
	require.NoError(t, err, "marshal no error")
	require.Equal(t, string(out), `{"dontBeOmitted":"test","regular":null,"type":"a"}`,
		"incorrect unmarshaling")

	type badUnion struct {
		OptionA *struct{} `union:"type,a" json:"-"`
		OptionB *struct{} `union:"type,b" json:"-"`
		BadType *string   `json:"badType,string"`
	}
	_, err = Marshal(badUnion{OptionB: &struct{}{}, BadType: ptrs.Ptr("bad")})
	require.ErrorContains(t, err, "features not support")
}
