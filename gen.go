package jj

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bingoohuang/jj/reggen"
	"io"
	"log"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
)

var DefaultGen = NewGenContext()

func init() {
	DefaultGen.RegisterFn("random_int", RandomInt)
	DefaultGen.RegisterFn("random_bool", RandomBool)
	DefaultGen.RegisterFn("random_time", RandomTime)
	DefaultGen.RegisterFn("objectId", ObjectId)
	DefaultGen.RegisterFn("random", Random)
	DefaultGen.RegisterFn("regex", Regex)
}

type SubstitutionFnMap map[string]SubstitutionFn

type GenRun struct {
	Src           string
	Out           string
	Opens         int
	repeater      *Repeater
	BreakRepeater bool

	*GenContext
	repeaterWait bool
}

type GenContext struct {
	MockTimes       int
	SubstitutionFns SubstitutionFnMap
}

func NewGenContext() *GenContext {
	return &GenContext{
		SubstitutionFns: map[string]SubstitutionFn{},
	}
}

func (r *GenRun) walk(start, end, info int) int {
	element := r.Src[start:end]

	switch {
	case IsToken(info, TokOpen):
		r.Opens++
		if r.repeater != nil {
			src := r.Src[start:]
			r.repeater.Repeat(func(last bool) {
				p, _ := r.GenContext.Process(src)
				if r.Out += p.Out; !last {
					r.Out += ","
				}
			})
			r.repeater = nil
			r.BreakRepeater = true
			return -1
		}
	case IsToken(info, TokClose):
		r.Opens--
		if r.BreakRepeater {
			r.BreakRepeater = false
			return Ifi(r.Opens > 0, 1, 0)
		}
	case IsToken(info, TokString):
		s := element[1 : len(element)-1]
		switch {
		case r.repeater == nil:
			r.repeater = r.parseRepeat(s)
			if r.repeaterWait = r.repeater != nil && r.repeater.Key == ""; r.repeaterWait {
				return 1
			}

			if r.repeater != nil && r.repeater.Key != "" {
				r.Out += strconv.Quote(r.repeater.Key)
				return 1
			}

			fallthrough
		case IsToken(info, TokValue):
			if subs := ParseSubstitutes(s); subs.CountVars() > 0 {
				if r.repeater == nil {
					r.Out += subs.Eval(r.SubstitutionFns, true)
					return 1
				}

				repeatedValue := ""
				r.repeater.Repeat(func(last bool) {
					if r.repeater.Key == "" {
						if r.Out += subs.Eval(r.SubstitutionFns, true); !last {
							r.Out += ","
						}
					} else {
						repeatedValue += subs.Eval(r.SubstitutionFns, false)
					}
				})
				if r.repeater.Key != "" {
					r.Out += strconv.Quote(repeatedValue)
				}

				r.repeater = nil
				return -1
			} else if r.repeater != nil {
				r.repeatStr(element)
				return 1
			}
		}
	case IsToken(info, TokValue) && r.repeater != nil:
		r.repeater.Repeat(func(last bool) {
			if r.Out += element; !last {
				r.Out += ","
			}
		})
		r.repeater = nil
		return 1
	}

	if r.repeater == nil || !r.repeaterWait {
		r.Out += element
	}
	return Ifi(r.Opens > 0, 1, 0)
}

func (r *GenRun) repeatStr(element string) {
	s := element[1 : len(element)-1]
	repeatedValue := ""
	r.repeater.Repeat(func(last bool) {
		if r.repeater.Key == "" {
			if r.Out += element; !last {
				r.Out += ","
			}
		} else {
			repeatedValue += s
		}
	})

	if r.repeater.Key != "" {
		r.Out += strconv.Quote(repeatedValue)
	}

	r.repeater = nil
}

type Sub interface {
	IsVar() bool
}

type Subs []Sub

func (s Subs) CountVars() (count int) {
	for _, sub := range s {
		if sub.IsVar() {
			count++
		}
	}

	return
}

type Valuer interface {
	Value(name, params string) interface{}
}

