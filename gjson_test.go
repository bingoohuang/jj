package jj

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestRandomData is a fuzzing test that throws random data at the Parse
// function looking for panics.
func TestRandomData(t *testing.T) {
	var lstr string
	defer func() {
		if v := recover(); v != nil {
			println("'" + hex.EncodeToString([]byte(lstr)) + "'")
			println("'" + lstr + "'")
			panic(v)
		}
	}()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 200)
	for i := 0; i < 2000000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		lstr = string(b[:n])
		GetBytes([]byte(lstr), "zzzz")
		Parse(lstr)
	}
}

func TestRandomValidStrings(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 200)
	for i := 0; i < 100000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		sm, err := json.Marshal(string(b[:n]))
		if err != nil {
			t.Fatal(err)
		}
		var su string
		if err := json.Unmarshal(sm, &su); err != nil {
			t.Fatal(err)
		}
		token := Get(`{"str":`+string(sm)+`}`, "str")
		if token.Type != String || token.Str != su {
			println("["+token.Raw+"]", "["+token.Str+"]", "["+su+"]",
				"["+string(sm)+"]")
			t.Fatal("string mismatch")
		}
	}
}

func TestEmoji(t *testing.T) {
	const input = `{"utf8":"Example emoji, KO: \ud83d\udd13, \ud83c\udfc3 OK: \u2764\ufe0f "}`
	value := Get(input, "utf8")
	var s string
	_ = json.Unmarshal([]byte(value.Raw), &s)
	if value.String() != s {
		t.Fatalf("expected '%v', got '%v'", s, value.String())
	}
}

func testEscapePath(t *testing.T, jso, path, expect string) {
	if Get(jso, path).String() != expect {
		t.Fatalf("expected '%v', got '%v'", expect, Get(jso, path).String())
	}
}

func TestEscapePath(t *testing.T) {
	jso := `{
		"test":{
			"*":"valZ",
			"*v":"val0",
			"keyv*":"val1",
			"key*v":"val2",
			"keyv?":"val3",
			"key?v":"val4",
			"keyv.":"val5",
			"key.v":"val6",
			"keyk*":{"key?":"val7"}
		}
	}`

	testEscapePath(t, jso, "test.\\*", "valZ")
	testEscapePath(t, jso, "test.\\*v", "val0")
	testEscapePath(t, jso, "test.keyv\\*", "val1")
	testEscapePath(t, jso, "test.key\\*v", "val2")
	testEscapePath(t, jso, "test.keyv\\?", "val3")
	testEscapePath(t, jso, "test.key\\?v", "val4")
	testEscapePath(t, jso, "test.keyv\\.", "val5")
	testEscapePath(t, jso, "test.key\\.v", "val6")
	testEscapePath(t, jso, "test.keyk\\*.key\\?", "val7")
}

// this jso block is poorly formed on purpose.
var basicJSON = `{"age":100, "name":{"here":"B\\\"R"},
	"noop":{"what is a wren?":"a bird"},
	"happy":true,"immortal":false,
	"items":[1,2,3,{"tags":[1,2,3],"points":[[1,2],[3,4]]},4,5,6,7],
	"arr":["1",2,"3",{"hello":"world"},"4",5],
	"vals":[1,2,3,{"sadf":sdf"asdf"}],"name":{"first":"tom","last":null},
	"created":"2014-05-16T08:28:06.989Z",
	"loggy":{
		"programmers": [
    	    {
    	        "firstName": "Brett",
    	        "lastName": "McLaughlin",
    	        "email": "aaaa",
				"tag": "good"
    	    },
    	    {
    	        "firstName": "Jason",
    	        "lastName": "Hunter",
    	        "email": "bbbb",
				"tag": "bad"
    	    },
    	    {
    	        "firstName": "Elliotte",
    	        "lastName": "Harold",
    	        "email": "cccc",
				"tag":, "good"
    	    },
			{
				"firstName": 1002.3,
				"age": 101
			}
    	]
	},
	"lastly":{"yay":"final"}
}`

func TestTimeResult(t *testing.T) {
	assert(Get(basicJSON, "created").String() ==
		Get(basicJSON, "created").Time().Format(time.RFC3339Nano))
}

func TestParseAny(t *testing.T) {
	assert(Parse("100").Float() == 100)
	assert(Parse("true").Bool())
	assert(Parse("false").Bool() == false)
	assert(Parse("yikes").Exists() == false)
}

func TestManyVariousPathCounts(t *testing.T) {
	jso := `{"a":"a","b":"b","c":"c"}`
	counts := []int{
		3, 4, 7, 8, 9, 15, 16, 17, 31, 32, 33, 63, 64, 65, 127,
		128, 129, 255, 256, 257, 511, 512, 513,
	}
	paths := []string{"a", "b", "c"}
	expects := []string{"a", "b", "c"}
	for _, count := range counts {
		var gpaths []string
		var gexpects []string
		for i := 0; i < count; i++ {
			if i < len(paths) {
				gpaths = append(gpaths, paths[i])
				gexpects = append(gexpects, expects[i])
			} else {
				gpaths = append(gpaths, fmt.Sprintf("not%d", i))
				gexpects = append(gexpects, "null")
			}
		}
		results := GetMany(jso, gpaths...)
		for i := 0; i < len(paths); i++ {
			if results[i].String() != expects[i] {
				t.Fatalf("expected '%v', got '%v'", expects[i],
					results[i].String())
			}
		}
	}
}

func TestManyRecursion(t *testing.T) {
	var jso string
	var path string
	for i := 0; i < 100; i++ {
		jso += `{"a":`
		path += ".a"
	}
	jso += `"b"`
	for i := 0; i < 100; i++ {
		jso += `}`
	}
	path = path[1:]
	assert(GetMany(jso, path)[0].String() == "b")
}

func TestByteSafety(t *testing.T) {
	jsonb := []byte(`{"name":"Janet","age":38}`)
	mtok := GetBytes(jsonb, "name")
	if mtok.String() != "Janet" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.String())
	}
	mtok2 := GetBytes(jsonb, "age")
	if mtok2.Raw != "38" {
		t.Fatalf("expected %v, got %v", "Jason", mtok2.Raw)
	}
	jsonb[9] = 'T'
	jsonb[12] = 'd'
	jsonb[13] = 'y'
	if mtok.String() != "Janet" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.String())
	}
}

func get(jso, path string) Result {
	return GetBytes([]byte(jso), path)
}

