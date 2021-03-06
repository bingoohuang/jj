package jj

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/bingoohuang/gg/pkg/chinaid"
	"github.com/bingoohuang/gg/pkg/randx"
	"github.com/bingoohuang/gg/pkg/timex"
	"github.com/bingoohuang/gg/pkg/vars"
	"github.com/bingoohuang/jj/reggen"
	"io"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var DefaultSubstituteFns = SubstituteFnMap(map[string]SubstitutionFn{
	"random":      Random,
	"random_int":  RandomInt,
	"random_bool": func(_ string) interface{} { return randx.Bool() },
	"random_time": RandomTime,
	"objectId":    func(string) interface{} { return NewObjectID().Hex() },
	"regex":       Regex,
	"uuid":        func(_ string) interface{} { return NewUUID().String() },

	"xx":   func(_ string) interface{} { return chinaid.RandChinese(2, 3) },
	"姓名":   func(_ string) interface{} { return chinaid.Name() },
	"性别":   func(_ string) interface{} { return chinaid.Sex() },
	"地址":   func(_ string) interface{} { return chinaid.Address() },
	"手机":   func(_ string) interface{} { return chinaid.Mobile() },
	"身份证":  func(_ string) interface{} { return chinaid.ChinaID() },
	"发证机关": func(_ string) interface{} { return chinaid.IssueOrg() },
	"邮箱":   func(_ string) interface{} { return chinaid.Email() },
	"银行卡":  func(_ string) interface{} { return chinaid.BankNo() },
})

type SubstituteFnMap map[string]SubstitutionFn

func (r SubstituteFnMap) Register(fn string, f SubstitutionFn) { r[fn] = f }

type Substitute interface {
	vars.Valuer
	Register(fn string, f SubstitutionFn)
}

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
	MockTimes int
	Substitute
}

var DefaultGen = NewGen()

