package fizzbuzz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/houssemlou/fizz/internal/domain/fizzbuzz"
)

type stubRecorder struct {
	err   error
	calls int
}

func (r *stubRecorder) Record(_ context.Context, _ fizzbuzz.Params) error {
	r.calls++
	return r.err
}

type stubRepo struct{}

func (r *stubRepo) Record(_ context.Context, _ fizzbuzz.Params, _ string) error { return nil }
func (r *stubRepo) Top(_ context.Context) (*fizzbuzz.TopResult, error)          { return nil, nil }

func newService(rec *stubRecorder) *fizzbuzz.Service {
	return fizzbuzz.NewService(rec, &stubRepo{})
}

func TestGenerate_Logic(t *testing.T) {
	tests := []struct {
		name   string
		params fizzbuzz.Params
		want   []string
	}{
		{
			name:   "classic fizzbuzz to 15",
			params: fizzbuzz.Params{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"},
			want: []string{
				"1", "2", "fizz", "4", "buzz",
				"fizz", "7", "8", "fizz", "buzz",
				"11", "fizz", "13", "14", "fizzbuzz",
			},
		},
		{
			name:   "custom strings",
			params: fizzbuzz.Params{Int1: 2, Int2: 7, Limit: 14, Str1: "foo", Str2: "bar"},
			want: []string{
				"1", "foo", "3", "foo", "5", "foo",
				"bar", "foo", "9", "foo", "11", "foo",
				"13", "foobar",
			},
		},
		{
			name:   "limit of 1",
			params: fizzbuzz.Params{Int1: 3, Int2: 5, Limit: 1, Str1: "fizz", Str2: "buzz"},
			want:   []string{"1"},
		},
		{
			name:   "int1 equals int2",
			params: fizzbuzz.Params{Int1: 3, Int2: 3, Limit: 6, Str1: "a", Str2: "b"},
			want:   []string{"1", "2", "ab", "4", "5", "ab"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newService(&stubRecorder{}).Generate(context.Background(), tc.params)
			assert.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestGenerate_InvalidParams(t *testing.T) {
	_, err := newService(&stubRecorder{}).Generate(context.Background(), fizzbuzz.Params{})
	assert.Error(t, err)
}

func TestGenerate_CallsRecorder(t *testing.T) {
	p := fizzbuzz.Params{Int1: 3, Int2: 5, Limit: 15, Str1: "fizz", Str2: "buzz"}
	rec := &stubRecorder{}

	result, err := newService(rec).Generate(context.Background(), p)

	assert.NoError(t, err)
	assert.Len(t, result, p.Limit)
	assert.Equal(t, 1, rec.calls)
}

func TestGenerate_RecorderError_StillReturns(t *testing.T) {
	p := fizzbuzz.Params{Int1: 3, Int2: 5, Limit: 3, Str1: "fizz", Str2: "buzz"}
	rec := &stubRecorder{err: errors.New("kafka down")}

	result, err := newService(rec).Generate(context.Background(), p)

	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2", "fizz"}, result)
}
