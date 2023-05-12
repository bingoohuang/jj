package jj

import (
	jsongo "encoding/json"
	"sort"
	"strconv"
	"unsafe"
)

type errorType struct {
	msg string
}

func (err *errorType) Error() string {
	return err.msg
}

// SetOptions represents additional options for the Set and Delete functions.
type SetOptions struct {
	// Optimistic is a hint that the value likely exists which
	// allows for the sjson to perform a fast-track search and replace.
	Optimistic bool
	// ReplaceInPlace is a hint to replace the input json rather than
	// allocate a new json byte slice. When this field is specified
	// the input json will not longer be valid and it should not be used
	// In the case when the destination slice doesn't have enough free
	// bytes to replace the data in place, a new bytes slice will be
	// created under the hood.
	// The Optimistic flag must be set to true and the input must be a
	// byte slice in order to use this field.
	ReplaceInPlace bool

	PathOption
}

type pathResult struct {
	part  string // current key part
	gpart string // gjson get part
	path  string // remaining path
	force bool   // force a string key
	more  bool   // there is more path to parse
}

func isSimpleChar(ch byte) bool {
	switch ch {
	case '|', '#', '@', '*', '?':
		return false
	default:
		return true
	}
}

func parsePath(path string, sc setConfig) (res pathResult, simple bool) {
	var r pathResult
	if sc.RawPath {
		r.part = path
		r.gpart = path
		return r, true
	}

	if len(path) > 0 && path[0] == ':' {
		r.force = true
		path = path[1:]
	}
	for i := 0; i < len(path); i++ {
		if path[i] == '.' {
			r.part = path[:i]
			r.gpart = path[:i]
			r.path = path[i+1:]
			r.more = true
			return r, true
		}
		if !isSimpleChar(path[i]) {
			return r, false
		}
		if path[i] == '\\' {
			// go into escape mode. this is a slower path that
			// strips off the escape character from the part.
			epart := []byte(path[:i])
			gpart := []byte(path[:i+1])
			i++
			if i < len(path) {
				epart = append(epart, path[i])
				gpart = append(gpart, path[i])
				i++
				for ; i < len(path); i++ {
					if path[i] == '\\' {
						gpart = append(gpart, '\\')
						i++
						if i < len(path) {
							epart = append(epart, path[i])
							gpart = append(gpart, path[i])
						}
						continue
					} else if path[i] == '.' {
						r.part = string(epart)
						r.gpart = string(gpart)
						r.path = path[i+1:]
						r.more = true
						return r, true
					} else if !isSimpleChar(path[i]) {
						return r, false
					}
					epart = append(epart, path[i])
					gpart = append(gpart, path[i])
				}
			}
			// append the last part
			r.part = string(epart)
			r.gpart = string(gpart)
			return r, true
		}
	}
	r.part = path
	r.gpart = path
	return r, true
}

func mustMarshalString(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < ' ' || s[i] > 0x7f || s[i] == '"' || s[i] == '\\' {
			return true
		}
	}
	return false
}

// appendStringify makes a json string and appends to buf.
func appendStringify(buf []byte, s string) []byte {
	if mustMarshalString(s) {
		b, _ := jsongo.Marshal(s)
		return append(buf, b...)
	}
	buf = append(buf, '"')
	buf = append(buf, s...)
	buf = append(buf, '"')
	return buf
}

// appendBuild builds a json block from a json path.
func appendBuild(buf []byte, array bool, paths []pathResult, raw string,
	stringify bool,
) []byte {
	if !array {
		buf = appendStringify(buf, paths[0].part)
		buf = append(buf, ':')
	}
	if len(paths) > 1 {
		n, numeric := atoui(paths[1])
		if numeric || (!paths[1].force && paths[1].part == "-1") {
			buf = append(buf, '[')
			buf = appendRepeat(buf, "null,", n)
			buf = appendBuild(buf, true, paths[1:], raw, stringify)
			buf = append(buf, ']')
		} else {
			buf = append(buf, '{')
			buf = appendBuild(buf, false, paths[1:], raw, stringify)
			buf = append(buf, '}')
		}
	} else {
		if stringify {
			buf = appendStringify(buf, raw)
		} else {
			buf = append(buf, raw...)
		}
	}
	return buf
}

