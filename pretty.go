package jj

import (
	"sort"
)

// Options is Pretty options
type Options struct {
	// Width is an max column width for single line arrays
	// Default is 80
	Width int
	// Prefix is a prefix for all lines
	// Default is an empty string
	Prefix string
	// Indent is the nested indentation
	// Default is two spaces
	Indent string
	// SortKeys will sort the keys alphabetically
	// Default is false
	SortKeys bool
}

// DefaultOptions is the default options for pretty formats.
var DefaultOptions = Options{Width: 80, Prefix: "", Indent: "  ", SortKeys: false}

// Pretty converts the input json into a more human readable format where each
// element is on it's own line with clear indentation with customized options.
func Pretty(json []byte, options ...Options) []byte {
	opts := DefaultOptions
	if len(options) > 0 {
		opts = options[0]
	}

	buf := make([]byte, 0, len(json))
	if len(opts.Prefix) > 0 {
		buf = append(buf, opts.Prefix...)
	}

	totalBuf := make([]byte, 0, len(json)*2)
	for i := 0; i < len(json); {
		buf = buf[0:0]
		buf, i, _, _ = appendPrettyAny(buf, json, i, makePrettyOption(true,
			opts.Width, opts.Prefix, opts.Indent, opts.SortKeys,
			0, 0, -1))
		if len(buf) > 0 {
			buf = append(buf, '\n')
		} else {
			break
		}
		totalBuf = append(totalBuf, buf...)
	}
	return totalBuf
}

// Ugly removes insignificant space characters from the input json byte slice
// and returns the compacted result.
func Ugly(json []byte) []byte {
	return ugly(make([]byte, 0, len(json)), json)
}

// UglyInPlace removes insignificant space characters from the input json
// byte slice and returns the compacted result. This method reuses the
// input json buffer to avoid allocations. Do not use the original bytes
// slice upon return.
func UglyInPlace(json []byte) []byte { return ugly(json, json) }

func ugly(dst, src []byte) []byte {
	dst = dst[:0]
	l := len(src)
	for i := 0; i < l; i++ {
		if src[i] <= ' ' {
			continue
		}

		dst = append(dst, src[i])
		if src[i] != '"' {
			continue
		}

		for i = i + 1; i < l; i++ {
			dst = append(dst, src[i])
			if src[i] != '"' {
				continue
			}

			j := i - 1
			for ; ; j-- {
				if src[j] != '\\' {
					break
				}
			}
			if (j-i)%2 != 0 {
				break
			}
		}
	}
	return dst
}

type prettyOption struct {
	pretty         bool
	width          int
	prefix, indent string
	sortkeys       bool
	tabs, nl, max  int
}

func makePrettyOption(pretty bool, width int, prefix, indent string, sortkeys bool, tabs, nl, max int) prettyOption {
	return prettyOption{pretty: pretty, width: width, prefix: prefix, indent: indent, sortkeys: sortkeys, tabs: tabs, nl: nl, max: max}
}

func appendPrettyAny(buf, json []byte, i int, p prettyOption) ([]byte, int, int, bool) {
	for ; i < len(json); i++ {
		c := json[i]
		if c <= ' ' {
			continue
		}
		if c == '"' {
			return appendPrettyString(buf, json, i, p.nl)
		}
		if (c >= '0' && c <= '9') || c == '-' {
			return appendPrettyNumber(buf, json, i, p.nl)
		}

		switch c {
		case '{':
			return appendPrettyObject(buf, json, i, '{', '}', p)
		case '[':
			return appendPrettyObject(buf, json, i, '[', ']', p)
		case 't':
			return append(buf, 't', 'r', 'u', 'e'), i + 4, p.nl, true
		case 'f':
			return append(buf, 'f', 'a', 'l', 's', 'e'), i + 5, p.nl, true
		case 'n':
			return append(buf, 'n', 'u', 'l', 'l'), i + 4, p.nl, true
		}
	}
	return buf, i, p.nl, true
}

type pair struct {
	kstart, kend int
	vstart, vend int
}

type byKey struct {
	sorted bool
	json   []byte
	pairs  []pair
}

func (arr *byKey) Len() int {
	return len(arr.pairs)
}

func (arr *byKey) Less(i, j int) bool {
	key1 := arr.json[arr.pairs[i].kstart+1 : arr.pairs[i].kend-1]
	key2 := arr.json[arr.pairs[j].kstart+1 : arr.pairs[j].kend-1]
	return string(key1) < string(key2)
}

