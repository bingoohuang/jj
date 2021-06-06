package jj_test

import (
	"fmt"
	"github.com/bingoohuang/jj"
	"github.com/stretchr/testify/assert"
	"testing"
)

// https://github.com/nuysoft/Mock/wiki/Syntax-Specification
// https://www.json-generator.com/

func TestGenKeyHitRepeat(t *testing.T) {
	assert.Equal(t, `{"id":"11"}`, jj.NewGenContext().Gen(`{"id|2": "1" }`))
}

func TestGenKeyHitRepeatObjectId(t *testing.T) {
	gen := jj.NewGenContext()
	gen.RegisterFn("objectId", func(args string) interface{} { return 456 })
	assert.Equal(t, `{"id":"456456"}`, gen.Gen(`{"id|2": "@objectId" }`))
}

func TestGenRepeatObject(t *testing.T) {
	assert.Equal(t, `[{"id":123},{"id":123}]`, jj.NewGenContext().Gen(`["|2", { "id": 123 }]`))
}

func TestGenRepeatString(t *testing.T) {
	assert.Equal(t, `["123","123"]`, jj.NewGenContext().Gen(`["|2", "123"]`))
}

func TestGenRepeatInt(t *testing.T) {
	assert.Equal(t, `[123,123]`, jj.NewGenContext().Gen(`["|2", 123]`))
}

func TestGenRepeatInt2(t *testing.T) {
	assert.Equal(t, `[123,123,456]`, jj.NewGenContext().Gen(`["|2", 123, 456]`))
}

func TestGenRepeatObjectId(t *testing.T) {
	gen := jj.NewGenContext()
	gen.MockTimes = 2
	gen.RegisterFn("objectId", func(args string) interface{} { return 456 })
	assert.Equal(t, `[{"id":456},{"id":456}]`, gen.Gen(`["|2-7", { "id": "@objectId" }]`))
}

func TestGenRepeat2(t *testing.T) {
	gen := jj.NewGenContext()
	gen.MockTimes = 2
	gen.RegisterFn("objectId", func(args string) interface{} { return 456 })
	gen.RegisterFn("random", func(args string) interface{} { return 1010 })
	out, _ := gen.Process(`["|2-7", { "id": "@objectId",  "tags": ["|3", "@random(10)"] }]`)
	assert.Equal(t, `[{"id":456,"tags":[1010,1010]},{"id":456,"tags":[1010,1010]}]`, out.Out)
}

func TestGenObjectId(t *testing.T) {
	gen := jj.NewGenContext()
	gen.RegisterFn("objectId", func(args string) interface{} { return "123" })
	assert.Equal(t, `{"id":"123"}`, gen.Gen(` {"id": "@objectId"} `))
}

func subLit(n string) *jj.SubLiteral { return &jj.SubLiteral{Val: n} }
func subVar(n string) *jj.SubVar     { return &jj.SubVar{Name: n} }
func subVarP(n, p string) *jj.SubVar { return &jj.SubVar{Name: n, Params: p} }

func TestParseSubstitutes(t *testing.T) {
	assert.Equal(t, jj.Subs{subVar("fn")}, jj.ParseSubstitutes("@fn"))
	assert.Equal(t, jj.Subs{subVar("fn"), subLit("@")}, jj.ParseSubstitutes("@fn@"))
	assert.Equal(t, jj.Subs{subLit("abc"), subVar("fn")}, jj.ParseSubstitutes("abc@{fn}"))
	assert.Equal(t, jj.Subs{subVar("fn"), subVar("fn")}, jj.ParseSubstitutes("@fn@fn"))
	assert.Equal(t, jj.Subs{subLit("abc"), subVar("fn"), subVar("fn"), subLit("efg")}, jj.ParseSubstitutes("abc@fn@{fn}efg"))
	assert.Equal(t, jj.Subs{subLit("abc"), subVar("fn"), subVarP("fn", "1"), subLit("efg")}, jj.ParseSubstitutes("abc@fn@{fn(1)}efg"))
	assert.Equal(t, jj.Subs{subVarP("fn", "100")}, jj.ParseSubstitutes("@fn(100)"))
	assert.Equal(t, jj.Subs{subLit("@")}, jj.ParseSubstitutes("@"))
	assert.Equal(t, jj.Subs{subLit("@@")}, jj.ParseSubstitutes("@@"))
}

func TestGenExample(t *testing.T) {
	fmt.Println(jj.Gen(`{"id": "@objectId"}`))
	fmt.Println(jj.Gen(`["|1-3", { "id": "@objectId",  "tags": ["|1-2", "@random(5-10)"] }]`))
	fmt.Println(jj.Gen(`{"id": "@random(red,green,blue)"}`))
	fmt.Println(jj.Gen(`{"id": "@random(1,2,3)"}`))
	fmt.Println(jj.Gen(`{"id": "@regex([abc]{10})"}`))
	fmt.Println(jj.Gen(`{"id|2-5": "1" }`))

	fmt.Println(jj.Gen(`{"id": "@random_int"}`))
	fmt.Println(jj.Gen(`{"id": "@random_int(100-999)"}`))
	fmt.Println(jj.Gen(`{"id": "Hello@random_int(100-999)"}`))
	fmt.Println(jj.Gen(`{"ok": "@random_bool"}`))

	fmt.Println(jj.Gen(`{"day": "@random_time"}`))
	fmt.Println(jj.Gen(`{"day": "@random_time(yyyy-MM-dd)"}`))
	fmt.Println(jj.Gen(`{"day": "@random_time(yyyy-MM-ddTHH:mm:ss)"}`))
	fmt.Println(jj.Gen(`{"day": "@random_time(yyyy-MM-ddTHH:mm:ss)"}`))
	fmt.Println(jj.Gen(`{"day": "@random_time(yyyy-MM-dd,1990-01-01,2021-06-06)"}`))
	fmt.Println(jj.Gen(`{"day": "@random_time(sep=# yyyy-MM-dd#1990-01-01#2021-06-06)"}`))
	fmt.Println(jj.Gen(`{"day": "@random_time(sep=# yyyy-MM-dd#1990-01-01#2021-06-06)"}`))
}
