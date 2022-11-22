package jj

import (
	"regexp"
	"strings"
	"sync"

	"github.com/bingoohuang/gg/pkg/vars"
)

func GenWithCache(s string) string {
	return vars.ToString(vars.ParseExpr(s).Eval(CachingSubstituter))
}

var CachingSubstituter Substitute = &cacheValuer{Map: make(map[string]interface{})}

type cacheValuer struct {
	Map map[string]interface{}
	sync.RWMutex
}

func (v *cacheValuer) Register(fn string, f interface{}) {
	DefaultSubstituteFns.Register(fn, f)
}

var cacheSuffix = regexp.MustCompile(`^(.+)_\d+`)

func (v *cacheValuer) Value(name, params, expr string) interface{} {
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
		v.RLock()
		x, ok := v.Map[name]
		v.RUnlock()
		if ok {
			return invokeJiami(x, wrapper)
		}
	}

	x := DefaultGen.Value(pureName+wrapper, params, expr)

	if hasCachingResultTip {
		v.Lock()
		v.Map[name] = x
		v.Unlock()
	}
	return x
}
