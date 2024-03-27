package main

import (
	"bufio"
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
	l = parseLoader(f, encoding)

	return l, nil
}

func parseLoader(r io.ReadCloser, encoding string) loader {
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

	if len(os.Args) < 3 {
		log.Fatalln("provide file name")
	}

	var (
		cmd      = os.Args[1]
		fileName = os.Args[2]
	)
	switch cmd {
	case "diff":
		f1, f2, ok := strings.Cut(fileName, ",")
		if !ok {
			return
		}
		b1 := sortValues(f1)
		b2 := sortValues(f2)
		fmt.Println("DIFF:")
		for _, v := range pretty.Diff(b1, b2) {
			fmt.Println(v)
		}
		return
	default:
		sortValues(fileName)
	}
}

func sortValues(fileName string) map[string]string {
	values := load(fileName)

	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
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
	})
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