func (r SubstitutionFnMap) Value(name, params string) interface{} {
	if f, ok := r[name]; ok {
		return f(params)
	}

	return ""
}

func (s Subs) Eval(valuer Valuer, quote bool) string {
	if len(s) == 1 && s.CountVars() == len(s) {
		v := s[0].(*SubVar)
		return convertValue(valuer, v, quote)
	}

	value := ""
	for _, sub := range s {
		switch v := sub.(type) {
		case *SubLiteral:
			value += v.Val
		case *SubVar:
			value += convertValue(valuer, v, false)
		}
	}

	if quote {
		return strconv.Quote(value)
	}

	return value
}

func convertValue(valuer Valuer, v *SubVar, quote bool) string {
	value := valuer.Value(v.Name, v.Params)
	switch vv := value.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", vv)
	case float32, float64:
		return fmt.Sprintf("%f", vv)
	case bool:
		return fmt.Sprintf("%t", vv)
	case string:
		if quote {
			return strconv.Quote(vv)
		} else {
			return vv
		}
	default:
		vvv := fmt.Sprintf("%v", value)
		if quote {
			return strconv.Quote(vvv)
		} else {
			return vvv
		}
	}
}

type SubLiteral struct {
	Val string
}

func (s SubLiteral) IsVar() bool { return false }

type SubVar struct {
	Name   string
	Params string
}

func (s SubVar) IsVar() bool { return true }

func ParseSubstitutes(src string) Subs {
	s := src
	var subs []Sub
	left := ""
	for {
		a := strings.IndexByte(s, '@')
		if a < 0 || a == len(s)-1 {
			left += s
			break
		}

		left += s[:a]

		a++
		s = s[a:]
		if s[0] == '@' {
			s = s[1:]
			left += "@"
		} else if s[0] == '{' {
			if rb := strings.IndexByte(s, '}'); rb > 0 {
				fn := s[1:rb]
				s = s[rb+1:]

				subLiteral, subVar := parseName(&fn, &left)
				if subLiteral != nil {
					subs = append(subs, subLiteral)
				}
				if subVar != nil {
					subs = append(subs, subVar)
				}
			}
		} else {
			subLiteral, subVar := parseName(&s, &left)
			if subLiteral != nil {
				subs = append(subs, subLiteral)
			}
			if subVar != nil {
				subs = append(subs, subVar)
			}
		}
	}

	if left != "" {
		subs = append(subs, &SubLiteral{Val: left})
	}

	if Subs(subs).CountVars() == 0 {
		return []Sub{&SubLiteral{Val: src}}
	}

	return subs
}

func parseName(s *string, left *string) (subLiteral, subVar Sub) {
	name := ""
	offset := 0
	for i, r := range *s {
		offset = i
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-') {
			name = (*s)[:i]
			break
		}
	}

	nonParam := false
	if name == "" && offset == len(*s)-1 {
		nonParam = true
		offset++
		name = *s
	}

	if *left != "" {
		subLiteral = &SubLiteral{Val: *left}
		*left = ""
	}

	sv := &SubVar{
		Name: name,
	}
	subVar = sv

	if !nonParam && offset > 0 && offset < len(*s) {
		if (*s)[offset] == '(' {
			if rb := strings.IndexByte(*s, ')'); rb > 0 {
				sv.Params = (*s)[offset+1 : rb]
				*s = (*s)[rb+1:]
				return
			}
		}
	}

	*s = (*s)[offset:]

	return
}

type Repeater struct {
	Key   string
	Times int
}

func (r Repeater) Repeat(f func(last bool)) {
	for i := 0; i < r.Times; i++ {
		f(i == r.Times-1)
	}
}

func (r *GenContext) parseRepeat(s string) *Repeater {
	p := strings.Index(s, "|")
	if p < 0 {
		return nil
	}

	key, s := s[:p], s[p+1:]
	_, _, times, err := parseRandSize(s)
	if err != nil {
		return nil
	}

	times = Ifi(r.MockTimes > 0, r.MockTimes, times)
	return &Repeater{Key: key, Times: times}
}

