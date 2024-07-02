package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/kr/pretty"
	"gopkg.in/yaml.v3"
)

type loader func(a any) error

func (l loader) Load(a any) error {
	return l(a)
}

type Loader interface {
	Load(a any) error
}

func parseFile(name, encoding string) (l Loader, err error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	s := bufio.NewScanner(bufio.NewReader(f))
	s.Split(bufio.ScanLines)
	i := 1
	for s.Scan() {
		t := strings.TrimLeft(s.Text(), " ")
		if strings.Replace(t, "\n", "", 1) == "" {
			continue
		}
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") {
			continue
		}
		t = strings.ReplaceAll(t, " :", ":")
		t = strings.ReplaceAll(t, " =", ":")
		t = strings.ReplaceAll(t, "  ", " ")
		t = strings.ReplaceAll(t, "=\"", ": \"")
		t = strings.ReplaceAll(t, "= ", ": ")
		t = strings.ReplaceAll(t, "=", ": ")

		b, _, ok := strings.Cut(t, "#")
		if !ok {
			b, _, ok = strings.Cut(t, "// ")
		}
		if ok {
			t = b
		}

		buf.WriteString(t)
		buf.WriteString("\n")
		//fmt.Println("i", i, t)
		i++
	}

	return parseLoader(buf, encoding), f.Close()
}

func parseLoader(r io.Reader, encoding string) loader {
	switch encoding {
	case "json":
		return json.NewDecoder(r).Decode
	}

	return yaml.NewDecoder(r).Decode
}

func load(fileName string) map[string]string {
	_, encoding, ok := strings.Cut(fileName, ".")
	if !ok {
		log.Fatalln("provide full file name")
	}
	l, err := parseFile(fileName, encoding)
	if err != nil {
		log.Fatalln("parseLoader", err)
	}
	values := map[string]string{}

	if err = l.Load(&values); err != nil {
		log.Fatalf("Load %s %s %T\n", err, encoding, l)
	}

	return values
}

func main() {

	if len(os.Args) < 2 {
		return
	}
	cmd := os.Args[1]
	fileName := ""
	if len(os.Args) >= 3 {
		fileName = os.Args[2]
	} else if cmd != "help" {
		fileName = os.Args[2]
	}
	fmt.Println(fileName)
	switch cmd {
	case "help", "h", "-h":
		fmt.Println("usage:\n ./envsort [sort|diff] [file1],[file2]")
	case "diff", "-d":
		f1, f2, ok := strings.Cut(fileName, ",")
		if !ok {
			return
		}
		b1 := sortValues(f1)
		b2 := sortValues(f2)
		diff := pretty.Diff(b1, b2)
		sort.Sort(SortedDiffs(diff[2:]))
		for _, v := range diff {
			fmt.Println(v)
		}
		return
	case "sort", "s", "-s":

		sortValues(fileName)
	}
}

type SortedEnvs []string

func (keys SortedEnvs) Len() int {
	return len(keys)
}

func (keys SortedEnvs) Less(i, j int) bool {
	vals1 := strings.Split(keys[i], "_")
	vals2 := strings.Split(keys[j], "_")

	if len(vals1) == 0 || len(vals2) == 0 {
		return keys[i] < keys[j]
	}
	for i := 0; i < min(len(vals2), len(vals1)); i++ {
		if vals1[i] != vals2[i] {
			return vals1[i] < vals2[i]
		}
	}

	return keys[i] < keys[j]
}

func (s SortedEnvs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type SortedDiffs []string

func (keys SortedDiffs) Len() int {
	return len(keys)
}

func (keys SortedDiffs) Less(i, j int) bool {
	idxi := strings.Index(keys[i], `"`)
	idxj := strings.Index(keys[j], `"`)
	vals1 := strings.Split(keys[i][idxi:], "_")
	vals2 := strings.Split(keys[j][idxj:], "_")

	if len(vals1) == 0 || len(vals2) == 0 {
		return keys[i] < keys[j]
	}
	for i := 0; i < min(len(vals2), len(vals1)); i++ {
		if vals1[i] != vals2[i] {
			return vals1[i] < vals2[i]
		}
	}

	return keys[i] < keys[j]
}

func (s SortedDiffs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

var _ sort.Interface = new(SortedEnvs)

func sortValues(fileName string) map[string]string {
	values := load(fileName)

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Sort(SortedEnvs(keys))

	file, err := os.Create("tmp_" + fileName)
	if err != nil {
		log.Fatalln("Create", err)
	}
	defer file.Close()
	f := bufio.NewWriter(file)

	for i, k := range keys {
		_, err = fmt.Fprintf(f, "%s: \"%s\"\n", k, values[k])
		if err != nil {
			log.Fatalln("Fprintf", err)
		}
		if i < len(keys)-1 {
			if ss := strings.Split(k, "_"); len(ss) > 0 {
				if s := strings.Split(keys[i+1], "_"); len(s) > 0 {
					if ss[0] != s[0] {
						_, _ = fmt.Fprintf(f, "\n")
					}
				}
			}
		}
	}

	return values
}