// atoui does a rip conversion of string -> unigned int.
func atoui(r pathResult) (n int, ok bool) {
	if r.force {
		return 0, false
	}
	for i := 0; i < len(r.part); i++ {
		if r.part[i] < '0' || r.part[i] > '9' {
			return 0, false
		}
		n = n*10 + int(r.part[i]-'0')
	}
	return n, true
}

// appendRepeat repeats string "n" times and appends to buf.
func appendRepeat(buf []byte, s string, n int) []byte {
	for i := 0; i < n; i++ {
		buf = append(buf, s...)
	}
	return buf
}

// deleteTailItem deletes the previous key or comma.
func deleteTailItem(buf []byte) ([]byte, bool) {
loop:
	for i := len(buf) - 1; i >= 0; i-- {
		// look for either a ',',':','['
		switch buf[i] {
		case '[':
			return buf, true
		case ',':
			return buf[:i], false
		case ':':
			// delete tail string
			i--
			for ; i >= 0; i-- {
				if buf[i] == '"' {
					i--
					for ; i >= 0; i-- {
						if buf[i] == '"' {
							i--
							if i >= 0 && buf[i] == '\\' {
								i--
								continue
							}
							for ; i >= 0; i-- {
								// look for either a ',','{'
								switch buf[i] {
								case '{':
									return buf[:i+1], true
								case ',':
									return buf[:i], false
								}
							}
						}
					}
					break
				}
			}
			break loop
		}
	}
	return buf, false
}

var errNoChange = &errorType{"no change"}

