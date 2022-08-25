// Package jj provides a fast way to validate the JSON and protect against
// vulnerable JSON content-level attacks (JSON Threat Protection)
// based on configured properties.
package jj

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// Option Function Parameters to creates verifier
type Option func(*Verify)

const (
	objectKeyValueLength = "maxKeyLenReached"
	stringValueLength    = "maxStringLenReached"
)

// ErrInvalidJSON denotes JSON is Malformed.
var ErrInvalidJSON = errors.New("jtp.MalformedJSON")

// Verifier is the interface that wraps the basic
// VerifyBytes and VerifyString methods.
type Verifier interface {
	VerifyBytes([]byte) error
	VerifyString(string) error
}

// Verify Configuration Parameters.
// Verify must be created with NewJtp function.
//
//	 // with some options
//		_ = NewJtp(
//				 WithMaxArrayLen(6),
//				 WithMaxDepth(7),
//				 WithMaxKeyLen(20), WithMaxStringLen(50),
//				 )
//
//	 // with single option
//		_ = NewJtp(WithMaxStringLen(25))
//
// Exported variable are for logging and reference.
type Verify struct {
	// MaxArrayLen specifies the maximum number of elements allowed in an array.
	MaxArrayLen int
	// MaxDepth specifies the maximum allowed containment depth, where the containers are objects or arrays.
	MaxDepth int

	// MaxEntryCount specifies the maximum number of entries allowed in an object
	MaxEntryCount int
	// MaxKeyLen specifies the maximum string length allowed for a property name within an object.
	MaxKeyLen int
	// MaxStringLen specifies the maximum length allowed for a string value.
	MaxStringLen int
}

// NewJtp creates and return a Verifier with passed Option Parameters,
// with default UTF-8 text encoding.
func NewJtp(opt ...Option) Verifier {
	v := &Verify{}
	for _, setter := range opt {
		setter(v)
	}

	return *v
}

// WithMaxArrayLen Option
// Specifies the maximum number of entries (
// comma delimited values)  allowed in an array.
// zero value disable the check.
func WithMaxArrayLen(l int) Option { return func(v *Verify) { v.MaxArrayLen = l } }

// WithMaxDepth Option
// Specifies the maximum allowed nested containers depth, within a JSON
// where the containers are objects or arrays.
// zero value disable the checks
func WithMaxDepth(l int) Option { return func(v *Verify) { v.MaxDepth = l } }

// WithMaxKeyLen Option
// Specifies the maximum number of characters (UTF-8 encoded)
// allowed for a property(key) name within an object.
// zero value disable the checks
func WithMaxKeyLen(l int) Option { return func(v *Verify) { v.MaxKeyLen = l } }

// WithMaxStringLen Option
// Specifies the maximum number of characters  (
// UTF-8 encoded) in a string value.
// zero value disable the checks
func WithMaxStringLen(l int) Option { return func(v *Verify) { v.MaxStringLen = l } }

// WithMaxEntryCount Option
// Specifies the maximum number of entries
// (comma delimited string:value pairs) in a single object
// zero value disable the checks
func WithMaxEntryCount(l int) Option { return func(v *Verify) { v.MaxEntryCount = l } }

func validateStringLen(data []byte, startIndex, endIndex, maxAllowed int, strType string) (err error) {
	str := data[startIndex:endIndex]
	// JSON exchange in an open ecosystem must be encoded in UTF-8.
	// https://tools.ietf.org/html/rfc8259#section-8.1
	l := utf8.RuneCount(str)
	// -2 for double quote validation skew in length
	if maxAllowed > 0 && l-2 > maxAllowed {
		err = fmt.Errorf("jtp.%s.Max-[%d]-Allowed.Found-[%d]: %w", strType, maxAllowed, l-2, ErrInvalidJSON)
	}
	return
}

// isValidateString checks if the string is valid or not
func isValidateString(data []byte, i int) (outi int,
	ok bool,
) {
	for ; i < len(data); i++ {
		if data[i] < ' ' {
			return i, false
		} else if data[i] == '\\' {
			//
			i++
			if i == len(data) {
				return i, false
			}
			switch data[i] {
			default:
				return i, false
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
			case 'u':
				for j := 0; j < 4; j++ {
					i++
					if i >= len(data) {
						return i, false
					}
					if !((data[i] >= '0' && data[i] <= '9') ||
						(data[i] >= 'a' && data[i] <= 'f') ||
						(data[i] >= 'A' && data[i] <= 'F')) {
						return i, false
					}
				}
			}
		} else if data[i] == '"' {
			return i + 1, true
		}
	}
	return i, false
}

