package jj_test

import (
	"fmt"
	"github.com/bingoohuang/jj"
	"testing"
)

func TestRandJSON(t *testing.T) {
	js := jj.Rand()
	fmt.Println(string(js))
}