func parseRandSize(s string) (from, to, time int, err error) {
	p := strings.Index(s, "-")
	times := 0
	if p < 0 {
		if times, err = strconv.Atoi(s); err != nil {
			return 0, 0, 0, err
		}
		return times, times, times, nil
	}

	from, err1 := strconv.Atoi(s[:p])
	if err1 != nil {
		return 0, 0, 0, err1
	}
	to, err2 := strconv.Atoi(s[p+1:])
	if err2 != nil {
		return 0, 0, 0, err2
	}
	times = RandBetween(from, to)
	return from, to, times, nil
}

type SubstitutionFn func(args string) interface{}

func (r *GenContext) RegisterFn(fn string, f SubstitutionFn) { r.SubstitutionFns[fn] = f }

func Gen(src string) string { return DefaultGen.Gen(src) }

func (r *GenContext) Gen(src string) string {
	p, _ := r.Process(src)
	return p.Out
}

func (r *GenContext) Process(src string) (*GenRun, int) {
	gr := &GenRun{Src: src, GenContext: r}
	ret := StreamParse([]byte(src), gr.walk)
	return gr, ret
}

func Ifi(b bool, x, y int) int {
	if b {
		return x
	}
	return y
}

func RandInt() int {
	// calculate the max we will be using
	bg := big.NewInt(math.MaxInt32)

	// get big.Int between 0 and bg
	// in this case 0 to 20
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		panic(err)
	}

	return int(n.Int64())
}

func RandInt64() int64 {
	// calculate the max we will be using
	bg := big.NewInt(math.MaxInt64)

	// get big.Int between 0 and bg
	// in this case 0 to 20
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		panic(err)
	}

	return n.Int64()
}

func RandBetweenInt64(min, max int64) int64 {
	// calculate the max we will be using
	bg := big.NewInt(max - min + 1)

	// get big.Int between 0 and bg
	// in this case 0 to 20
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		panic(err)
	}

	// add n to min to support the passed in range
	return n.Int64() + min
}

func RandBetween(min, max int) int {
	// calculate the max we will be using
	bg := big.NewInt(int64(max - min + 1))

	// get big.Int between 0 and bg
	// in this case 0 to 20
	n, err := rand.Int(rand.Reader, bg)
	if err != nil {
		panic(err)
	}

	// add n to min to support the passed in range
	return int(n.Int64() + int64(min))
}

func ParseParams(params string) []string {
	params = strings.TrimSpace(params)
	sep := ","
	if strings.HasPrefix(params, "sep=") {
		if idx := strings.Index(params, " "); idx > 0 {
			sep = params[4:idx]
			params = params[idx+1:]
		}
	}

	return strings.Split(params, sep)
}

var timeFormatConvert = map[*regexp.Regexp]string{
	regexp.MustCompile(`(?i)yyyy`): "2006",
	regexp.MustCompile(`MM`):       "01",
	regexp.MustCompile(`(?i)dd`):   "02",
	regexp.MustCompile(`(?i)hh`):   "15",
	regexp.MustCompile(`mm`):       "04",
	regexp.MustCompile(`(?i)sss`):  "000",
	regexp.MustCompile(`(?i)ss`):   "05",
}

func ConvertTimeLayout(s string) string {
	for r, f := range timeFormatConvert {
		s = r.ReplaceAllString(s, f)
	}

	return s
}

func RandomTime(args string) interface{} {
	if args == "" {
		return time.Now().Format(time.RFC3339Nano)
	}

	pp := ParseParams(args)
	layout := ConvertTimeLayout(pp[0])
	if len(pp) == 1 {
		return time.Now().Format(layout)
	}

	if len(pp) == 3 {
		from, err := time.ParseInLocation(layout, pp[1], time.Local)
		if err != nil {
			log.Printf("failed to parse %s by layout %s, error:%v", pp[1], pp[0], err)
			return time.Now().Format(time.RFC3339Nano)
		}
		to, err := time.ParseInLocation(layout, pp[2], time.Local)
		if err != nil {
			log.Printf("failed to parse %s by layout %s, error:%v", pp[2], pp[0], err)
			return time.Now().Format(time.RFC3339Nano)
		}

		fromUnix := from.Unix()
		toUnix := to.Unix()
		r := RandBetweenInt64(fromUnix, toUnix)
		return time.Unix(r, 0).Format(layout)
	}

	return time.Now().Format(time.RFC3339Nano)
}