func NewGenContext(s Substitute) *GenContext { return &GenContext{Substitute: s} }
func NewGen() *GenContext                    { return NewGenContext(DefaultSubstituteFns) }

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
			if subs := vars.ParseExpr(s); subs.CountVars() > 0 {
				if r.repeater == nil {
					r.Out += r.Eval(subs, true)
					return 1
				}

				repeatedValue := ""
				r.repeater.Repeat(func(last bool) {
					if r.repeater.Key == "" {
						if r.Out += r.Eval(subs, true); !last {
							r.Out += ","
						}
					} else {
						repeatedValue += r.Eval(subs, false)
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

func (r *GenRun) Eval(subs vars.Subs, quote bool) (s string) {
	result := subs.Eval(r.Substitute)
	if v, ok := result.(string); ok {
		if quote {
			return strconv.Quote(v)
		} else {
			return v
		}
	}

	return vars.ToString(result)
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

func (r SubstituteFnMap) Value(name, params string) interface{} {
	if f, ok := r[name]; ok {
		return f(params)
	}

	return ""
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

	n := Ifi(r.MockTimes > 0, r.MockTimes, int(times))
	return &Repeater{Key: key, Times: n}
}

func parseRandSize(s string) (from, to, time int64, err error) {
	p := strings.Index(s, "-")
	times := int64(0)
	if p < 0 {
		if times, err = strconv.ParseInt(s, 10, 64); err != nil {
			return 0, 0, 0, err
		}
		return times, times, times, nil
	}

	from, err1 := strconv.ParseInt(s[:p], 10, 64)
	if err1 != nil {
		return 0, 0, 0, err1
	}
	to, err2 := strconv.ParseInt(s[p+1:], 10, 64)
	if err2 != nil {
		return 0, 0, 0, err2
	}
	times = randx.Int64Between(from, to)
	return from, to, times, nil
}

type SubstitutionFn func(args string) interface{}

func (r *GenContext) RegisterFn(fn string, f SubstitutionFn) { r.Substitute.Register(fn, f) }

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

func ParseParams(params string) []string {
	params = strings.TrimSpace(params)
	sep := ","
	if strings.HasPrefix(params, "sep=") {
		if idx := strings.Index(params, " "); idx > 0 {
			sep = params[4:idx]
			params = params[idx+1:]
		}
	}

	return SplitTrim(params, sep)
}

func SplitTrim(s, sep string) []string {
	pp := strings.Split(s, sep)
	p2 := make([]string, 0, len(pp))
	for _, p := range pp {
		p = strings.TrimSpace(p)
		if p != "" {
			p2 = append(p2, p)
		}
	}

	return p2
}

func RandomTime(args string) interface{} {
	t := randx.Time()
	if args == "" {
		return t.Format(time.RFC3339Nano)
	}

	pp := ParseParams(args)
	if v, found := filter(pp, "now"); found {
		t = time.Now()
		pp = v
	}

	layout := timex.ConvertFormat(pp[0])
	if len(pp) == 1 {
		return t.Format(layout)
	}

	if len(pp) == 3 {
		from, err := time.ParseInLocation(layout, pp[1], time.Local)
		if err != nil {
			log.Printf("failed to parse %s by layout %s, error:%v", pp[1], pp[0], err)
			return t.Format(time.RFC3339Nano)
		}
		to, err := time.ParseInLocation(layout, pp[2], time.Local)
		if err != nil {
			log.Printf("failed to parse %s by layout %s, error:%v", pp[2], pp[0], err)
			return t.Format(time.RFC3339Nano)
		}

		fromUnix := from.Unix()
		toUnix := to.Unix()
		r := randx.Int64Between(fromUnix, toUnix)
		return time.Unix(r, 0).Format(layout)
	}

	return t.Format(time.RFC3339Nano)
}

func filter(pp []string, s string) (filtered []string, found bool) {
	filtered = make([]string, 0, len(pp))
	for _, p := range pp {
		if p == s {
			found = true
		} else {
			filtered = append(filtered, p)
		}
	}
	return
}

func RandomInt(args string) interface{} {
	if args == "" {
		return randx.Int64()
	}

	if i, err := strconv.ParseInt(args, 10, 64); err == nil {
		return randx.Int64Between(0, i)
	}

	if from, to, _, err := parseRandSize(args); err == nil {
		return randx.Int64Between(from, to)
	}

	var err error
	vv := int64(0)
	count := 0
	for _, el := range strings.Split(args, ",") {
		v := strings.TrimSpace(el)
		if v == "" {
			continue
		} else if !randx.Bool() {
			continue
		}

		if vv, err = strconv.ParseInt(v, 10, 64); err == nil {
			return vv
		}
		count++
	}

	if count > 0 {
		return vv
	}

	return randx.Int64()
}

func Random(args string) interface{} {
	if args == "" {
		return randx.String(10)
	}
	if i, err := strconv.Atoi(args); err == nil {
		return randx.String(i)
	}

	if _, _, times, err := parseRandSize(args); err == nil {
		return randx.String(int(times))
	}

	lastEl := ""
	for _, el := range strings.Split(args, ",") {
		if lastEl = strings.TrimSpace(el); lastEl == "" {
			continue
		}

		if randx.Bool() {
			return el
		}
	}

	if lastEl != "" {
		return lastEl
	}

	return randx.String(10)
}

func Regex(args string) interface{} {
	g, err := reggen.Generate(args, 100)
	if err != nil {
		log.Printf("bad regex: %s, err: %v", args, err)
	}
	return g
}

// ObjectID is the BSON ObjectID type.
type ObjectID [12]byte

var objectIDCounter = readRandomUint32()
var processUnique = processUniqueBytes()

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
	_, err := io.ReadFull(rander, b[:])
	if err != nil {
		panic(fmt.Errorf("cannot initialize objectid package with crypto.rand.Reader: %v", err))
	}

	return b
}

func readRandomUint32() uint32 {
	var b [4]byte
	_, err := io.ReadFull(rander, b[:])
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

// NewUUID creates a new random UUID or panics.
func NewUUID() UUID {
	return MustNewUUID(NewRandomUUID())
}

// MustNewUUID returns uuid if err is nil and panics otherwise.
func MustNewUUID(uuid UUID, err error) UUID {
	if err != nil {
		panic(err)
	}
	return uuid
}

// String returns the string form of uuid, xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
// , or "" if uuid is invalid.
func (uuid UUID) String() string {
	var buf [36]byte
	encodeHex(buf[:], uuid)
	return string(buf[:])
}

func encodeHex(dst []byte, uuid UUID) {
	hex.Encode(dst, uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}

var rander = rand.Reader // random function
var Nil UUID             // empty UUID, all zeros

// A UUID is a 128 bit (16 byte) Universal Unique IDentifier as defined in RFC 4122.
type UUID [16]byte

// NewRandomUUID returns a Random (Version 4) UUID.
//
// The strength of the UUIDs is based on the strength of the crypto/rand
// package.
//
// A note about uniqueness derived from the UUID Wikipedia entry:
//
//  Randomly generated UUIDs have 122 random bits.  One's annual risk of being
//  hit by a meteorite is estimated to be one chance in 17 billion, that
//  means the probability is about 0.00000000006 (6 × 10−11),
//  equivalent to the odds of creating a few tens of trillions of UUIDs in a
//  year and having one duplicate.
func NewRandomUUID() (UUID, error) {
	return NewRandomUUIDFromReader(rander)
}

// NewRandomUUIDFromReader returns a UUID based on bytes read from a given io.Reader.
func NewRandomUUIDFromReader(r io.Reader) (UUID, error) {
	var uuid UUID
	_, err := io.ReadFull(r, uuid[:])
	if err != nil {
		return Nil, err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	return uuid, nil
}
