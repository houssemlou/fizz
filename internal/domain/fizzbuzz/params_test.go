package fizzbuzz_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
)

func TestParamsValidate(t *testing.T) {
	base := fizzbuzz.Params{Int1: 3, Int2: 5, Limit: 100, Str1: "fizz", Str2: "buzz"}

	cases := []struct {
		name    string
		mutate  func(*fizzbuzz.Params)
		wantErr bool
	}{
		{"valid", func(p *fizzbuzz.Params) {}, false},
		{"int1 zero", func(p *fizzbuzz.Params) { p.Int1 = 0 }, true},
		{"int1 negative", func(p *fizzbuzz.Params) { p.Int1 = -1 }, true},
		{"int2 zero", func(p *fizzbuzz.Params) { p.Int2 = 0 }, true},
		{"limit zero", func(p *fizzbuzz.Params) { p.Limit = 0 }, true},
		{"limit exceeds max", func(p *fizzbuzz.Params) { p.Limit = 1_001 }, true},
		{"str1 empty", func(p *fizzbuzz.Params) { p.Str1 = "" }, true},
		{"str2 empty", func(p *fizzbuzz.Params) { p.Str2 = "" }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base
			tc.mutate(&p)
			err := p.Validate()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