func TestBasic(t *testing.T) {
	var mtok Result
	mtok = get(basicJSON, `loggy.programmers.#[tag="good"].firstName`)
	if mtok.String() != "Brett" {
		t.Fatalf("expected %v, got %v", "Brett", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[tag="good"]#.firstName`)
	if mtok.String() != `["Brett","Elliotte"]` {
		t.Fatalf("expected %v, got %v", `["Brett","Elliotte"]`, mtok.String())
	}
}

func TestIsArrayIsObject(t *testing.T) {
	mtok := get(basicJSON, "loggy")
	assert(mtok.IsObject())
	assert(!mtok.IsArray())

	mtok = get(basicJSON, "loggy.programmers")
	assert(!mtok.IsObject())
	assert(mtok.IsArray())

	mtok = get(basicJSON, `loggy.programmers.#[tag="good"]#.firstName`)
	assert(mtok.IsArray())

	mtok = get(basicJSON, `loggy.programmers.0.firstName`)
	assert(!mtok.IsObject())
	assert(!mtok.IsArray())
}

func TestPlus53BitInts(t *testing.T) {
	jso := `{"IdentityData":{"GameInstanceId":634866135153775564}}`
	value := Get(jso, "IdentityData.GameInstanceId")
	assert(value.Uint() == 634866135153775564)
	assert(value.Int() == 634866135153775564)
	assert(value.Float() == 634866135153775616)

	jso = `{"IdentityData":{"GameInstanceId":634866135153775564.88172}}`
	value = Get(jso, "IdentityData.GameInstanceId")
	assert(value.Uint() == 634866135153775616)
	assert(value.Int() == 634866135153775616)
	assert(value.Float() == 634866135153775616.88172)

	jso = `{
		"min_uint64": 0,
		"max_uint64": 18446744073709551615,
		"overflow_uint64": 18446744073709551616,
		"min_int64": -9223372036854775808,
		"max_int64": 9223372036854775807,
		"overflow_int64": 9223372036854775808,
		"min_uint53":  0,
		"max_uint53":  4503599627370495,
		"overflow_uint53": 4503599627370496,
		"min_int53": -2251799813685248,
		"max_int53": 2251799813685247,
		"overflow_int53": 2251799813685248
	}`

	assert(Get(jso, "min_uint53").Uint() == 0)
	assert(Get(jso, "max_uint53").Uint() == 4503599627370495)
	assert(Get(jso, "overflow_uint53").Int() == 4503599627370496)
	assert(Get(jso, "min_int53").Int() == -2251799813685248)
	assert(Get(jso, "max_int53").Int() == 2251799813685247)
	assert(Get(jso, "overflow_int53").Int() == 2251799813685248)
	assert(Get(jso, "min_uint64").Uint() == 0)
	assert(Get(jso, "max_uint64").Uint() == 18446744073709551615)
	// this next value overflows the max uint64 by one which will just
	// flip the number to zero
	assert(Get(jso, "overflow_uint64").Int() == 0)
	assert(Get(jso, "min_int64").Int() == -9223372036854775808)
	assert(Get(jso, "max_int64").Int() == 9223372036854775807)
	// this next value overflows the max int64 by one which will just
	// flip the number to the negative sign.
	assert(Get(jso, "overflow_int64").Int() == -9223372036854775808)
}

func TestIssue38(t *testing.T) {
	// These should not fail, even though the unicode is invalid.
	Get(`["S3O PEDRO DO BUTI\udf93"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93asdf"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u1"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u13"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u134"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u1345"]`, "0")
	Get(`["S3O PEDRO DO BUTI\udf93\u1345asd"]`, "0")
}

func TestTypes(t *testing.T) {
	assert((Result{Type: String}).Type.String() == "String")
	assert((Result{Type: Number}).Type.String() == "Number")
	assert((Result{Type: Null}).Type.String() == "Null")
	assert((Result{Type: False}).Type.String() == "False")
	assert((Result{Type: True}).Type.String() == "True")
	assert((Result{Type: JSON}).Type.String() == "JSON")
	assert((Result{Type: 100}).Type.String() == "")
	// bool
	assert((Result{Type: True}).Bool() == true)
	assert((Result{Type: False}).Bool() == false)
	assert((Result{Type: Number, Num: 1}).Bool() == true)
	assert((Result{Type: Number, Num: 0}).Bool() == false)
	assert((Result{Type: String, Str: "1"}).Bool() == true)
	assert((Result{Type: String, Str: "T"}).Bool() == true)
	assert((Result{Type: String, Str: "t"}).Bool() == true)
	assert((Result{Type: String, Str: "true"}).Bool() == true)
	assert((Result{Type: String, Str: "True"}).Bool() == true)
	assert((Result{Type: String, Str: "TRUE"}).Bool() == true)
	assert((Result{Type: String, Str: "tRuE"}).Bool() == true)
	assert((Result{Type: String, Str: "0"}).Bool() == false)
	assert((Result{Type: String, Str: "f"}).Bool() == false)
	assert((Result{Type: String, Str: "F"}).Bool() == false)
	assert((Result{Type: String, Str: "false"}).Bool() == false)
	assert((Result{Type: String, Str: "False"}).Bool() == false)
	assert((Result{Type: String, Str: "FALSE"}).Bool() == false)
	assert((Result{Type: String, Str: "fAlSe"}).Bool() == false)
	assert((Result{Type: String, Str: "random"}).Bool() == false)

	// int
	assert((Result{Type: String, Str: "1"}).Int() == 1)
	assert((Result{Type: True}).Int() == 1)
	assert((Result{Type: False}).Int() == 0)
	assert((Result{Type: Number, Num: 1}).Int() == 1)
	// uint
	assert((Result{Type: String, Str: "1"}).Uint() == 1)
	assert((Result{Type: True}).Uint() == 1)
	assert((Result{Type: False}).Uint() == 0)
	assert((Result{Type: Number, Num: 1}).Uint() == 1)
	// float
	assert((Result{Type: String, Str: "1"}).Float() == 1)
	assert((Result{Type: True}).Float() == 1)
	assert((Result{Type: False}).Float() == 0)
	assert((Result{Type: Number, Num: 1}).Float() == 1)
}

func TestForEach(t *testing.T) {
	Result{}.ForEach(nil)
	Result{Type: String, Str: "Hello"}.ForEach(func(_, value Result) bool {
		assert(value.String() == "Hello")
		return false
	})
	Result{Type: JSON, Raw: "*invalid*"}.ForEach(nil)

	jso := ` {"name": {"first": "Janet","last": "Prichard"},
	"asd\nf":"\ud83d\udd13","age": 47}`
	var count int
	ParseBytes([]byte(jso)).ForEach(func(key, value Result) bool {
		count++
		return true
	})
	assert(count == 3)
	ParseBytes([]byte(`{"bad`)).ForEach(nil)
	ParseBytes([]byte(`{"ok":"bad`)).ForEach(nil)
}

func TestMap(t *testing.T) {
	assert(len(ParseBytes([]byte(`"asdf"`)).Map()) == 0)
	assert(ParseBytes([]byte(`{"asdf":"ghjk"`)).Map()["asdf"].String() ==
		"ghjk")
	assert(len(Result{Type: JSON, Raw: "**invalid**"}.Map()) == 0)
	assert(Result{Type: JSON, Raw: "**invalid**"}.Value() == nil)
	assert(Result{Type: JSON, Raw: "{"}.Map() != nil)
}

func TestBasic1(t *testing.T) {
	mtok := get(basicJSON, `loggy.programmers`)
	var count int
	mtok.ForEach(func(key, value Result) bool {
		if key.Exists() {
			t.Fatalf("expected %v, got %v", false, key.Exists())
		}
		count++
		if count == 3 {
			return false
		}
		if count == 1 {
			i := 0
			value.ForEach(func(key, value Result) bool {
				switch i {
				case 0:
					if key.String() != "firstName" ||
						value.String() != "Brett" {
						t.Fatalf("expected %v/%v got %v/%v", "firstName",
							"Brett", key.String(), value.String())
					}
				case 1:
					if key.String() != "lastName" ||
						value.String() != "McLaughlin" {
						t.Fatalf("expected %v/%v got %v/%v", "lastName",
							"McLaughlin", key.String(), value.String())
					}
				case 2:
					if key.String() != "email" || value.String() != "aaaa" {
						t.Fatalf("expected %v/%v got %v/%v", "email", "aaaa",
							key.String(), value.String())
					}
				}
				i++
				return true
			})
		}
		return true
	})
	if count != 3 {
		t.Fatalf("expected %v, got %v", 3, count)
	}
}

func TestBasic2(t *testing.T) {
	mtok := get(basicJSON, `loggy.programmers.#[age=101].firstName`)
	if mtok.String() != "1002.3" {
		t.Fatalf("expected %v, got %v", "1002.3", mtok.String())
	}
	mtok = get(basicJSON,
		`loggy.programmers.#[firstName != "Brett"].firstName`)
	if mtok.String() != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[firstName % "Bre*"].email`)
	if mtok.String() != "aaaa" {
		t.Fatalf("expected %v, got %v", "aaaa", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[firstName !% "Bre*"].email`)
	if mtok.String() != "bbbb" {
		t.Fatalf("expected %v, got %v", "bbbb", mtok.String())
	}
	mtok = get(basicJSON, `loggy.programmers.#[firstName == "Brett"].email`)
	if mtok.String() != "aaaa" {
		t.Fatalf("expected %v, got %v", "aaaa", mtok.String())
	}
	mtok = get(basicJSON, "loggy")
	if mtok.Type != JSON {
		t.Fatalf("expected %v, got %v", JSON, mtok.Type)
	}
	if len(mtok.Map()) != 1 {
		t.Fatalf("expected %v, got %v", 1, len(mtok.Map()))
	}
	programmers := mtok.Map()["programmers"]
	if programmers.Array()[1].Map()["firstName"].Str != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason",
			mtok.Map()["programmers"].Array()[1].Map()["firstName"].Str)
	}
}

func TestBasic3(t *testing.T) {
	var mtok Result
	if Parse(basicJSON).Get("loggy.programmers").Get("1").
		Get("firstName").Str != "Jason" {
		t.Fatalf("expected %v, got %v", "Jason", Parse(basicJSON).
			Get("loggy.programmers").Get("1").Get("firstName").Str)
	}
	var token Result
	if token = Parse("-102"); token.Num != -102 {
		t.Fatalf("expected %v, got %v", -102, token.Num)
	}
	if token = Parse("102"); token.Num != 102 {
		t.Fatalf("expected %v, got %v", 102, token.Num)
	}
	if token = Parse("102.2"); token.Num != 102.2 {
		t.Fatalf("expected %v, got %v", 102.2, token.Num)
	}
	if token = Parse(`"hello"`); token.Str != "hello" {
		t.Fatalf("expected %v, got %v", "hello", token.Str)
	}
	if token = Parse(`"\"he\nllo\""`); token.Str != "\"he\nllo\"" {
		t.Fatalf("expected %v, got %v", "\"he\nllo\"", token.Str)
	}
	mtok = get(basicJSON, "loggy.programmers.#.firstName")
	if len(mtok.Array()) != 4 {
		t.Fatalf("expected 4, got %v", len(mtok.Array()))
	}
	for i, ex := range []string{"Brett", "Jason", "Elliotte", "1002.3"} {
		if mtok.Array()[i].String() != ex {
			t.Fatalf("expected '%v', got '%v'", ex, mtok.Array()[i].String())
		}
	}
	mtok = get(basicJSON, "loggy.programmers.#.asd")
	if mtok.Type != JSON {
		t.Fatalf("expected %v, got %v", JSON, mtok.Type)
	}
	if len(mtok.Array()) != 0 {
		t.Fatalf("expected 0, got %v", len(mtok.Array()))
	}
}