func (arr *byKey) Swap(i, j int) {
	arr.pairs[i], arr.pairs[j] = arr.pairs[j], arr.pairs[i]
	arr.sorted = true
}

func appendPrettyObject(buf, json []byte, i int, open, close byte, po prettyOption) ([]byte, int, int, bool) {
	var ok bool
	if po.width > 0 {
		if po.pretty && open == '[' && po.max == -1 {
			// here we try to create a single line array
			max := po.width - (len(buf) - po.nl)
			if max > 3 {
				s1, s2 := len(buf), i
				buf, i, _, ok = appendPrettyObject(buf, json, i, '[', ']',
					makePrettyOption(false, po.width, po.prefix, "", po.sortkeys, 0, 0, max))
				if ok && len(buf)-s1 <= max {
					return buf, i, po.nl, true
				}
				buf = buf[:s1]
				i = s2
			}
		} else if po.max != -1 && open == '{' {
			return buf, i, po.nl, false
		}
	}
	buf = append(buf, open)
	i++
	var pairs []pair
	if open == '{' && po.sortkeys {
		pairs = make([]pair, 0, 8)
	}
	var n int
	for ; i < len(json); i++ {
		if json[i] <= ' ' {
			continue
		}
		if json[i] == close {
			if po.pretty {
				if open == '{' && po.sortkeys {
					buf = sortPairs(json, buf, pairs)
				}
				if n > 0 {
					po.nl = len(buf)
					buf = append(buf, '\n')
				}
				if buf[len(buf)-1] != open {
					buf = appendTabs(buf, po.prefix, po.indent, po.tabs)
				}
			}
			buf = append(buf, close)
			return buf, i + 1, po.nl, open != '{'
		}
		if open == '[' || json[i] == '"' {
			if n > 0 {
				buf = append(buf, ',')
				if po.width != -1 && open == '[' {
					buf = append(buf, ' ')
				}
			}
			var p pair
			if po.pretty {
				po.nl = len(buf)
				buf = append(buf, '\n')
				if open == '{' && po.sortkeys {
					p.kstart = i
					p.vstart = len(buf)
				}
				buf = appendTabs(buf, po.prefix, po.indent, po.tabs+1)
			}
			if open == '{' {
				buf, i, po.nl, _ = appendPrettyString(buf, json, i, po.nl)
				if po.sortkeys {
					p.kend = i
				}
				buf = append(buf, ':')
				if po.pretty {
					buf = append(buf, ' ')
				}
			}
			buf, i, po.nl, ok = appendPrettyAny(buf, json, i,
				makePrettyOption(po.pretty, po.width, po.prefix, po.indent, po.sortkeys, po.tabs+1, po.nl, po.max))
			if po.max != -1 && !ok {
				return buf, i, po.nl, false
			}
			if po.pretty && open == '{' && po.sortkeys {
				p.vend = len(buf)
				if p.kstart > p.kend || p.vstart > p.vend {
					// bad data. disable sorting
					po.sortkeys = false
				} else {
					pairs = append(pairs, p)
				}
			}
			i--
			n++
		}
	}

	return buf, i, po.nl, open != '{'
}

func sortPairs(json, buf []byte, pairs []pair) []byte {
	if len(pairs) == 0 {
		return buf
	}
	vstart := pairs[0].vstart
	vend := pairs[len(pairs)-1].vend
	arr := byKey{false, json, pairs}
	sort.Sort(&arr)
	if !arr.sorted {
		return buf
	}
	nbuf := make([]byte, 0, vend-vstart)
	for i, p := range pairs {
		nbuf = append(nbuf, buf[p.vstart:p.vend]...)
		if i < len(pairs)-1 {
			nbuf = append(nbuf, ',')
			nbuf = append(nbuf, '\n')
		}
	}
	return append(buf[:vstart], nbuf...)
}

func appendPrettyString(buf, json []byte, i, nl int) ([]byte, int, int, bool) {
	s := i
	i++
	for ; i < len(json); i++ {
		if json[i] != '"' {
			continue
		}

		var sc int
		for j := i - 1; j > s; j-- {
			if json[j] == '\\' {
				sc++
			} else {
				break
			}
		}
		if sc%2 == 1 {
			continue
		}
		i++
		break
	}
	return append(buf, json[s:i]...), i, nl, true
}

func appendPrettyNumber(buf, json []byte, i, nl int) ([]byte, int, int, bool) {
	s := i
	i++
	for ; i < len(json); i++ {
		c := json[i]
		if c <= ' ' || c == ',' || c == ':' || c == ']' || c == '}' {
			break
		}
	}
	return append(buf, json[s:i]...), i, nl, true
}