func appendRawPaths(buf []byte, jstr string, paths []pathResult, raw string, sc setConfig) ([]byte, error) {
	var err error
	var res Result
	var found bool
	if sc.del {
		if sc.RawPath {
			res = Get(jstr, paths[0].gpart, ApplyGetOption(sc.PathOption))
		} else if paths[0].part == "-1" && !paths[0].force {
			res = Get(jstr, "#")
			if res.Int() > 0 {
				res = Get(jstr, strconv.FormatInt(res.Int()-1, 10))
				found = true
			}
		}
	}
	if !found {
		res = Get(jstr, paths[0].gpart, ApplyGetOption(sc.PathOption), DisableNegativeIndex(true))
	}
	if res.Index > 0 {
		if len(paths) > 1 {
			buf = append(buf, jstr[:res.Index]...)
			buf, err = appendRawPaths(buf, res.Raw, paths[1:], raw, sc)
			if err != nil {
				return nil, err
			}
			buf = append(buf, jstr[res.Index+len(res.Raw):]...)
			return buf, nil
		}
		buf = append(buf, jstr[:res.Index]...)
		var exidx int // additional forward stripping
		if sc.del {
			var delNextComma bool
			buf, delNextComma = deleteTailItem(buf)
			if delNextComma {
				i, j := res.Index+len(res.Raw), 0
				for ; i < len(jstr); i, j = i+1, j+1 {
					if jstr[i] <= ' ' {
						continue
					}
					if jstr[i] == ',' {
						exidx = j + 1
					}
					break
				}
			}
		} else {
			if sc.stringify {
				buf = appendStringify(buf, raw)
			} else {
				buf = append(buf, raw...)
			}
		}
		buf = append(buf, jstr[res.Index+len(res.Raw)+exidx:]...)
		return buf, nil
	}
	if sc.del {
		return nil, errNoChange
	}
	n, numeric := atoui(paths[0])
	isempty := true
	for i := 0; i < len(jstr); i++ {
		if jstr[i] > ' ' {
			isempty = false
			break
		}
	}
	if isempty {
		if numeric {
			jstr = "[]"
		} else {
			jstr = "{}"
		}
	}
	jsres := Parse(jstr)
	if jsres.Type != JSON {
		if numeric {
			jstr = "[]"
		} else {
			jstr = "{}"
		}
		jsres = Parse(jstr)
	}
	var comma bool
	for i := 1; i < len(jsres.Raw); i++ {
		if jsres.Raw[i] <= ' ' {
			continue
		}
		if jsres.Raw[i] == '}' || jsres.Raw[i] == ']' {
			break
		}
		comma = true
		break
	}
	switch jsres.Raw[0] {
	default:
		return nil, &errorType{"json must be an object or array"}
	case '{':
		end := len(jsres.Raw) - 1
		for ; end > 0; end-- {
			if jsres.Raw[end] == '}' {
				break
			}
		}
		buf = append(buf, jsres.Raw[:end]...)
		if comma {
			buf = append(buf, ',')
		}
		buf = appendBuild(buf, false, paths, raw, sc.stringify)
		buf = append(buf, '}')
		return buf, nil
	case '[':
		var appendit bool
		if !numeric {
			if paths[0].part == "-1" && !paths[0].force {
				appendit = true
			} else {
				return nil, &errorType{
					"cannot set array element for non-numeric key '" +
						paths[0].part + "'",
				}
			}
		}
		if appendit {
			njson := trim(jsres.Raw)
			if njson[len(njson)-1] == ']' {
				njson = njson[:len(njson)-1]
			}
			buf = append(buf, njson...)
			if comma {
				buf = append(buf, ',')
			}

			buf = appendBuild(buf, true, paths, raw, sc.stringify)
			buf = append(buf, ']')
			return buf, nil
		}
		buf = append(buf, '[')
		ress := jsres.Array()
		for i := 0; i < len(ress); i++ {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, ress[i].Raw...)
		}
		if len(ress) == 0 {
			buf = appendRepeat(buf, "null,", n-len(ress))
		} else {
			buf = appendRepeat(buf, ",null", n-len(ress))
			if comma {
				buf = append(buf, ',')
			}
		}
		buf = appendBuild(buf, true, paths, raw, sc.stringify)
		buf = append(buf, ']')
		return buf, nil
	}
}

func isOptimisticPath(path string, sc setConfig) bool {
	if sc.RawPath {
		return true
	}

	for i := 0; i < len(path); i++ {
		if path[i] < '.' || path[i] > 'z' {
			return false
		}
		if path[i] > '9' && path[i] < 'A' {
			return false
		}
		if path[i] > 'z' {
			return false
		}
	}
	return true
}

// SetRaw sets a raw json value for the specified path.
// This function works the same as Set except that the value is set as a
// raw block of json. This allows for setting premarshalled json objects.
func SetRaw(json, path, value string, options ...SetOptions) (string, error) {
	var optimistic bool
	if len(options) > 0 {
		optimistic = options[0].Optimistic
	}
	res, err := set(json, path, value, makeSetConfig(false, false, optimistic, false))
	if err == errNoChange {
		return json, nil
	}
	return string(res), err
}

type dtype struct{}

// Delete deletes a value from json for the specified path.
func Delete(json, path string, options ...SetOptions) (string, error) {
	return Set(json, path, dtype{}, options...)
}

// DeleteBytes deletes a value from json for the specified path.
func DeleteBytes(json []byte, path string, options ...SetOptions) ([]byte, error) {
	return SetBytes(json, path, dtype{}, options...)
}

type setConfig struct {
	stringify, del, optimistic, inplace bool
	PathOption
}

func makeSetConfig(stringify, del, optimistic, inplace bool) setConfig {
	return setConfig{stringify: stringify, del: del, optimistic: optimistic, inplace: inplace}
}