func TestBasic4(t *testing.T) {
	if get(basicJSON, "items.3.tags.#").Num != 3 {
		t.Fatalf("expected 3, got %v", get(basicJSON, "items.3.tags.#").Num)
	}
	if get(basicJSON, "items.3.points.1.#").Num != 2 {
		t.Fatalf("expected 2, got %v",
			get(basicJSON, "items.3.points.1.#").Num)
	}
	if get(basicJSON, "items.#").Num != 8 {
		t.Fatalf("expected 6, got %v", get(basicJSON, "items.#").Num)
	}
	if get(basicJSON, "vals.#").Num != 4 {
		t.Fatalf("expected 4, got %v", get(basicJSON, "vals.#").Num)
	}
	if !get(basicJSON, "name.last").Exists() {
		t.Fatal("expected true, got false")
	}
	token := get(basicJSON, "name.here")
	if token.String() != "B\\\"R" {
		t.Fatal("expecting 'B\\\"R'", "got", token.String())
	}
	token = get(basicJSON, "arr.#")
	if token.String() != "6" {
		fmt.Printf("%#v\n", token)
		t.Fatal("expecting 6", "got", token.String())
	}
	token = get(basicJSON, "arr.3.hello")
	if token.String() != "world" {
		t.Fatal("expecting 'world'", "got", token.String())
	}
	_ = token.Value().(string)
	token = get(basicJSON, "name.first")
	if token.String() != "tom" {
		t.Fatal("expecting 'tom'", "got", token.String())
	}
	_ = token.Value().(string)
	token = get(basicJSON, "name.last")
	if token.String() != "" {
		t.Fatal("expecting ''", "got", token.String())
	}
	if token.Value() != nil {
		t.Fatal("should be nil")
	}
}

func TestBasic5(t *testing.T) {
	token := get(basicJSON, "age")
	if token.String() != "100" {
		t.Fatal("expecting '100'", "got", token.String())
	}
	_ = token.Value().(float64)
	token = get(basicJSON, "happy")
	if token.String() != "true" {
		t.Fatal("expecting 'true'", "got", token.String())
	}
	_ = token.Value().(bool)
	token = get(basicJSON, "immortal")
	if token.String() != "false" {
		t.Fatal("expecting 'false'", "got", token.String())
	}
	_ = token.Value().(bool)
	token = get(basicJSON, "noop")
	if token.String() != `{"what is a wren?":"a bird"}` {
		t.Fatal("expecting '"+`{"what is a wren?":"a bird"}`+"'", "got",
			token.String())
	}
	_ = token.Value().(map[string]interface{})

	if get(basicJSON, "").Value() != nil {
		t.Fatal("should be nil")
	}

	get(basicJSON, "vals.hello")

	type msi = map[string]interface{}
	type fi = []interface{}
	mm := Parse(basicJSON).Value().(msi)
	fn := mm["loggy"].(msi)["programmers"].(fi)[1].(msi)["firstName"].(string)
	if fn != "Jason" {
		t.Fatalf("expecting %v, got %v", "Jason", fn)
	}
}

func TestGetPathAsSingleKey(t *testing.T) {
	s := `{"a.b.c":"abc"}}`
	if Get(s, "a.b.c").Str != "" {
		t.Fatal("fail")
	}
	if Get(s, "a.b.c", PathAsSingleKey(true)).Str != "abc" {
		t.Fatal("fail")
	}
}

func TestUnicode(t *testing.T) {
	jso := `{"key":0,"的情况下解":{"key":1,"的情况":2}}`
	if Get(jso, "的情况下解.key").Num != 1 {
		t.Fatal("fail")
	}
	if Get(jso, "的情况下解.的情况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(jso, "的情况下解.的?况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(jso, "的情况下解.的?*").Num != 2 {
		t.Fatal("fail")
	}
	if Get(jso, "的情况下解.*?况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(jso, "的情?下解.*?况").Num != 2 {
		t.Fatal("fail")
	}
	if Get(jso, "的情下解.*?况").Num != 0 {
		t.Fatal("fail")
	}
}

func TestUnescape(t *testing.T) {
	unescape(string([]byte{'\\', '\\', 0}))
	unescape(string([]byte{'\\', '/', '\\', 'b', '\\', 'f'}))
}

func assert(cond bool) {
	if !cond {
		panic("assert failed")
	}
}

func TestLess(t *testing.T) {
	assert(!Result{Type: Null}.Less(Result{Type: Null}, true))
	assert(Result{Type: Null}.Less(Result{Type: False}, true))
	assert(Result{Type: Null}.Less(Result{Type: True}, true))
	assert(Result{Type: Null}.Less(Result{Type: JSON}, true))
	assert(Result{Type: Null}.Less(Result{Type: Number}, true))
	assert(Result{Type: Null}.Less(Result{Type: String}, true))
	assert(!Result{Type: False}.Less(Result{Type: Null}, true))
	assert(Result{Type: False}.Less(Result{Type: True}, true))
	assert(Result{Type: String, Str: "abc"}.Less(Result{
		Type: String,
		Str:  "bcd",
	}, true))
	assert(Result{Type: String, Str: "ABC"}.Less(Result{
		Type: String,
		Str:  "abc",
	}, true))
	assert(!Result{Type: String, Str: "ABC"}.Less(Result{
		Type: String,
		Str:  "abc",
	}, false))
	assert(Result{Type: Number, Num: 123}.Less(Result{
		Type: Number,
		Num:  456,
	}, true))
	assert(!Result{Type: Number, Num: 456}.Less(Result{
		Type: Number,
		Num:  123,
	}, true))
	assert(!Result{Type: Number, Num: 456}.Less(Result{
		Type: Number,
		Num:  456,
	}, true))
	assert(stringLessInsensitive("abcde", "BBCDE"))
	assert(stringLessInsensitive("abcde", "bBCDE"))
	assert(stringLessInsensitive("Abcde", "BBCDE"))
	assert(stringLessInsensitive("Abcde", "bBCDE"))
	assert(!stringLessInsensitive("bbcde", "aBCDE"))
	assert(!stringLessInsensitive("bbcde", "ABCDE"))
	assert(!stringLessInsensitive("Bbcde", "aBCDE"))
	assert(!stringLessInsensitive("Bbcde", "ABCDE"))
	assert(!stringLessInsensitive("abcde", "ABCDE"))
	assert(!stringLessInsensitive("Abcde", "ABCDE"))
	assert(!stringLessInsensitive("abcde", "ABCDE"))
	assert(!stringLessInsensitive("ABCDE", "ABCDE"))
	assert(!stringLessInsensitive("abcde", "abcde"))
	assert(!stringLessInsensitive("123abcde", "123Abcde"))
	assert(!stringLessInsensitive("123Abcde", "123Abcde"))
	assert(!stringLessInsensitive("123Abcde", "123abcde"))
	assert(!stringLessInsensitive("123abcde", "123abcde"))
	assert(!stringLessInsensitive("124abcde", "123abcde"))
	assert(!stringLessInsensitive("124Abcde", "123Abcde"))
	assert(!stringLessInsensitive("124Abcde", "123abcde"))
	assert(!stringLessInsensitive("124abcde", "123abcde"))
	assert(stringLessInsensitive("124abcde", "125abcde"))
	assert(stringLessInsensitive("124Abcde", "125Abcde"))
	assert(stringLessInsensitive("124Abcde", "125abcde"))
	assert(stringLessInsensitive("124abcde", "125abcde"))
}

func TestIssue6(t *testing.T) {
	data := `{
      "code": 0,
      "msg": "",
      "data": {
        "sz002024": {
          "qfqday": [
            [
              "2014-01-02",
              "8.93",
              "9.03",
              "9.17",
              "8.88",
              "621143.00"
            ],
            [
              "2014-01-03",
              "9.03",
              "9.30",
              "9.47",
              "8.98",
              "1624438.00"
            ]
          ]
        }
      }
    }`

	var num []string
	for _, v := range Get(data, "data.sz002024.qfqday.0").Array() {
		num = append(num, v.String())
	}
	if fmt.Sprintf("%v", num) != "[2014-01-02 8.93 9.03 9.17 8.88 621143.00]" {
		t.Fatalf("invalid result")
	}
}

var exampleJSON = `{
	"widget": {
		"debug": "on",
		"window": {
			"title": "Sample Konfabulator Widget",
			"name": "main_window",
			"width": 500,
			"height": 500
		},
		"image": {
			"src": "Images/Sun.png",
			"hOffset": 250,
			"vOffset": 250,
			"alignment": "center"
		},
		"text": {
			"data": "Click Here",
			"size": 36,
			"style": "bold",
			"vOffset": 100,
			"alignment": "center",
			"onMouseUp": "sun1.opacity = (sun1.opacity / 100) * 90;"
		}
	}
}`

func TestNewParse(t *testing.T) {
	// fmt.Printf("%v\n", parse2(exampleJSON, "widget").String())
}

func TestUnmarshalMap(t *testing.T) {
	m1 := Parse(exampleJSON).Value().(map[string]interface{})
	var m2 map[string]interface{}
	if err := json.Unmarshal([]byte(exampleJSON), &m2); err != nil {
		t.Fatal(err)
	}
	b1, err := json.Marshal(m1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(m2)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(b1, b2) != 0 {
		t.Fatal("b1 != b2")
	}
}

func TestSingleArrayValue(t *testing.T) {
	jso := `{"key": "value","key2":[1,2,3,4,"A"]}`
	result := Get(jso, "key")
	array := result.Array()
	if len(array) != 1 {
		t.Fatal("array is empty")
	}
	if array[0].String() != "value" {
		t.Fatalf("got %s, should be %s", array[0].String(), "value")
	}

	array = Get(jso, "key2.#").Array()
	if len(array) != 1 {
		t.Fatalf("got '%v', expected '%v'", len(array), 1)
	}

	array = Get(jso, "key3").Array()
	if len(array) != 0 {
		t.Fatalf("got '%v', expected '%v'", len(array), 0)
	}
}

var manyJSON = `  {
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{
	"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"a":{"hello":"world"
	}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}}
	"position":{"type":"Point","coordinates":[-115.24,33.09]},
	"loves":["world peace"],
	"name":{"last":"Anderson","first":"Nancy"},
	"age":31
	"":{"a":"emptya","b":"emptyb"},
	"name.last":"Yellow",
	"name.first":"Cat",
}`

func TestManyBasic(t *testing.T) {
	testWatchForFallback = true
	defer func() {
		testWatchForFallback = false
	}()
	testMany := func(shouldFallback bool, expect string, paths ...string) {
		results := GetManyBytes(
			[]byte(manyJSON),
			paths...,
		)
		if len(results) != len(paths) {
			t.Fatalf("expected %v, got %v", len(paths), len(results))
		}
		if fmt.Sprintf("%v", results) != expect {
			fmt.Printf("%v\n", paths)
			t.Fatalf("expected %v, got %v", expect, results)
		}
		//if testLastWasFallback != shouldFallback {
		//	t.Fatalf("expected %v, got %v", shouldFallback, testLastWasFallback)
		//}
	}
	testMany(false, "[Point]", "position.type")
	testMany(false, `[emptya ["world peace"] 31]`, ".a", "loves", "age")
	testMany(false, `[["world peace"]]`, "loves")
	testMany(false, `[{"last":"Anderson","first":"Nancy"} Nancy]`, "name",
		"name.first")
	testMany(true, `[]`, strings.Repeat("a.", 40)+"hello")
	res := Get(manyJSON, strings.Repeat("a.", 48)+"a")
	testMany(true, `[`+res.String()+`]`, strings.Repeat("a.", 48)+"a")
	// these should fallback
	testMany(true, `[Cat Nancy]`, "name\\.first", "name.first")
	testMany(true, `[world]`, strings.Repeat("a.", 70)+"hello")
}

func testMany(t *testing.T, jso string, paths, expected []string) {
	testManyAny(t, jso, paths, expected, true)
	testManyAny(t, jso, paths, expected, false)
}

func testManyAny(t *testing.T, jso string, paths, expected []string,
	bytes bool) {
	var result []Result
	for i := 0; i < 2; i++ {
		var which string
		if i == 0 {
			which = "Get"
			result = nil
			for j := 0; j < len(expected); j++ {
				if bytes {
					result = append(result, GetBytes([]byte(jso), paths[j]))
				} else {
					result = append(result, Get(jso, paths[j]))
				}
			}
		} else if i == 1 {
			which = "GetMany"
			if bytes {
				result = GetManyBytes([]byte(jso), paths...)
			} else {
				result = GetMany(jso, paths...)
			}
		}
		for j := 0; j < len(expected); j++ {
			if result[j].String() != expected[j] {
				t.Fatalf("Using key '%s' for '%s'\nexpected '%v', got '%v'",
					paths[j], which, expected[j], result[j].String())
			}
		}
	}
}

func TestIssue20(t *testing.T) {
	jso := `{ "name": "FirstName", "name1": "FirstName1", ` +
		`"address": "address1", "addressDetails": "address2", }`
	paths := []string{"name", "name1", "address", "addressDetails"}
	expected := []string{"FirstName", "FirstName1", "address1", "address2"}
	t.Run("SingleMany", func(t *testing.T) {
		testMany(t, jso, paths,
			expected)
	})
}

func TestIssue21(t *testing.T) {
	jso := `{ "Level1Field1":3, 
	           "Level1Field4":4, 
			   "Level1Field2":{ "Level2Field1":[ "value1", "value2" ], 
			   "Level2Field2":{ "Level3Field1":[ { "key1":"value1" } ] } } }`
	paths := []string{
		"Level1Field1", "Level1Field2.Level2Field1",
		"Level1Field2.Level2Field2.Level3Field1", "Level1Field4",
	}
	expected := []string{
		"3", `[ "value1", "value2" ]`,
		`[ { "key1":"value1" } ]`, "4",
	}
	t.Run("SingleMany", func(t *testing.T) {
		testMany(t, jso, paths,
			expected)
	})
}

func TestRandomMany(t *testing.T) {
	var lstr string
	defer func() {
		if v := recover(); v != nil {
			println("'" + hex.EncodeToString([]byte(lstr)) + "'")
			println("'" + lstr + "'")
			panic(v)
		}
	}()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 512)
	for i := 0; i < 50000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		lstr = string(b[:n])
		paths := make([]string, rand.Int()%64)
		for i := range paths {
			var b []byte
			n := rand.Int() % 5
			for j := 0; j < n; j++ {
				if j > 0 {
					b = append(b, '.')
				}
				nn := rand.Int() % 10
				for k := 0; k < nn; k++ {
					b = append(b, 'a'+byte(rand.Int()%26))
				}
			}
			paths[i] = string(b)
		}
		GetMany(lstr, paths...)
	}
}