func (v *Verify) isValidArray(data []byte, i int, depth *int) (outi int, ok bool, err error) {
	if v.MaxDepth > 0 && v.MaxDepth < *depth {
		return i, false,
			fmt.Errorf("jtp.maxDepthReached.Max-[%d]-Allowed.Found-[%d]: %w", v.MaxDepth, *depth, ErrInvalidJSON)
	}
	for ; i < len(data); i++ {
		child := 0
		switch data[i] {
		default:
			for ; i < len(data); i++ {
				// can contain Any value
				if i, ok, err = v.validateAny(data, i, depth); !ok {
					return i, false, err
				}
				// children
				i, ok = isValidComma(data, i, ']')
				if !ok {
					return i, false, err
				}
				child++
				if v.MaxArrayLen > 0 && child > v.MaxArrayLen {
					return i, false,
						fmt.Errorf("jtp.maxArrayLenReached.Max-[%d]-Allowed.Found-[%d]: %w", v.MaxArrayLen, child, ErrInvalidJSON)
				}
				if data[i] == ']' {
					*depth--
					return i + 1, true, err
				}
			}
		case ' ', '\t', '\n', '\r':
			continue
		case ']':
			*depth--
			return i + 1, true, err
		}
	}
	return i, false, err
}

func (v *Verify) isValidObject(data []byte, i int, depth *int) (outi int, ok bool, err error) {
	if v.MaxDepth > 0 && v.MaxDepth < *depth {
		return i, false,
			fmt.Errorf("jtp.maxDepthReached.Max-[%d]-Allowed.Found-[%d]: %w", v.MaxDepth, *depth, ErrInvalidJSON)
	}
	for ; i < len(data); i++ {
		switch data[i] {
		default:
			return i, false, err
		case ' ', '\t', '\n', '\r':
			continue
		case '}':
			*depth--
			return i + 1, true, err
		case '"':
			// entries
			entries := 0
		key:
			// key should be string
			tempI := i // for string length
			i, ok = isValidateString(data, i+1)
			if !ok {
				return i, false, err
			}
			entries++

			// check for entries count
			if v.MaxEntryCount > 0 && v.MaxEntryCount < entries {
				return i, false,
					fmt.Errorf("jtp.maxEntryCountReached.Max-[%d]-Allowed.Found-[%d]: %w", v.MaxEntryCount, entries, ErrInvalidJSON)
			}

			if ok { // validate key length
				err = validateStringLen(data, tempI, i, v.MaxKeyLen, objectKeyValueLength)
				if err != nil {
					// no further json verification done
					return i, false, err
				}
			}

			// key should be followed by :
			if i, ok = isValidColon(data, i); !ok {
				return i, false, err
			}
			// followed by Any Value
			if i, ok, err = v.validateAny(data, i, depth); !ok || err != nil {
				return i, false, err
			}

			if i, ok = isValidComma(data, i, '}'); !ok {
				return i, false, err
			}
			if data[i] == '}' {
				*depth--
				return i + 1, true, err
			}
			i++
			for ; i < len(data); i++ {
				switch data[i] {
				default:
					return i, false, err
				case ' ', '\t', '\n', '\r':
					continue
				case '"':
					goto key
				}
			}
			return i, false, err
		}
	}
	return i, false, err
}

func (v *Verify) validateAny(data []byte, i int, depth *int) (outi int, ok bool, err error) {
	if v.MaxDepth > 0 && v.MaxDepth < *depth {
		return i, false,
			fmt.Errorf("jtp.maxDepthReached.Max-[%d]-Allowed.Found-[%d]: %w", v.MaxDepth, *depth, ErrInvalidJSON)
	}
	for ; i < len(data); i++ {
		switch data[i] {
		default:
			return i, false, err
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			*depth++
			return v.isValidObject(data, i+1, depth)
		case '[':
			*depth++
			return v.isValidArray(data, i+1, depth)
		case '"':
			// validate string
			outi, ok = isValidateString(data, i+1)
			err = validateStringLen(data, i, outi, v.MaxStringLen, stringValueLength)
			return
		case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			outi, ok = isValidNumber(data, i+1)
			return
		case 't':
			outi, ok = isValidTrue(data, i+1)
			return
		case 'f':
			outi, ok = isValidFalse(data, i+1)
		case 'n':
			outi, ok = isValidNull(data, i+1)
			return
		}
	}
	return i, false, err
}