func set(jstr, path, raw string, sc setConfig) ([]byte, error) {
	if path == "" {
		return nil, &errorType{"path cannot be empty"}
	}
	if !sc.del && sc.optimistic && isOptimisticPath(path, sc) {
		res := Get(jstr, path, ApplyGetOption(sc.PathOption), DisableNegativeIndex(true))
		if res.Exists() && res.Index > 0 {
			sz := len(jstr) - len(res.Raw) + len(raw)
			if sc.stringify {
				sz += 2
			}
			if sc.inplace && sz <= len(jstr) {
				if !sc.stringify || !mustMarshalString(raw) {
					jsonh := *(*stringHeader)(unsafe.Pointer(&jstr))
					jsonbh := sliceHeader{data: jsonh.data, len: jsonh.len, cap: jsonh.len}
					jbytes := *(*[]byte)(unsafe.Pointer(&jsonbh))
					if sc.stringify {
						jbytes[res.Index] = '"'
						copy(jbytes[res.Index+1:], raw)
						jbytes[res.Index+1+len(raw)] = '"'
						copy(jbytes[res.Index+1+len(raw)+1:],
							jbytes[res.Index+len(res.Raw):])
					} else {
						copy(jbytes[res.Index:], raw)
						copy(jbytes[res.Index+len(raw):],
							jbytes[res.Index+len(res.Raw):])
					}
					return jbytes[:sz], nil
				}
				return []byte(jstr), nil
			}
			buf := make([]byte, 0, sz)
			buf = append(buf, jstr[:res.Index]...)
			if sc.stringify {
				buf = appendStringify(buf, raw)
			} else {
				buf = append(buf, raw...)
			}
			buf = append(buf, jstr[res.Index+len(res.Raw):]...)
			return buf, nil
		}
	}
	var paths []pathResult
	r, simple := parsePath(path, sc)
	if simple {
		paths = append(paths, r)
		for r.more {
			r, simple = parsePath(r.path, sc)
			if !simple {
				break
			}
			paths = append(paths, r)
		}
	}
	if !simple {
		if sc.del {
			return []byte(jstr),
				&errorType{"cannot delete value from a complex path"}
		}
		return setComplexPath(jstr, path, raw, sc.stringify)
	}
	njson, err := appendRawPaths(nil, jstr, paths, raw, sc)
	if err != nil {
		return []byte(jstr), err
	}
	return njson, nil
}

func setComplexPath(jstr, path, raw string, stringify bool) ([]byte, error) {
	res := Get(jstr, path)
	if !res.Exists() || !(res.Index != 0 || len(res.Indexes) != 0) {
		return []byte(jstr), errNoChange
	}
	if res.Index != 0 {
		njson := []byte(jstr[:res.Index])
		if stringify {
			njson = appendStringify(njson, raw)
		} else {
			njson = append(njson, raw...)
		}
		njson = append(njson, jstr[res.Index+len(res.Raw):]...)
		jstr = string(njson)
	}
	if len(res.Indexes) > 0 {
		type val struct {
			index int
			res   Result
		}
		vals := make([]val, 0, len(res.Indexes))
		res.ForEach(func(_, vres Result) bool {
			vals = append(vals, val{res: vres})
			return true
		})
		if len(res.Indexes) != len(vals) {
			return []byte(jstr), errNoChange
		}
		for i := 0; i < len(res.Indexes); i++ {
			vals[i].index = res.Indexes[i]
		}
		sort.SliceStable(vals, func(i, j int) bool {
			return vals[i].index > vals[j].index
		})
		for _, val := range vals {
			vres := val.res
			index := val.index
			njson := []byte(jstr[:index])
			if stringify {
				njson = appendStringify(njson, raw)
			} else {
				njson = append(njson, raw...)
			}
			njson = append(njson, jstr[index+len(vres.Raw):]...)
			jstr = string(njson)
		}
	}
	return []byte(jstr), nil
}