func RandomBool(args string) interface{} {
	return RandBetween(0, 1) == 0
}

func RandomInt(args string) interface{} {
	if args == "" {
		return RandInt()
	}

	if i, err := strconv.Atoi(args); err == nil {
		return RandBetween(0, i)
	}

	if from, to, _, err := parseRandSize(args); err == nil {
		return RandBetween(from, to)
	}

	var err error
	vv := 0
	count := 0
	for _, el := range strings.Split(args, ",") {
		v := strings.TrimSpace(el)
		if v == "" {
			continue
		} else if RandBetween(0, 1) == 0 {
			continue
		}

		if vv, err = strconv.Atoi(v); err == nil {
			return vv
		}
		count++
	}

	if count > 0 {
		return vv
	}

	return RandInt()
}

func Random(args string) interface{} {
	if args == "" {
		return RandStr(10)
	}
	if i, err := strconv.Atoi(args); err == nil {
		return RandStr(i)
	}

	if _, _, times, err := parseRandSize(args); err == nil {
		return RandStr(times)
	}

	lastEl := ""
	for _, el := range strings.Split(args, ",") {
		if lastEl = strings.TrimSpace(el); lastEl == "" {
			continue
		}

		if RandBetween(0, 1) == 0 {
			return el
		}
	}

	if lastEl != "" {
		return lastEl
	}

	return RandStr(10)
}

func Regex(args string) interface{} {
	g, err := reggen.Generate(args, 100)
	if err != nil {
		log.Printf("bad regex: %s, err: %v", args, err)
	}
	return g
}

// RandStr copy from https://stackoverflow.com/a/50581165.
func RandStr(n int) string {
	buff := make([]byte, n)
	rand.Read(buff)
	// Base 64 can be longer than len
	return base64.RawURLEncoding.EncodeToString(buff)[:n]
}

// ObjectID is the BSON ObjectID type.
type ObjectID [12]byte

var objectIDCounter = readRandomUint32()
var processUnique = processUniqueBytes()

func ObjectId(string) interface{} { return NewObjectID().Hex() }

// NewObjectID generates a new ObjectID.
func NewObjectID() ObjectID {
	return NewObjectIDFromTimestamp(time.Now())
}

// NewObjectIDFromTimestamp generates a new ObjectID based on the given time.
func NewObjectIDFromTimestamp(timestamp time.Time) ObjectID {
	var b [12]byte

	binary.BigEndian.PutUint32(b[0:4], uint32(timestamp.Unix()))
	copy(b[4:9], processUnique[:])
	putUint24(b[9:12], atomic.AddUint32(&objectIDCounter, 1))

	return b
}

// Timestamp extracts the time part of the ObjectId.
func (id ObjectID) Timestamp() time.Time {
	unixSecs := binary.BigEndian.Uint32(id[0:4])
	return time.Unix(int64(unixSecs), 0).UTC()
}

// Hex returns the hex encoding of the ObjectID as a string.
func (id ObjectID) Hex() string {
	return hex.EncodeToString(id[:])
}

func processUniqueBytes() [5]byte {
	var b [5]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %v", err))
	}

	return b
}

func readRandomUint32() uint32 {
	var b [4]byte
	_, err := io.ReadFull(rand.Reader, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %v", err))
	}

	return (uint32(b[0]) << 0) | (uint32(b[1]) << 8) | (uint32(b[2]) << 16) | (uint32(b[3]) << 24)
}

func putUint24(b []byte, v uint32) {
	b[0] = byte(v >> 16)
	b[1] = byte(v >> 8)
	b[2] = byte(v)
}
