package jj

import "testing"

func TestGenCached(t *testing.T) {
	cached := GenWithCache("@身份证_1 @身份证_1..jiami")
	t.Log(cached)
}
