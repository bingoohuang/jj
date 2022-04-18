package jj

import (
	"math/rand"
	"strconv"
)

// RandOptions for Make()
type RandOptions struct {
	// Pretty formats and indents the random json. Default true
	Pretty bool
	// Words is the number of unique words to use. Default 1,000
	Words int
	// Depth is the maximum of nested child elements
	Depth int
	// Rand is the random number generator to use. Default global rng
	Rand *rand.Rand
}

// DefaultRandOptions for Make()
var DefaultRandOptions = RandOptions{
	Pretty: true,
	Words:  1000,
	Depth:  5,
	Rand:   nil,
}

// Rand returns a random json document. The depth param is the maximum nested
// depth of json document
func Rand(opts ...RandOptions) []byte {
	return RandAppend(nil, opts...)
}

// RandAppend appends a random json document to dst. The depth param is the maximum nested
// depth of json document
func RandAppend(dst []byte, optss ...RandOptions) []byte {
	var opts RandOptions
	if len(optss) == 0 {
		opts = DefaultRandOptions
	} else {
		opts = optss[0]
	}

	var p float64
	if opts.Words > len(Words) {
		p = 1.0
	} else if opts.Words < 1 {
		p = 1 / float64(len(Words))
	} else {
		p = float64(opts.Words) / float64(len(Words))
	}
	s := int(float64(len(Words)) * p)
	t := len(Words) / s
	mark := len(dst)
	dst = appendRandObject(dst, opts.Rand, s, t, opts.Depth)
	if opts.Pretty {
		dst = append(dst[:mark], Pretty(dst[mark:])...)
	}
	return dst
}

func randInt(rng *rand.Rand) int {
	if rng == nil {
		return rand.Int()
	}
	return rng.Int()
}

func appendRandString(dst []byte, rng *rand.Rand, s, t int) []byte {
	dst = append(dst, '"')
	dst = append(dst, Words[(randInt(rng)%s)*t]...)
	return append(dst, '"')
}

func appendRandAny(dst []byte, rng *rand.Rand, nested bool, s, t, d int) []byte {
	switch randInt(rng) % 7 {
	case 0:
		dst = appendRandString(dst, rng, s, t)
	case 1:
		if !nested {
			dst = appendRandAny(dst, rng, nested, s, t, d)
		} else {
			dst = append(dst, '[')
			if d > 1 {
				n := randInt(rng) % (d - 1)
				for i := 0; i < n; i++ {
					if i > 0 {
						dst = append(dst, ',')
					}
					dst = appendRandAny(dst, rng, false, s, t, d-1)
				}
			}
			dst = append(dst, ']')
		}
	case 2:
		if !nested {
			dst = appendRandAny(dst, rng, nested, s, t, d)
		} else {
			if d > 1 {
				d = randInt(rng) % (d - 1)
			}
			dst = appendRandObject(dst, rng, s, t, d)
		}
	case 3:
		dst = strconv.AppendFloat(dst, float64(randInt(rng)%10000)/100, 'f', 2, 64)
	case 4:
		dst = append(dst, "true"...)
	case 5:
		dst = append(dst, "false"...)
	case 6:
		dst = append(dst, "null"...)
	}
	return dst
}

func appendRandObject(dst []byte, rng *rand.Rand, s, t, d int) []byte {
	dst = append(dst, '{')
	for i := 0; i < d; i++ {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendRandString(dst, rng, s, t)
		dst = append(dst, ':')
		dst = appendRandAny(dst, rng, true, s, t, d-1)
	}
	return append(dst, '}')
}
