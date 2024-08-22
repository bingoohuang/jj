package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bingoohuang/jj"
	_ "github.com/bingoohuang/jj/randpoem"
	"github.com/bingoohuang/ngg/ss"
	"github.com/bingoohuang/ngg/ver"
	"github.com/expr-lang/expr"
	"github.com/mattn/go-isatty"
)

var (
	version = "1.0.1"
	tag     = "jj - JSON Stream Editor " + version
	usage   = `usage: jj [-v value] [-curnOD] [-i infile] [-o outfile] keypath
eg.: jj keypath                      read value from stdin
     jj -i infile keypath            read value from infile
     jj -v value keypath             edit value
     jj -v value -o outfile keypath  edit value and write to outfile
options:
     -v value   Edit JSON key path value
     -V         Print version and exit
     -c         Print cheatsheet
     -C         Print items counting in colored output
     -u         Make json ugly, keypath is optional
     -R         Create a random json, use env N for #element, e.g. N=10 jj -R
     -r         Use raw values, otherwise types are auto-detected
     -n         Do not modifyOutput color or extra formatting
     -O         Performance boost for value updates
     -D         Delete the value at the specified key path
     -l         Output array values on multiple lines
     -I         Print each child of json array
     -i infile  Use input file instead of stdin
     -g         Generate random JSON by input, use env N for more times, e.g. N=3 jj -gu name=@name
     -e         Eval keypath value as an expression
     -p         Parse inner JSON string as a JSON
     -o outfile Use output file instead of stdout
     -f regex   List the key and values which regex matches its key
     -J         Pure javascript object which has name quoting leniently
     -JJ        Pure javascript object which has all string quoting leniently
     -k keypath JSON key path (like "name.last")
     -K keypath JSON key path as raw whole key
      keypath   Last argument for JSON key path`
)

type args struct {
	infile, outfile, value *string

	keypath, findRegex string

	raw, del, opt, keypathok, random      bool
	ugly, notty, lines, rawKey, gen, expr bool
	iterateArray, parseInnerJSONString    bool
	countingItems                         bool
	quoteNameLeniently                    int

	jsonMap map[string]any
}

func parseArgs() args {
	fail := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "%s\n", tag)
		if format != "" {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
		fmt.Fprintf(os.Stderr, "%s\n", usage)
		os.Exit(1)
	}
	help := func() {
		buf := &bytes.Buffer{}
		fmt.Fprintf(buf, "%s\n", tag)
		fmt.Fprintf(buf, "%s\n", usage)
		os.Stdout.Write(buf.Bytes())
		os.Exit(0)
	}

	var a args
	a.jsonMap = make(map[string]any)

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		default:
			if len(os.Args[i]) > 1 && os.Args[i][0] == '-' {
				for j := 1; j < len(os.Args[i]); j++ {
					switch os.Args[i][j] {
					case 'c':
						fmt.Println(cheatText)
						os.Exit(0)
					case 'V':
						fmt.Printf("%s version: %s\n", os.Args[0], ver.Version())
						os.Exit(0)
					case 'u':
						a.ugly = true
					case 'r':
						a.raw = true
					case 'R':
						a.random = true
					case 'C':
						a.countingItems = true
					case 'O':
						a.opt = true
					case 'D':
						a.del = true
					case 'n':
						a.notty = true
					case 'p':
						a.parseInnerJSONString = true
					case 'I':
						a.iterateArray = true
					case 'l':
						a.lines = true
					case 'g':
						a.gen = true
					case 'e':
						a.expr = true
					case 'J':
						a.quoteNameLeniently++
					default:
						goto P1
					}
				}
				continue
			}
		P1:
			if p1 := strings.Index(os.Args[i], ":="); p1 > 0 {
				// Raw JSON fields
				a.jsonMap[os.Args[i][:p1]] = json.RawMessage(os.Args[i][p1+2:])
			} else if p2 := strings.Index(os.Args[i], "="); p2 > 0 {
				// Json fields
				a.jsonMap[os.Args[i][:p2]] = os.Args[i][p2+1:]
			} else if !a.keypathok {
				a.keypathok = true
				a.keypath = os.Args[i]
			} else {
				fail("unknown option argument: \"%s\"", a.keypath)
			}
		case "-v", "-i", "-o", "-k", "-K", "-f":
			arg := os.Args[i]
			i++
			if i >= len(os.Args) {
				fail("argument missing after: \"%s\"", arg)
			}
			switch arg {
			case "-v":
				a.value = &os.Args[i]
			case "-i":
				a.infile = &os.Args[i]
			case "-o":
				a.outfile = &os.Args[i]
			case "-k", "-K":
				a.keypathok = true
				a.keypath = os.Args[i]
				a.rawKey = arg == "-K"
			case "-f":
				a.findRegex = os.Args[i]
			}
		case "--force-notty":
			a.notty = true
		case "--version":
			fmt.Fprintf(os.Stdout, "%s\n", tag)
			os.Exit(0)
		case "-h", "--help", "-?":
			help()
		}
	}

	return a
}