func appendTabs(buf []byte, prefix, indent string, tabs int) []byte {
	if len(prefix) != 0 {
		buf = append(buf, prefix...)
	}
	if len(indent) == 2 && indent[0] == ' ' && indent[1] == ' ' {
		for i := 0; i < tabs; i++ {
			buf = append(buf, ' ', ' ')
		}
	} else {
		for i := 0; i < tabs; i++ {
			buf = append(buf, indent...)
		}
	}
	return buf
}

// Style is the color style
type Style struct {
	Key, String, Number []string
	True, False, Null   []string
	Append              func(dst []byte, c byte) []byte
}

func hexp(p byte) byte {
	switch {
	case p < 10:
		return p + '0'
	default:
		return (p - 10) + 'a'
	}
}

// TerminalStyle is for terminals
var TerminalStyle = &Style{
	Key:    []string{"\x1B[94m", "\x1B[0m"},
	String: []string{"\x1B[92m", "\x1B[0m"},
	Number: []string{"\x1B[93m", "\x1B[0m"},
	True:   []string{"\x1B[96m", "\x1B[0m"},
	False:  []string{"\x1B[96m", "\x1B[0m"},
	Null:   []string{"\x1B[91m", "\x1B[0m"},
	Append: func(dst []byte, c byte) []byte {
		if c < ' ' && (c != '\r' && c != '\n' && c != '\t' && c != '\v') {
			dst = append(dst, "\\u00"...)
			dst = append(dst, hexp((c>>4)&0xF))
			return append(dst, hexp((c)&0xF))
		}
		return append(dst, c)
	},
}

// Color will colorize the json. The style parma is used for customizing
// the colors. Passing nil to the style param will use the default
// TerminalStyle.
func Color(src []byte, style *Style) []byte {
	if style == nil {
		style = TerminalStyle
	}
	apnd := style.Append
	if apnd == nil {
		apnd = func(dst []byte, c byte) []byte {
			return append(dst, c)
		}
	}
	type stackt struct {
		kind byte
		key  bool
	}
	var dst []byte
	var stack []stackt
	for i := 0; i < len(src); i++ {
		c := src[i]
		if c == '"' {
			key := len(stack) > 0 && stack[len(stack)-1].key
			if key {
				dst = append(dst, style.Key[0]...)
			} else {
				dst = append(dst, style.String[0]...)
			}
			dst = apnd(dst, '"')
			for i = i + 1; i < len(src); i++ {
				dst = apnd(dst, src[i])
				if src[i] == '"' {
					j := i - 1
					for ; ; j-- {
						if src[j] != '\\' {
							break
						}
					}
					if (j-i)%2 != 0 {
						break
					}
				}
			}
			if key {
				dst = append(dst, style.Key[1]...)
			} else {
				dst = append(dst, style.String[1]...)
			}
		} else if c == '{' || c == '[' {
			stack = append(stack, stackt{c, c == '{'})
			dst = apnd(dst, c)
		} else if (c == '}' || c == ']') && len(stack) > 0 {
			stack = stack[:len(stack)-1]
			dst = apnd(dst, c)
		} else if (c == ':' || c == ',') && len(stack) > 0 && stack[len(stack)-1].kind == '{' {
			stack[len(stack)-1].key = !stack[len(stack)-1].key
			dst = apnd(dst, c)
		} else {
			var kind byte
			if (c >= '0' && c <= '9') || c == '-' {
				kind = '0'
				dst = append(dst, style.Number[0]...)
			} else if c == 't' {
				kind = 't'
				dst = append(dst, style.True[0]...)
			} else if c == 'f' {
				kind = 'f'
				dst = append(dst, style.False[0]...)
			} else if c == 'n' {
				kind = 'n'
				dst = append(dst, style.Null[0]...)
			} else {
				dst = apnd(dst, c)
			}
			if kind != 0 {
			FOR:
				for ; i < len(src); i++ {
					switch src[i] {
					case ' ', ',', ':', ']', '}':
						i--
						break FOR
					}
					dst = apnd(dst, src[i])
				}
				switch kind {
				case '0':
					dst = append(dst, style.Number[1]...)
				case 't':
					dst = append(dst, style.True[1]...)
				case 'f':
					dst = append(dst, style.False[1]...)
				case 'n':
					dst = append(dst, style.Null[1]...)
				}
			}
		}
	}
	return dst
}