// Set sets a json value for the specified path.
// A path is in dot syntax, such as "name.last" or "age".
// This function expects that the json is well-formed, and does not validate.
// Invalid json will not panic, but it may return unexpected results.
// An error is returned if the path is not valid.
//
// A path is a series of keys separated by a dot.
//
//	{
//	  "name": {"first": "Tom", "last": "Anderson"},
//	  "age":37,
//	  "children": ["Sara","Alex","Jack"],
//	  "friends": [
//	    {"first": "James", "last": "Murphy"},
//	    {"first": "Roger", "last": "Craig"}
//	  ]
//	}
//	"name.last"          >> "Anderson"
//	"age"                >> 37
//	"children.1"         >> "Alex"
func Set(json, path string, value interface{}, options ...SetOptions) (string, error) {
	opts := SetOptions{}
	if len(options) > 0 {
		opts = options[0]
	}
	if opts.ReplaceInPlace {
		// it's not safe to replace bytes in-place for strings
		// copy the Options and set options.ReplaceInPlace to false.
		opts.ReplaceInPlace = false
	}
	jsonh := *(*stringHeader)(unsafe.Pointer(&json))
	jsonbh := sliceHeader{data: jsonh.data, len: jsonh.len, cap: jsonh.len}
	jsonb := *(*[]byte)(unsafe.Pointer(&jsonbh))
	res, err := SetBytes(jsonb, path, value, opts)
	return string(res), err
}

// SetBytes sets a json value for the specified path.
// If working with bytes, this method preferred over
// Set(string(data), path, value)
func SetBytes(json []byte, path string, value interface{}, options ...SetOptions) ([]byte, error) {
	sc := makeSetConfig(false, false, false, false)

	if len(options) > 0 {
		sc.optimistic = options[0].Optimistic
		sc.inplace = options[0].ReplaceInPlace
		sc.PathOption = options[0].PathOption
	}
	jstr := *(*string)(unsafe.Pointer(&json))
	var raw string

	switch v := value.(type) {
	default:
		b, merr := jsongo.Marshal(value)
		if merr != nil {
			return nil, merr
		}
		raw = *(*string)(unsafe.Pointer(&b))
	case dtype:
		raw = ""
		sc.del = true
	case string:
		raw = v
		sc.stringify = true
	case []byte:
		raw = *(*string)(unsafe.Pointer(&v))
		sc.stringify = true
	case bool:
		raw = If(v, "true", "false")
	case int8:
		raw = strconv.FormatInt(int64(v), 10)
	case int16:
		raw = strconv.FormatInt(int64(v), 10)
	case int32:
		raw = strconv.FormatInt(int64(v), 10)
	case int64:
		raw = strconv.FormatInt(v, 10)
	case uint8:
		raw = strconv.FormatUint(uint64(v), 10)
	case uint16:
		raw = strconv.FormatUint(uint64(v), 10)
	case uint32:
		raw = strconv.FormatUint(uint64(v), 10)
	case uint64:
		raw = strconv.FormatUint(v, 10)
	case float32:
		raw = strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		raw = strconv.FormatFloat(v, 'f', -1, 64)
	}

	res, err := set(jstr, path, raw, sc)
	if err == errNoChange {
		return json, nil
	}
	return res, err
}

// If returns a if v is true, else returns b.
func If(v bool, a, b string) string {
	if v {
		return a
	}

	return b
}

// SetRawBytes sets a raw json value for the specified path.
// If working with bytes, this method preferred over
// SetRaw(string(data), path, value)
func SetRawBytes(json []byte, path string, value []byte, options ...SetOptions) ([]byte, error) {
	jstr := *(*string)(unsafe.Pointer(&json))
	vstr := *(*string)(unsafe.Pointer(&value))
	sc := makeSetConfig(false, false, false, false)
	if len(options) > 0 {
		sc.optimistic = options[0].Optimistic
		sc.inplace = options[0].ReplaceInPlace
	}

	res, err := set(jstr, path, vstr, sc)
	if err == errNoChange {
		return json, nil
	}
	return res, err
}