var (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
	White  = "\033[97m"
)

func init() {
	if runtime.GOOS == "windows" {
		Reset = ""
		Red = ""
		Green = ""
		Yellow = ""
		Blue = ""
		Purple = ""
		Cyan = ""
		Gray = ""
		White = ""
	}
}

//go:embed cheat.txt
var cheatText string

func init() {
	cheatText = strings.ReplaceAll(cheatText, "=>", Green+"=>"+Reset)
	cheatText = strings.ReplaceAll(cheatText, "$ ", Purple+"$ "+Reset)
	cheatText = strings.ReplaceAll(cheatText, " jj", Cyan+" jj"+Reset)
	for i := 30; i > 0; i-- {
		num := fmt.Sprintf("%d. ", i)
		cheatText = strings.Replace(cheatText, num, Red+num+Reset, 1)
	}
}

func main() {
	a := parseArgs()
	f := a.createOutFile()

	outChan := make(chan Out)
	go a.createOut(outChan)

	for out := range outChan {
		outData := a.modifyOutput(f, out)
		_, _ = f.Write(outData)
	}
	_ = f.Close()
}

type Out struct {
	Data    []byte
	IsArray bool
	Type    jj.Type
}

func (a args) createOut(outChan chan Out) {
	if a.random {
		a.randomJSON(outChan)
		return
	}

	var input []byte
	var err error
	if len(a.jsonMap) > 0 {
		input, err = json.Marshal(a.jsonMap)
	} else {
		input, err = createInput(a)
	}
	if err != nil {
		fail(err)
	}

	if a.gen {
		a.generate(outChan, input)
		return
	}

	if a.iterateArray {
		a.doIterateArray(outChan, input)
		return
	}

	if a.parseInnerJSONString {
		a.formatInnerJsonString(outChan, input)
		return
	}

	if a.findRegex != "" {
		a.findKeyValues(input)
		close(outChan)
		return
	}

	opts := jj.SetOptions{PathOption: jj.PathOption{RawPath: a.rawKey}}

	if a.del {
		var out Out
		if out.Data, err = jj.DeleteBytes(input, a.keypath, opts); err != nil {
			fail(err)
		}
		outChan <- out
		close(outChan)
		return
	}

	if a.value != nil {
		raw := a.raw
		val := *a.value
		if !raw {
			switch val {
			default:
				if len(val) > 0 {
					if (val[0] >= '0' && val[0] <= '9') || val[0] == '-' {
						if _, err := strconv.ParseFloat(val, 64); err == nil {
							raw = true
						}
					}
				}
			case "true", "false", "null":
				raw = true
			}
		}

		if a.opt {
			opts.Optimistic = true
			opts.ReplaceInPlace = true
		}

		var out Out
		if raw { // set as raw block
			out.Data, err = jj.SetRawBytes(input, a.keypath, []byte(val), opts)
		} else { // set as a string
			out.Data, err = jj.SetBytes(input, a.keypath, val, opts)
		}
		if err != nil {
			fail(err)
		}

		outChan <- out
		close(outChan)
		return
	}

	if !a.keypathok {
		var out Out

		out.Data = input
		outChan <- out
		close(outChan)
		return
	}

	if a.expr {
		env := map[string]any{}
		if err := json.Unmarshal(input, &env); err != nil {
			panic(err)
		}
		program, err := expr.Compile(a.keypath, expr.Env(env))
		if err != nil {
			panic(err)
		}

		output, err := expr.Run(program, env)
		if err != nil {
			panic(err)
		}
		v, err := json.Marshal(output)
		if err != nil {
			fail(err)
		}
		var out Out
		a.assignOut(&out, jj.ParseBytes(v))
		outChan <- out
		close(outChan)
		return
	}

	var out Out
	for {
		typ, outi, _ := jj.ValidPayload(input, 0)
		if outi == 0 {
			break
		}
		if typ == jj.JSON {
			res := jj.GetBytes(input, a.keypath, jj.WithRawPath(a.rawKey))
			a.assignOut(&out, res)
			outChan <- out
		}
		input = input[outi:]
	}

	close(outChan)
	return
}

