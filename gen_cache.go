package jj

import (
	"regexp"
	"strings"

	"github.com/bingoohuang/gg/pkg/vars"
)

func GenWithCache(s string) string {
	return vars.ToString(vars.ParseExpr(s).Eval(NewCachingSubstituter()))
}

func NewCachingSubstituter() Substitute {
	internal := NewSubstituter(DefaultSubstituteFns)
	return &cacheValuer{Map: make(map[string]any), internal: internal}
}

type cacheValuer struct {
	Map      map[string]any
	internal *Substituter
}

func (v *cacheValuer) Register(fn string, f any) {
	v.internal.Register(fn, f)
}

var cacheSuffix = regexp.MustCompile(`^(.+)_\d+`)

func (v *cacheValuer) Value(name, params, expr string) any {
	wrapper := ""
	if p := strings.LastIndex(name, ".."); p > 0 {
		wrapper = name[p:]
		name = name[:p]
	}
	pureName := name

	subs := cacheSuffix.FindStringSubmatch(name)
	hasCachingResultTip := len(subs) > 0
	if hasCachingResultTip { // CachingSubstituter tips found
		pureName = subs[1]
		x, ok := v.Map[name]
		if ok {
			return invokeJiami(x, wrapper)
		}
	}

	x := v.internal.Value(pureName+wrapper, params, expr)

	if hasCachingResultTip {
		v.Map[name] = x
	}
	return x
}
