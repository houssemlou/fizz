package fizzbuzz

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// fixed namespace so identical params always produce the same UUID across restarts
var fizzbuzzNamespace = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

type TopResult struct {
	Params Params `json:"request"`
	Hits   int    `json:"hits"`
}

type Params struct {
	Int1  int    `json:"int1"`
	Int2  int    `json:"int2"`
	Limit int    `json:"limit"`
	Str1  string `json:"str1"`
	Str2  string `json:"str2"`
}

func (p Params) Validate() error {
	if p.Int1 <= 0 {
		return errors.New("int1 must be a positive integer")
	}
	if p.Int2 <= 0 {
		return errors.New("int2 must be a positive integer")
	}
	if p.Limit <= 0 {
		return errors.New("limit must be a positive integer")
	}
	if p.Limit > 1_000 {
		return errors.New("limit must not exceed 1,000")
	}
	if p.Str1 == "" {
		return errors.New("str1 must not be empty")
	}
	if p.Str2 == "" {
		return errors.New("str2 must not be empty")
	}
	return nil
}

// IdempotentID uses json.Marshal so any new field is automatically included in the hash.
func (p Params) IdempotentID() string {
	data, _ := json.Marshal(p)
	return uuid.NewSHA1(fizzbuzzNamespace, data).String()
}