func (a args) randomJSON(outChan chan Out) {
	rand.Seed(time.Now().UnixNano())
	randOptions := jj.DefaultRandOptions
	randOptions.Pretty = false
	times := 1
	if j := os.Getenv("N"); j != "" {
		if strings.Contains(j, ",") {
			k := strings.IndexByte(j, ',')
			if times, _ = ss.Parse[int](j[:k]); times < 1 {
				times = 1
			}
			j = j[k+1:]
		}
		if k, _ := ss.Parse[int](j); k > 0 {
			randOptions.Depth = k
		}
	}

	for i := 0; i < times; i++ {
		outChan <- Out{Data: jj.Rand(randOptions)}
	}

	close(outChan)
}

func (a args) formatInnerJsonString(outChan chan Out, input []byte) {
	out := jj.ParseBytes(jj.FreeInnerJSON(input))
	outChan <- Out{
		Data:    []byte(out.String()),
		IsArray: out.IsArray(),
		Type:    out.Type,
	}
	close(outChan)
}

func (a args) doIterateArray(outChan chan Out, input []byte) {
	started := false
	openCount := 0
	isArray := false
	elemStart := 0
	jj.StreamParse(input, func(start, end, info int) int {
		if !started {
			started = true
			isArray = jj.IsToken(info, jj.TokArray)
			return 1
		}

		if !isArray {
			return -1
		}

		if jj.IsToken(info, jj.TokOpen) {
			openCount++
			if openCount == 1 {
				elemStart = start
			}
		} else if jj.IsToken(info, jj.TokClose) {
			if openCount == 1 {
				out := Out{Data: input[elemStart:end], IsArray: jj.IsToken(info, jj.TokArray), Type: jj.JSON}
				outChan <- out
			}
			openCount--
		}

		if openCount == 0 && jj.IsToken(info, jj.TokString, jj.TokNumber, jj.TokTrue, jj.TokFalse, jj.TokNull) {
			var typ jj.Type
			switch {
			case jj.IsToken(info, jj.TokString):
				typ = jj.String
			case jj.IsToken(info, jj.TokNumber):
				typ = jj.Number
			case jj.IsToken(info, jj.TokTrue):
				typ = jj.True
			case jj.IsToken(info, jj.TokFalse):
				typ = jj.False
			case jj.IsToken(info, jj.TokNull):
				typ = jj.Null
			}

			out := Out{Data: input[start:end], IsArray: false, Type: typ}
			outChan <- out
		}

		return -1
	})

	close(outChan)
}