type ComplicatedType struct {
	unsettable int
	Tagged     string `jso:"tagged"`
	NotTagged  bool
	Nested     struct {
		Yellow string `jso:"yellow"`
	}
	NestedTagged struct {
		Green string
		Map   map[string]interface{}
		Ints  struct {
			Int   int `jso:"int"`
			Int8  int8
			Int16 int16
			Int32 int32
			Int64 int64 `jso:"int64"`
		}
		Uints struct {
			Uint   uint
			Uint8  uint8
			Uint16 uint16
			Uint32 uint32
			Uint64 uint64
		}
		Floats struct {
			Float64 float64
			Float32 float32
		}
		Byte byte
		Bool bool
	} `jso:"nestedTagged"`
	LeftOut      string `jso:"-"`
	SelfPtr      *ComplicatedType
	SelfSlice    []ComplicatedType
	SelfSlicePtr []*ComplicatedType
	SelfPtrSlice *[]ComplicatedType
	Interface    interface{} `jso:"interface"`
	Array        [3]int
	Time         time.Time `jso:"time"`
	Binary       []byte
	NonBinary    []byte
}

var complicatedJSON = `
{
	"tagged": "OK",
	"Tagged": "KO",
	"NotTagged": true,
	"unsettable": 101,
	"Nested": {
		"Yellow": "Green",
		"yellow": "yellow"
	},
	"nestedTagged": {
		"Green": "Green",
		"Map": {
			"this": "that", 
			"and": "the other thing"
		},
		"Ints": {
			"Uint": 99,
			"Uint16": 16,
			"Uint32": 32,
			"Uint64": 65
		},
		"Uints": {
			"int": -99,
			"Int": -98,
			"Int16": -16,
			"Int32": -32,
			"int64": -64,
			"Int64": -65
		},
		"Uints": {
			"Float32": 32.32,
			"Float64": 64.64
		},
		"Byte": 254,
		"Bool": true
	},
	"LeftOut": "you shouldn't be here",
	"SelfPtr": {"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}},
	"SelfSlice": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"SelfSlicePtr": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"SelfPtrSlice": [{"tagged":"OK","nestedTagged":{"Ints":{"Uint32":32}}}],
	"interface": "Tile38 Rocks!",
	"Interface": "Please Download",
	"Array": [0,2,3,4,5],
	"time": "2017-05-07T13:24:43-07:00",
	"Binary": "R0lGODlhPQBEAPeo",
	"NonBinary": [9,3,100,115]
}
`

func testvalid(t *testing.T, jso string, expect bool) {
	t.Helper()
	_, ok := validpayload([]byte(jso), 0)
	if ok != expect {
		t.Fatal("mismatch")
	}
}

