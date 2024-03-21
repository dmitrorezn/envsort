package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type loader func(a any) error

func (l loader) Load(a any) error {
	return l(a)
}

type Loader interface {
	Load(a any) error
}

func parseLoader(name, encoding string) (l Loader, err error) {
	var b []byte
	b, err = os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	switch encoding {
	case "json":
		return loader(func(a any) error {
			return json.Unmarshal(b, a)
		}), nil
	}

	return loader(func(a any) error {
		return yaml.Unmarshal(b, a)
	}), nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("provide file name")
	}

	var (
		fileName = os.Args[1]
	)
	_, encoding, ok := strings.Cut(fileName, ".")
	if !ok {
		log.Fatalln("provide full file name")
	}

	l, err := parseLoader(fileName, encoding)
	if err != nil {
		log.Fatalln("parseLoader", err)
	}

	values := map[string]string{}
	if err = l.Load(&values); err != nil {
		log.Fatalf("Load %s %T\n", err, encoding, l)
	}
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

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatalln("Create", err)
	}
	defer f.Close()
	for i, k := range keys {
		_, err = fmt.Fprintf(f, "	%s: \"%s\"\n", k, values[k])
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
}