func (a args) findKeyValues(input []byte) {
	re := regexp.MustCompile(a.findRegex)
	foundKey := ""
	found := false
	openCount := 0
	openStart := 0
	jj.StreamParse(input, func(start, end, info int) int {
		v := string(input[start:end])
		if found {
			if jj.IsToken(info, jj.TokOpen) {
				openCount++
				if openCount == 1 {
					openStart = start
				}
			} else if jj.IsToken(info, jj.TokClose) {
				openCount--
				if openCount == 0 {
					fmt.Printf("%s: %s\n", foundKey, input[openStart:end])
					found = false
				}
			} else if openCount == 0 && jj.IsToken(info, jj.TokString, jj.TokNumber, jj.TokTrue, jj.TokFalse, jj.TokNull) {
				fmt.Printf("%s: %s\n", foundKey, v)
				found = false
			}
		} else if jj.IsToken(info, jj.TokKey) {
			k := v[1 : len(v)-1] // 去除两端双引号
			if found = re.MatchString(k); found {
				foundKey = v
			}
		}

		return 1
	})
}

func (a args) assignOut(out *Out, res jj.Result) {
	if a.raw {
		out.Data = []byte(res.Raw)
	} else {
		out.Type = res.Type
		out.IsArray = res.IsArray()
		out.Data = []byte(res.String())
	}
}

func GetEnvInt(name string, defaultValue int) int {
	s := os.Getenv(name)
	if s == "" {
		return defaultValue
	}

	if ev, err := ss.Parse[int](s); err == nil {
		return ev
	}

	return defaultValue
}

func (a args) generate(outChan chan Out, input []byte) {
	gen := jj.NewGenContext(jj.NewCachingSubstituter())
	defer close(outChan)

	for j := 0; j < GetEnvInt(`N`, 1); j++ {
		s := string(input)
		for {
			genResult, i, err := gen.Process(s)
			if err != nil {
				log.Printf("error: %v", err)
				return
			}
			if i <= 0 {
				break
			}

			outChan <- Out{Data: []byte(genResult.Out)}
			s = s[i:]
		}
	}
}

func createInput(a args) ([]byte, error) {
	if a.infile == nil {
		return io.ReadAll(os.Stdin)
	}

	if stat, err := os.Stat(*a.infile); err == nil && !stat.IsDir() {
		return os.ReadFile(*a.infile)
	}

	if strings.HasPrefix(*a.infile, "@") {
		return os.ReadFile((*a.infile)[1:])
	}

	return []byte(*a.infile), nil
}

func (a args) createOutFile() *os.File {
	if a.outfile == nil {
		return os.Stdout
	}

	f, err := os.Create(*a.outfile)
	if err != nil {
		fail(err)
	}
	return f
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "error: %v\n", err.Error())
	os.Exit(1)
}

func (a args) modifyOutput(f *os.File, out Out) []byte {
	if a.lines && out.IsArray {
		var b2 []byte
		jj.ParseBytes(out.Data).ForEach(func(_, v jj.Result) bool {
			b2 = append(b2, jj.Ugly([]byte(v.Raw))...)
			b2 = append(b2, '\n')
			return true
		})
		out.Data = b2
	} else if a.raw || out.Type != jj.String {
		if a.ugly {
			out.Data = jj.Ugly(out.Data)
		} else {
			out.Data = jj.Pretty(out.Data)
		}
	}

	if a.quoteNameLeniently > 0 {
		var opt []jj.QuoteOptionFunc
		if a.quoteNameLeniently > 1 {
			opt = append(opt, jj.WithLenientValue())
		}

		out.Data = jj.FormatQuoteNameLeniently(out.Data, opt...)
	}

	for len(out.Data) > 0 && out.Data[len(out.Data)-1] == '\n' {
		out.Data = out.Data[:len(out.Data)-1]
	}

	if !a.notty && isatty.IsTerminal(f.Fd()) {
		if a.raw || out.Type != jj.String {
			out.Data = jj.Color(out.Data, jj.TerminalStyle, &jj.ColorOption{CountEntries: a.countingItems})
		} else {
			out.Data = append([]byte(jj.TerminalStyle.String[0]), out.Data...)
			out.Data = append(out.Data, jj.TerminalStyle.String[1]...)
		}
	}

	return append(out.Data, '\n')
}