func TestValidBasic(t *testing.T) {
	testvalid(t, "0", true)
	testvalid(t, "00", false)
	testvalid(t, "-00", false)
	testvalid(t, "-.", false)
	testvalid(t, "0.0", true)
	testvalid(t, "10.0", true)
	testvalid(t, "10e1", true)
	testvalid(t, "10EE", false)
	testvalid(t, "10E-", false)
	testvalid(t, "10E+", false)
	testvalid(t, "10E123", true)
	testvalid(t, "10E-123", true)
	testvalid(t, "10E-0123", true)
	testvalid(t, "", false)
	testvalid(t, " ", false)
	testvalid(t, "{}", true)
	testvalid(t, "{", false)
	testvalid(t, "-", false)
	testvalid(t, "-1", true)
	testvalid(t, "-1.", false)
	testvalid(t, "-1.0", true)
	testvalid(t, " -1.0", true)
	testvalid(t, " -1.0 ", true)
	testvalid(t, "-1.0 ", true)
	testvalid(t, "-1.0 i", false)
	testvalid(t, "-1.0 i", false)
	testvalid(t, "true", true)
	testvalid(t, " true", true)
	testvalid(t, " true ", true)
	testvalid(t, " True ", false)
	testvalid(t, " tru", false)
	testvalid(t, "false", true)
	testvalid(t, " false", true)
	testvalid(t, " false ", true)
	testvalid(t, " False ", false)
	testvalid(t, " fals", false)
	testvalid(t, "null", true)
	testvalid(t, " null", true)
	testvalid(t, " null ", true)
	testvalid(t, " Null ", false)
	testvalid(t, " nul", false)
	testvalid(t, " []", true)
	testvalid(t, " [true]", true)
	testvalid(t, " [ true, null ]", true)
	testvalid(t, " [ true,]", false)
	testvalid(t, `{"hello":"world"}`, true)
	testvalid(t, `{ "hello": "world" }`, true)
	testvalid(t, `{ "hello": "world", }`, false)
	testvalid(t, `{"a":"b",}`, false)
	testvalid(t, `{"a":"b","a"}`, false)
	testvalid(t, `{"a":"b","a":}`, false)
	testvalid(t, `{"a":"b","a":1}`, true)
	testvalid(t, `{"a":"b",2"1":2}`, false)
	testvalid(t, `{"a":"b","a": 1, "c":{"hi":"there"} }`, true)
	testvalid(t, `{"a":"b","a": 1, "c":{"hi":"there", "easy":["going",`+
		`{"mixed":"bag"}]} }`, true)
	testvalid(t, `""`, true)
	testvalid(t, `"`, false)
	testvalid(t, `"\n"`, true)
	testvalid(t, `"\"`, false)
	testvalid(t, `"\\"`, true)
	testvalid(t, `"a\\b"`, true)
	testvalid(t, `"a\\b\\\"a"`, true)
	testvalid(t, `"a\\b\\\uFFAAa"`, true)
	testvalid(t, `"a\\b\\\uFFAZa"`, false)
	testvalid(t, `"a\\b\\\uFFA"`, false)
	testvalid(t, complicatedJSON, true)
	testvalid(t, exampleJSON, true)
}

var jsonchars = []string{
	"{", "[", ",", ":", "}", "]", "1", "0", "true",
	"false", "null", `""`, `"\""`, `"a"`,
}

func makeRandomJSONChars(b []byte) {
	var bb []byte
	for len(bb) < len(b) {
		bb = append(bb, jsonchars[rand.Int()%len(jsonchars)]...)
	}
	copy(b, bb[:len(b)])
}

func TestValidRandom(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 100000)
	start := time.Now()
	for time.Since(start) < time.Second*3 {
		n := rand.Int() % len(b)
		rand.Read(b[:n])
		validpayload(b[:n], 0)
	}

	start = time.Now()
	for time.Since(start) < time.Second*3 {
		n := rand.Int() % len(b)
		makeRandomJSONChars(b[:n])
		validpayload(b[:n], 0)
	}
}

func TestGetMany47(t *testing.T) {
	jso := `{"bar": {"id": 99, "mybar": "my mybar" }, "foo": ` +
		`{"myfoo": [605]}}`
	paths := []string{"foo.myfoo", "bar.id", "bar.mybar", "bar.mybarx"}
	expected := []string{"[605]", "99", "my mybar", ""}
	results := GetMany(jso, paths...)
	if len(expected) != len(results) {
		t.Fatalf("expected %v, got %v", len(expected), len(results))
	}
	for i, path := range paths {
		if results[i].String() != expected[i] {
			t.Fatalf("expected '%v', got '%v' for path '%v'", expected[i],
				results[i].String(), path)
		}
	}
}

func TestGetMany48(t *testing.T) {
	jso := `{"bar": {"id": 99, "xyz": "my xyz"}, "foo": {"myfoo": [605]}}`
	paths := []string{"foo.myfoo", "bar.id", "bar.xyz", "bar.abc"}
	expected := []string{"[605]", "99", "my xyz", ""}
	results := GetMany(jso, paths...)
	if len(expected) != len(results) {
		t.Fatalf("expected %v, got %v", len(expected), len(results))
	}
	for i, path := range paths {
		if results[i].String() != expected[i] {
			t.Fatalf("expected '%v', got '%v' for path '%v'", expected[i],
				results[i].String(), path)
		}
	}
}

func TestResultRawForLiteral(t *testing.T) {
	for _, lit := range []string{"null", "true", "false"} {
		result := Parse(lit)
		if result.Raw != lit {
			t.Fatalf("expected '%v', got '%v'", lit, result.Raw)
		}
	}
}

func TestNullArray(t *testing.T) {
	n := len(Get(`{"data":null}`, "data").Array())
	if n != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, n)
	}
	n = len(Get(`{}`, "data").Array())
	if n != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, n)
	}
	n = len(Get(`{"data":[]}`, "data").Array())
	if n != 0 {
		t.Fatalf("expected '%v', got '%v'", 0, n)
	}
	n = len(Get(`{"data":[null]}`, "data").Array())
	if n != 1 {
		t.Fatalf("expected '%v', got '%v'", 1, n)
	}
}

func TestRandomGetMany(t *testing.T) {
	start := time.Now()
	for time.Since(start) < time.Second*3 {
		testRandomGetMany(t)
	}
}

func testRandomGetMany(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	jso, keys := randomJSON()
	for _, key := range keys {
		r := Get(jso, key)
		if !r.Exists() {
			t.Fatal("should exist")
		}
	}
	rkeysi := rand.Perm(len(keys))
	rkeysn := 1 + rand.Int()%32
	if len(rkeysi) > rkeysn {
		rkeysi = rkeysi[:rkeysn]
	}
	var rkeys []string
	for i := 0; i < len(rkeysi); i++ {
		rkeys = append(rkeys, keys[rkeysi[i]])
	}
	mres1 := GetMany(jso, rkeys...)
	var mres2 []Result
	for _, rkey := range rkeys {
		mres2 = append(mres2, Get(jso, rkey))
	}
	if len(mres1) != len(mres2) {
		t.Fatalf("expected %d, got %d", len(mres2), len(mres1))
	}
	for i := 0; i < len(mres1); i++ {
		mres1[i].Index = 0
		mres2[i].Index = 0
		v1 := fmt.Sprintf("%#v", mres1[i])
		v2 := fmt.Sprintf("%#v", mres2[i])
		if v1 != v2 {
			t.Fatalf("\nexpected %s\n"+
				"     got %s", v2, v1)
		}
	}
}

func TestIssue54(t *testing.T) {
	var r []Result
	jso := `{"MarketName":null,"Nounce":6115}`
	r = GetMany(jso, "Nounce", "Buys", "Sells", "Fills")
	if strings.Replace(fmt.Sprintf("%v", r), " ", "", -1) != "[6115]" {
		t.Fatalf("expected '%v', got '%v'", "[6115]",
			strings.Replace(fmt.Sprintf("%v", r), " ", "", -1))
	}
	r = GetMany(jso, "Nounce", "Buys", "Sells")
	if strings.Replace(fmt.Sprintf("%v", r), " ", "", -1) != "[6115]" {
		t.Fatalf("expected '%v', got '%v'", "[6115]",
			strings.Replace(fmt.Sprintf("%v", r), " ", "", -1))
	}
	r = GetMany(jso, "Nounce")
	if strings.Replace(fmt.Sprintf("%v", r), " ", "", -1) != "[6115]" {
		t.Fatalf("expected '%v', got '%v'", "[6115]",
			strings.Replace(fmt.Sprintf("%v", r), " ", "", -1))
	}
}

func randomString() string {
	var key string
	N := 1 + rand.Int()%16
	for i := 0; i < N; i++ {
		r := rand.Int() % 62
		if r < 10 {
			key += string(byte('0' + r))
		} else if r-10 < 26 {
			key += string(byte('a' + r - 10))
		} else {
			key += string(byte('A' + r - 10 - 26))
		}
	}
	return `"` + key + `"`
}

func randomBool() string {
	switch rand.Int() % 2 {
	default:
		return "false"
	case 1:
		return "true"
	}
}

func randomNumber() string {
	return strconv.FormatInt(int64(rand.Int()%1000000), 10)
}

func randomObjectOrArray(keys []string, prefix string, array bool, depth int) (
	string, []string) {
	N := 5 + rand.Int()%5
	var jso string
	if array {
		jso = "["
	} else {
		jso = "{"
	}
	for i := 0; i < N; i++ {
		if i > 0 {
			jso += ","
		}
		var pkey string
		if array {
			pkey = prefix + "." + strconv.FormatInt(int64(i), 10)
		} else {
			key := randomString()
			pkey = prefix + "." + key[1:len(key)-1]
			jso += key + `:`
		}
		keys = append(keys, pkey[1:])
		var kind int
		if depth == 5 {
			kind = rand.Int() % 4
		} else {
			kind = rand.Int() % 6
		}
		switch kind {
		case 0:
			jso += randomString()
		case 1:
			jso += randomBool()
		case 2:
			jso += "null"
		case 3:
			jso += randomNumber()
		case 4:
			var njson string
			njson, keys = randomObjectOrArray(keys, pkey, true, depth+1)
			jso += njson
		case 5:
			var njson string
			njson, keys = randomObjectOrArray(keys, pkey, false, depth+1)
			jso += njson
		}

	}
	if array {
		jso += "]"
	} else {
		jso += "}"
	}
	return jso, keys
}

func randomJSON() (jso string, keys []string) {
	return randomObjectOrArray(nil, "", false, 0)
}