// HELPERS

func isValidTrue(d []byte, i int) (outi int, ok bool) {
	if i+3 <= len(d) && d[i] == 'r' && d[i+1] == 'u' && d[i+2] == 'e' {
		return i + 3, true
	}
	return i, false
}

func isValidFalse(d []byte, i int) (outi int, ok bool) {
	if i+4 <= len(d) && d[i] == 'a' && d[i+1] == 'l' && d[i+2] == 's' && d[i+3] == 'e' {
		return i + 4, true
	}
	return i, false
}

func isValidNull(d []byte, i int) (newI int, ok bool) {
	if i+3 <= len(d) && d[i] == 'u' && d[i+1] == 'l' && d[i+2] == 'l' {
		return i + 3, true
	}
	return i, false
}

func isValidNumber(data []byte, i int) (newI int, ok bool) {
	i--
	// sign
	if data[i] == '-' {
		i++
	}
	// int
	if i == len(data) {
		return i, false
	}
	if data[i] == '0' {
		i++
	} else {
		for ; i < len(data); i++ {
			if data[i] >= '0' && data[i] <= '9' {
				continue
			}
			break
		}
	}
	// frac
	if i == len(data) {
		return i, true
	}
	if data[i] == '.' {
		i++
		if i == len(data) {
			return i, false
		}
		if data[i] < '0' || data[i] > '9' {
			return i, false
		}
		i++
		for ; i < len(data); i++ {
			if data[i] >= '0' && data[i] <= '9' {
				continue
			}
			break
		}
	}
	// exp
	if i == len(data) {
		return i, true
	}
	if data[i] == 'e' || data[i] == 'E' {
		i++
		if i == len(data) {
			return i, false
		}
		if data[i] == '+' || data[i] == '-' {
			i++
		}
		if i == len(data) {
			return i, false
		}
		if data[i] < '0' || data[i] > '9' {
			return i, false
		}
		i++
		for ; i < len(data); i++ {
			if data[i] >= '0' && data[i] <= '9' {
				continue
			}
			break
		}
	}
	return i, true
}

func isValidComma(data []byte, i int, end byte) (outi int, ok bool) {
	for ; i < len(data); i++ {
		switch data[i] {
		default:
			return i, false
		case ' ', '\t', '\n', '\r':
			continue
		case ',':
			return i, true
		case end:
			return i, true
		}
	}
	return i, false
}

func isValidColon(data []byte, i int) (outi int, ok bool) {
	for ; i < len(data); i++ {
		switch data[i] {
		default:
			return i, false
		case ' ', '\t', '\n', '\r':
			continue
		case ':':
			return i + 1, true
		}
	}
	return i, false
}

func (v *Verify) isValidJSON(data []byte, i int, depth *int) (outi int, ok bool, err error) {
	for ; i < len(data); i++ {
		switch data[i] {
		default:
			i, ok, err = v.validateAny(data, i, depth)
			if !ok || err != nil {
				return i, false, err
			}
			for ; i < len(data); i++ {
				switch data[i] {
				default:
					return i, false, err
				case ' ', '\t', '\n', '\r':
					continue
				}
			}
			return i, true, err
		case ' ', '\t', '\n', '\r':
			continue
		}
	}
	return i, false, err
}

// VerifyBytes returns true if the input is valid json,
// and is JSON THREAT Protection Safe.
// A successful VerifyBytes returns err == nil,
func (v Verify) VerifyBytes(json []byte) error {
	var depth int
	_, ok, err := v.isValidJSON(json, 0, &depth)
	if err == nil && ok == false {
		err = ErrInvalidJSON
	}
	return err
}

// VerifyString returns true if the input is valid json,
// and is JSON THREAT Protection Safe.
// A successful VerifyString returns err == nil,
func (v Verify) VerifyString(json string) error {
	return v.VerifyBytes([]byte(json))
}