func TestIssue55(t *testing.T) {
	jso := `{"one": {"two": 2, "three": 3}, "four": 4, "five": 5}`
	results := GetMany(jso, "four", "five", "one.two", "one.six")
	expected := []string{"4", "5", "2", ""}
	for i, r := range results {
		if r.String() != expected[i] {
			t.Fatalf("expected %v, got %v", expected[i], r.String())
		}
	}
}

func TestIssue58(t *testing.T) {
	jso := `{"data":[{"uid": 1},{"uid": 2}]}`
	res := Get(jso, `data.#[uid!=1]`).Raw
	if res != `{"uid": 2}` {
		t.Fatalf("expected '%v', got '%v'", `{"uid": 1}`, res)
	}
}

func TestObjectGrouping(t *testing.T) {
	jso := `
[
	true,
	{"name":"tom"},
	false,
	{"name":"janet"},
	null
]
`
	res := Get(jso, "#.name")
	if res.String() != `["tom","janet"]` {
		t.Fatalf("expected '%v', got '%v'", `["tom","janet"]`, res.String())
	}
}

func TestJSONLines(t *testing.T) {
	jso := `
true
false
{"name":"tom"}
[1,2,3,4,5]
{"name":"janet"}
null
12930.1203
	`
	paths := []string{"..#", "..0", "..2.name", "..#.name", "..6", "..7"}
	ress := []string{"7", "true", "tom", `["tom","janet"]`, "12930.1203", ""}
	for i, path := range paths {
		res := Get(jso, path)
		if res.String() != ress[i] {
			t.Fatalf("expected '%v', got '%v'", ress[i], res.String())
		}
	}

	jso = `
{"name": "Gilbert", "wins": [["straight", "7♣"], ["one pair", "10♥"]]}
{"name": "Alexa", "wins": [["two pair", "4♠"], ["two pair", "9♠"]]}
{"name": "May", "wins": []}
{"name": "Deloise", "wins": [["three of a kind", "5♣"]]}
`

	var i int
	lines := strings.Split(strings.TrimSpace(jso), "\n")
	ForEachLine(jso, func(line Result) bool {
		if line.Raw != lines[i] {
			t.Fatalf("expected '%v', got '%v'", lines[i], line.Raw)
		}
		i++
		return true
	})
	if i != 4 {
		t.Fatalf("expected '%v', got '%v'", 4, i)
	}
}

func TestNumUint64String(t *testing.T) {
	var i int64 = 9007199254740993 // 2^53 + 1
	j := fmt.Sprintf(`{"data":  [  %d, "hello" ] }`, i)
	res := Get(j, "data.0")
	if res.String() != "9007199254740993" {
		t.Fatalf("expected '%v', got '%v'", "9007199254740993", res.String())
	}
}

func TestNumInt64String(t *testing.T) {
	var i int64 = -9007199254740993
	j := fmt.Sprintf(`{"data":[ "hello", %d ]}`, i)
	res := Get(j, "data.1")
	if res.String() != "-9007199254740993" {
		t.Fatalf("expected '%v', got '%v'", "-9007199254740993", res.String())
	}
}

func TestNumBigString(t *testing.T) {
	i := "900719925474099301239109123101" // very big
	j := fmt.Sprintf(`{"data":[ "hello", "%s" ]}`, i)
	res := Get(j, "data.1")
	if res.String() != "900719925474099301239109123101" {
		t.Fatalf("expected '%v', got '%v'", "900719925474099301239109123101",
			res.String())
	}
}

func TestNumFloatString(t *testing.T) {
	var i int64 = -9007199254740993
	j := fmt.Sprintf(`{"data":[ "hello", %d ]}`, i) // No quotes around value!!
	res := Get(j, "data.1")
	if res.String() != "-9007199254740993" {
		t.Fatalf("expected '%v', got '%v'", "-9007199254740993", res.String())
	}
}

func TestDuplicateKeys(t *testing.T) {
	// this is vaild jso according to the JSON spec
	jso := `{"name": "Alex","name": "Peter"}`
	if Parse(jso).Get("name").String() !=
		Parse(jso).Map()["name"].String() {
		t.Fatalf("expected '%v', got '%v'",
			Parse(jso).Get("name").String(),
			Parse(jso).Map()["name"].String(),
		)
	}
	if !Valid(jso) {
		t.Fatal("should be valid")
	}
}

func TestArrayValues(t *testing.T) {
	jso := `{"array": ["PERSON1","PERSON2",0],}`
	values := Get(jso, "array").Array()
	var output string
	for i, val := range values {
		if i > 0 {
			output += "\n"
		}
		output += fmt.Sprintf("%#v", val)
	}
	expect := strings.Join([]string{
		`jj.Result{Type:3, Raw:"\"PERSON1\"", Str:"PERSON1", Num:0, Index:0}`,
		`jj.Result{Type:3, Raw:"\"PERSON2\"", Str:"PERSON2", Num:0, Index:0}`,
		`jj.Result{Type:2, Raw:"0", Str:"", Num:0, Index:0}`,
	}, "\n")
	if output != expect {
		t.Fatalf("expected '%v', got '%v'", expect, output)
	}
}

func BenchmarkValid(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Valid(complicatedJSON)
	}
}

func BenchmarkValidBytes(b *testing.B) {
	complicatedJSON := []byte(complicatedJSON)
	for i := 0; i < b.N; i++ {
		ValidBytes(complicatedJSON)
	}
}

func BenchmarkGoStdlibValidBytes(b *testing.B) {
	complicatedJSON := []byte(complicatedJSON)
	for i := 0; i < b.N; i++ {
		json.Valid(complicatedJSON)
	}
}

func TestModifier(t *testing.T) {
	jso := `{"other":{"hello":"world"},"arr":[1,2,3,4,5,6]}`
	opts := DefaultOptions
	opts.SortKeys = true
	exp := string(Pretty([]byte(jso), opts))
	res := Get(jso, `@pretty:{"sortKeys":true}`).String()
	if res != exp {
		t.Fatalf("expected '%v', got '%v'", exp, res)
	}
	res = Get(res, "@pretty|@reverse|@ugly").String()
	if res != jso {
		t.Fatalf("expected '%v', got '%v'", jso, res)
	}
	if res := Get(res, "@this").String(); res != jso {
		t.Fatalf("expected '%v', got '%v'", jso, res)
	}
	if res := Get(res, "other.@this").String(); res != `{"hello":"world"}` {
		t.Fatalf("expected '%v', got '%v'", jso, res)
	}
	res = Get(res, "@pretty|@reverse|arr|@reverse|2").String()
	if res != "4" {
		t.Fatalf("expected '%v', got '%v'", "4", res)
	}
	AddModifier("case", func(jso, arg string) string {
		if arg == "upper" {
			return strings.ToUpper(jso)
		}
		if arg == "lower" {
			return strings.ToLower(jso)
		}
		return jso
	})
	res = Get(jso, "other|@case:upper").String()
	if res != `{"HELLO":"WORLD"}` {
		t.Fatalf("expected '%v', got '%v'", `{"HELLO":"WORLD"}`, res)
	}
}

func TestChaining(t *testing.T) {
	jso := `{
		"info": {
			"friends": [
				{"first": "Dale", "last": "Murphy", "age": 44},
				{"first": "Roger", "last": "Craig", "age": 68},
				{"first": "Jane", "last": "Murphy", "age": 47}
			]
		}
	  }`
	res := Get(jso, "info.friends|0|first").String()
	if res != "Dale" {
		t.Fatalf("expected '%v', got '%v'", "Dale", res)
	}
	res = Get(jso, "info.friends|@reverse|0|age").String()
	if res != "47" {
		t.Fatalf("expected '%v', got '%v'", "47", res)
	}
	res = Get(jso, "@ugly|i\\nfo|friends.0.first").String()
	if res != "Dale" {
		t.Fatalf("expected '%v', got '%v'", "Dale", res)
	}
}

func TestSplitPipe(t *testing.T) {
	split := func(t *testing.T, path, el, er string, eo bool) {
		t.Helper()
		left, right, ok := splitPossiblePipe(path)
		// fmt.Printf("%-40s [%v] [%v] [%v]\n", path, left, right, ok)
		if left != el || right != er || ok != eo {
			t.Fatalf("expected '%v/%v/%v', got '%v/%v/%v",
				el, er, eo, left, right, ok)
		}
	}

	split(t, "hello", "", "", false)
	split(t, "hello.world", "", "", false)
	split(t, "hello|world", "hello", "world", true)
	split(t, "hello\\|world", "", "", false)
	split(t, "hello.#", "", "", false)
	split(t, `hello.#[a|1="asdf\"|1324"]#\|that`, "", "", false)
	split(t, `hello.#[a|1="asdf\"|1324"]#|that.more|yikes`,
		`hello.#[a|1="asdf\"|1324"]#`, "that.more|yikes", true)
	split(t, `a.#[]#\|b`, "", "", false)
}

func TestArrayEx(t *testing.T) {
	jso := `
	[
		{
			"c":[
				{"a":10.11}
			]
		}, {
			"c":[
				{"a":11.11}
			]
		}
	]`
	res := Get(jso, "@ugly|#.c.#[a=10.11]").String()
	if res != `[{"a":10.11}]` {
		t.Fatalf("expected '%v', got '%v'", `[{"a":10.11}]`, res)
	}
	res = Get(jso, "@ugly|#.c.#").String()
	if res != `[1,1]` {
		t.Fatalf("expected '%v', got '%v'", `[1,1]`, res)
	}
	res = Get(jso, "@reverse|0|c|0|a").String()
	if res != "11.11" {
		t.Fatalf("expected '%v', got '%v'", "11.11", res)
	}
	res = Get(jso, "#.c|#").String()
	if res != "2" {
		t.Fatalf("expected '%v', got '%v'", "2", res)
	}
}

func TestPipeDotMixing(t *testing.T) {
	jso := `{
		"info": {
			"friends": [
				{"first": "Dale", "last": "Murphy", "age": 44},
				{"first": "Roger", "last": "Craig", "age": 68},
				{"first": "Jane", "last": "Murphy", "age": 47}
			]
		}
	  }`
	var res string
	res = Get(jso, `info.friends.#[first="Dale"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `info|friends.#[first="Dale"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `info|friends.#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `info|friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `@ugly|info|friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `@ugly|info.@ugly|friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `@ugly.info|@ugly.friends|#[first="Dale"]|last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
}

func TestDeepSelectors(t *testing.T) {
	jso := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy",
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", 
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`
	var res string
	res = Get(jso, `info.friends.#[first="Dale"].extra.0`).String()
	if res != "10" {
		t.Fatalf("expected '%v', got '%v'", "10", res)
	}
	res = Get(jso, `info.friends.#[first="Dale"].extra|0`).String()
	if res != "10" {
		t.Fatalf("expected '%v', got '%v'", "10", res)
	}
	res = Get(jso, `info.friends.#[first="Dale"]|extra|0`).String()
	if res != "10" {
		t.Fatalf("expected '%v', got '%v'", "10", res)
	}
	res = Get(jso, `info.friends.#[details.city="Tempe"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
	res = Get(jso, `info.friends.#[details.city="Phoenix"].last`).String()
	if res != "Craig" {
		t.Fatalf("expected '%v', got '%v'", "Craig", res)
	}
	res = Get(jso, `info.friends.#[details.state="Arizona"].last`).String()
	if res != "Murphy" {
		t.Fatalf("expected '%v', got '%v'", "Murphy", res)
	}
}

func TestMultiArrayEx(t *testing.T) {
	jso := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy", "kind": "Person",
					"cust1": true,
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", "kind": "Person",
					"cust2": false,
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`

	var res string

	res = Get(jso, `info.friends.#[kind="Person"]#.kind|0`).String()
	if res != "Person" {
		t.Fatalf("expected '%v', got '%v'", "Person", res)
	}
	res = Get(jso, `info.friends.#.kind|0`).String()
	if res != "Person" {
		t.Fatalf("expected '%v', got '%v'", "Person", res)
	}

	res = Get(jso, `info.friends.#[kind="Person"]#.kind`).String()
	if res != `["Person","Person"]` {
		t.Fatalf("expected '%v', got '%v'", `["Person","Person"]`, res)
	}
	res = Get(jso, `info.friends.#.kind`).String()
	if res != `["Person","Person"]` {
		t.Fatalf("expected '%v', got '%v'", `["Person","Person"]`, res)
	}

	res = Get(jso, `info.friends.#[kind="Person"]#|kind`).String()
	if res != `` {
		t.Fatalf("expected '%v', got '%v'", ``, res)
	}
	res = Get(jso, `info.friends.#|kind`).String()
	if res != `` {
		t.Fatalf("expected '%v', got '%v'", ``, res)
	}

	res = Get(jso, `i*.f*.#[kind="Other"]#`).String()
	if res != `[]` {
		t.Fatalf("expected '%v', got '%v'", `[]`, res)
	}
}

func TestQueries(t *testing.T) {
	jso := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy", "kind": "Person",
					"cust1": true,
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", "kind": "Person",
					"cust2": false,
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`

	// numbers
	assert(Get(jso, "i*.f*.#[extra.0<11].first").Exists())
	assert(Get(jso, "i*.f*.#[extra.0<=11].first").Exists())
	assert(!Get(jso, "i*.f*.#[extra.0<10].first").Exists())
	assert(Get(jso, "i*.f*.#[extra.0<=10].first").Exists())
	assert(Get(jso, "i*.f*.#[extra.0=10].first").Exists())
	assert(!Get(jso, "i*.f*.#[extra.0=11].first").Exists())
	assert(Get(jso, "i*.f*.#[extra.0!=10].first").String() == "Roger")
	assert(Get(jso, "i*.f*.#[extra.0>10].first").String() == "Roger")
	assert(Get(jso, "i*.f*.#[extra.0>=10].first").String() == "Dale")

	// strings
	assert(Get(jso, `i*.f*.#[extra.0<"11"].first`).Exists())
	assert(Get(jso, `i*.f*.#[first>"Dale"].last`).String() == "Craig")
	assert(Get(jso, `i*.f*.#[first>="Dale"].last`).String() == "Murphy")
	assert(Get(jso, `i*.f*.#[first="Dale"].last`).String() == "Murphy")
	assert(Get(jso, `i*.f*.#[first!="Dale"].last`).String() == "Craig")
	assert(!Get(jso, `i*.f*.#[first<"Dale"].last`).Exists())
	assert(Get(jso, `i*.f*.#[first<="Dale"].last`).Exists())
	assert(Get(jso, `i*.f*.#[first%"Da*"].last`).Exists())
	assert(Get(jso, `i*.f*.#[first%"Dale"].last`).Exists())
	assert(Get(jso, `i*.f*.#[first%"*a*"]#|#`).String() == "1")
	assert(Get(jso, `i*.f*.#[first%"*e*"]#|#`).String() == "2")
	assert(Get(jso, `i*.f*.#[first!%"*e*"]#|#`).String() == "0")

	// trues
	assert(Get(jso, `i*.f*.#[cust1=true].first`).String() == "Dale")
	assert(Get(jso, `i*.f*.#[cust2=false].first`).String() == "Roger")
	assert(Get(jso, `i*.f*.#[cust1!=false].first`).String() == "Dale")
	assert(Get(jso, `i*.f*.#[cust2!=true].first`).String() == "Roger")
	assert(!Get(jso, `i*.f*.#[cust1>true].first`).Exists())
	assert(Get(jso, `i*.f*.#[cust1>=true].first`).Exists())
	assert(!Get(jso, `i*.f*.#[cust2<false].first`).Exists())
	assert(Get(jso, `i*.f*.#[cust2<=false].first`).Exists())
}

func TestQueryArrayValues(t *testing.T) {
	jso := `{
		"artists": [
			["Bob Dylan"],
			"John Lennon",
			"Mick Jagger",
			"Elton John",
			"Michael Jackson",
			"John Smith",
			true,
			123,
			456,
			false,
			null
		]
	}`
	assert(Get(jso, `a*.#[0="Bob Dylan"]#|#`).String() == "1")
	assert(Get(jso, `a*.#[0="Bob Dylan 2"]#|#`).String() == "0")
	assert(Get(jso, `a*.#[%"John*"]#|#`).String() == "2")
	assert(Get(jso, `a*.#[_%"John*"]#|#`).String() == "0")
	assert(Get(jso, `a*.#[="123"]#|#`).String() == "1")
}

func TestParenQueries(t *testing.T) {
	jso := `{
		"friends": [{"a":10},{"a":20},{"a":30},{"a":40}]
	}`
	assert(Get(jso, "friends.#(a>9)#|#").Int() == 4)
	assert(Get(jso, "friends.#(a>10)#|#").Int() == 3)
	assert(Get(jso, "friends.#(a>40)#|#").Int() == 0)
}

func TestSubSelectors(t *testing.T) {
	jso := `{
		"info": {
			"friends": [
				{
					"first": "Dale", "last": "Murphy", "kind": "Person",
					"cust1": true,
					"extra": [10,20,30],
					"details": {
						"city": "Tempe",
						"state": "Arizona"
					}
				},
				{
					"first": "Roger", "last": "Craig", "kind": "Person",
					"cust2": false,
					"extra": [40,50,60],
					"details": {
						"city": "Phoenix",
						"state": "Arizona"
					}
				}
			]
		}
	  }`
	assert(Get(jso, "[]").String() == "[]")
	assert(Get(jso, "{}").String() == "{}")
	res := Get(jso, `{`+
		`abc:info.friends.0.first,`+
		`info.friends.1.last,`+
		`"a`+"\r"+`a":info.friends.0.kind,`+
		`"abc":info.friends.1.kind,`+
		`{123:info.friends.1.cust2},`+
		`[info.friends.#[details.city="Phoenix"]#|#]`+
		`}.@pretty.@ugly`).String()
	// println(res)
	// {"abc":"Dale","last":"Craig","\"a\ra\"":"Person","_":{"123":false},"_":[1]}
	assert(Get(res, "abc").String() == "Dale")
	assert(Get(res, "last").String() == "Craig")
	assert(Get(res, "\"a\ra\"").String() == "Person")
	assert(Get(res, "@reverse.abc").String() == "Person")
	assert(Get(res, "_.123").String() == "false")
	assert(Get(res, "@reverse._.0").String() == "1")
	assert(Get(jso, "info.friends.[0.first,1.extra.0]").String() ==
		`["Dale",40]`)
	assert(Get(jso, "info.friends.#.[first,extra.0]").String() ==
		`[["Dale",10],["Roger",40]]`)
}

func TestArrayCountRawOutput(t *testing.T) {
	assert(Get(`[1,2,3,4]`, "#").Raw == "4")
}

func TestParseQuery(t *testing.T) {
	var path, op, value, remain string
	var ok bool

	path, op, value, remain, _, ok =
		parseQuery(`#(service_roles.#(=="one").()==asdf).cap`)
	assert(ok &&
		path == `service_roles.#(=="one").()` &&
		op == "=" &&
		value == `asdf` &&
		remain == `.cap`)

	path, op, value, remain, _, ok = parseQuery(`#(first_name%"Murphy").last`)
	assert(ok &&
		path == `first_name` &&
		op == `%` &&
		value == `"Murphy"` &&
		remain == `.last`)

	path, op, value, remain, _, ok = parseQuery(`#( first_name !% "Murphy" ).last`)
	assert(ok &&
		path == `first_name` &&
		op == `!%` &&
		value == `"Murphy"` &&
		remain == `.last`)

	path, op, value, remain, _, ok = parseQuery(`#(service_roles.#(=="one"))`)
	assert(ok &&
		path == `service_roles.#(=="one")` &&
		op == `` &&
		value == `` &&
		remain == ``)

	path, op, value, remain, _, ok =
		parseQuery(`#(a\("\"(".#(=="o\"(ne")%"ab\")").remain`)
	assert(ok &&
		path == `a\("\"(".#(=="o\"(ne")` &&
		op == "%" &&
		value == `"ab\")"` &&
		remain == `.remain`)
}

func TestParentSubQuery(t *testing.T) {
	jso := `{
		"topology": {
		  "instances": [
			{
			  "service_version": "1.2.3",
			  "service_locale": {"lang": "en"},
			  "service_roles": ["one", "two"]
			},
			{
			  "service_version": "1.2.4",
			  "service_locale": {"lang": "th"},
			  "service_roles": ["three", "four"]
			},
			{
			  "service_version": "1.2.2",
			  "service_locale": {"lang": "en"},
			  "service_roles": ["one"]
			}
		  ]
		}
	  }`
	res := Get(jso, `topology.instances.#( service_roles.#(=="one"))#.service_version`)
	// should return two instances
	assert(res.String() == `["1.2.3","1.2.2"]`)
}

func TestSingleModifier(t *testing.T) {
	data := `{"@key": "value"}`
	assert(Get(data, "@key").String() == "value")
	assert(Get(data, "\\@key").String() == "value")
}

func TestModifiersInMultipaths(t *testing.T) {
	AddModifier("case", func(jso, arg string) string {
		if arg == "upper" {
			return strings.ToUpper(jso)
		}
		if arg == "lower" {
			return strings.ToLower(jso)
		}
		return jso
	})
	jso := `{"friends": [
		{"age": 44, "first": "Dale", "last": "Murphy"},
		{"age": 68, "first": "Roger", "last": "Craig"},
		{"age": 47, "first": "Jane", "last": "Murphy"}
	]}`

	res := Get(jso, `friends.#.{age,first|@case:upper}|@ugly`)
	exp := `[{"age":44,"@case:upper":"DALE"},{"age":68,"@case:upper":"ROGER"},{"age":47,"@case:upper":"JANE"}]`
	assert(res.Raw == exp)

	res = Get(jso, `{friends.#.{age,first:first|@case:upper}|0.first}`)
	exp = `{"first":"DALE"}`
	assert(res.Raw == exp)
}

func TestIssue141(t *testing.T) {
	jso := `{"data": [{"q": 11, "w": 12}, {"q": 21, "w": 22}, {"q": 31, "w": 32} ], "sql": "some stuff here"}`
	assert(Get(jso, "data.#").Int() == 3)
	assert(Get(jso, "data.#.{q}|@ugly").Raw == `[{"q":11},{"q":21},{"q":31}]`)
	assert(Get(jso, "data.#.q|@ugly").Raw == `[11,21,31]`)
}

func TestChainedModifierStringArgs(t *testing.T) {
	// issue #143
	AddModifier("push", func(jso, arg string) string {
		jso = strings.TrimSpace(jso)
		if len(jso) < 2 || !Parse(jso).IsArray() {
			return jso
		}
		jso = strings.TrimSpace(jso[1 : len(jso)-1])
		if len(jso) == 0 {
			return "[" + arg + "]"
		}
		return "[" + jso + "," + arg + "]"
	})
	res := Get("[]", `@push:"2"|@push:"3"|@push:{"a":"b","c":["e","f"]}|@push:true|@push:10.23`)
	assert(res.String() == `["2","3",{"a":"b","c":["e","f"]},true,10.23]`)
}

func TestFlatten(t *testing.T) {
	jso := `[1,[2],[3,4],[5,[6,[7]]],{"hi":"there"},8,[9]]`
	assert(Get(jso, "@flatten").String() == `[1,2,3,4,5,[6,[7]],{"hi":"there"},8,9]`)
	assert(Get(jso, `@flatten:{"deep":true}`).String() == `[1,2,3,4,5,6,7,{"hi":"there"},8,9]`)
	assert(Get(`{"9999":1234}`, "@flatten").String() == `{"9999":1234}`)
}

func TestJoin(t *testing.T) {
	assert(Get(`[{},{}]`, "@join").String() == `{}`)
	assert(Get(`[{"a":1},{"b":2}]`, "@join").String() == `{"a":1,"b":2}`)
	assert(Get(`[{"a":1,"b":1},{"b":2}]`, "@join").String() == `{"a":1,"b":2}`)
	assert(Get(`[{"a":1,"b":1},{"b":2},5,{"c":3}]`, "@join").String() == `{"a":1,"b":2,"c":3}`)
	assert(Get(`[{"a":1,"b":1},{"b":2},5,{"c":3}]`, `@join:{"preserve":true}`).String() == `{"a":1,"b":1,"b":2,"c":3}`)
	assert(Get(`[{"a":1,"b":1},{"b":2},5,{"c":3}]`, `@join:{"preserve":true}.b`).String() == `1`)
	assert(Get(`{"9999":1234}`, "@join").String() == `{"9999":1234}`)
}

func TestValid(t *testing.T) {
	assert(Get("[{}", "@valid").Exists() == false)
	assert(Get("[{}]", "@valid").Exists() == true)
}

// https://github.com/tidwall/gjson/issues/152
func TestJoin152(t *testing.T) {
	jso := `{
		"distance": 1374.0,
		"validFrom": "2005-11-14",
		"historical": {
		  "type": "Day",
		  "name": "last25Hours",
		  "summary": {
			"units": {
			  "temperature": "C",
			  "wind": "m/s",
			  "snow": "cm",
			  "precipitation": "mm"
			},
			"days": [
			  {
				"time": "2020-02-08",
				"hours": [
				  {
					"temperature": {
					  "min": -2.0,
					  "max": -1.6,
					  "value": -1.6
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 92.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-08T16:00:00+01:00"
				  },
				  {
					"temperature": {
					  "min": -1.7,
					  "max": -1.3,
					  "value": -1.3
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 92.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-08T17:00:00+01:00"
				  },
				  {
					"temperature": {
					  "min": -1.3,
					  "max": -0.9,
					  "value": -1.2
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 91.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-08T18:00:00+01:00"
				  }
				]
			  },
			  {
				"time": "2020-02-09",
				"hours": [
				  {
					"temperature": {
					  "min": -1.7,
					  "max": -0.9,
					  "value": -1.5
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 91.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-09T00:00:00+01:00"
				  },
				  {
					"temperature": {
					  "min": -1.5,
					  "max": 0.9,
					  "value": 0.2
					},
					"wind": {},
					"precipitation": {},
					"humidity": {
					  "value": 67.0
					},
					"snow": {
					  "depth": 49.0
					},
					"time": "2020-02-09T01:00:00+01:00"
				  }
				]
			  }
			]
		  }
		}
	  }`

	res := Get(jso, "historical.summary.days.#.hours|@flatten|#.humidity.value")
	assert(res.Raw == `[92.0,92.0,91.0,91.0,67.0]`)
}

func TestVariousFuzz(t *testing.T) {
	// Issue #192	assert(t, squash(`"000"hello`) == `"000"`)
	assert(squash(`"000"`) == `"000"`)
	assert(squash(`"000`) == `"000`)
	assert(squash(`"`) == `"`)

	assert(squash(`[000]hello`) == `[000]`)
	assert(squash(`[000]`) == `[000]`)
	assert(squash(`[000`) == `[000`)
	assert(squash(`[`) == `[`)
	assert(squash(`]`) == `]`)

	testJSON := `0.#[[{}]].@valid:"000`
	Get(testJSON, testJSON)

	// Issue #195
	testJSON = `\************************************` +
		`**********{**",**,,**,**,**,**,"",**,**,**,**,**,**,**,**,**,**]`
	Get(testJSON, testJSON)

	// Issue #196
	testJSON = `[#.@pretty.@join:{""[]""preserve"3,"][{]]]`
	Get(testJSON, testJSON)
}
