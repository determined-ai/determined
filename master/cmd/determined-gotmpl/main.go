package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/ghodss/yaml"

	"github.com/Masterminds/sprig/v3"
	"github.com/huandu/xstrings"
)

func stdinData() map[string]interface{} {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return nil
	}
	stdin, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	var data = map[string]interface{}{}
	err = yaml.Unmarshal(stdin, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func toYaml(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RunTmpl renders template `files` given context `data`.
func RunTmpl(data map[string]interface{}, files []string) *bytes.Buffer {
	funcs := sprig.TxtFuncMap()
	funcs["cwd"] = os.Getwd
	funcs["args"] = func() []string { return os.Args }
	funcs["templateFiles"] = func() []string { return files }
	funcs["absPath"] = filepath.Abs
	funcs["upperFirst"] = xstrings.FirstRuneToUpper
	funcs["lowerFirst"] = xstrings.FirstRuneToLower
	funcs["reflectKind"] = func(val interface{}) string {
		if val == nil {
			return ""
		}
		return reflect.TypeOf(val).Kind().String()
	}
	funcs["toYaml"] = toYaml

	tmpl, err := template.
		New(filepath.Base(files[0])).
		Funcs(funcs).
		ParseFiles(files...)
	if err != nil {
		log.Fatal(err)
	}
	b := new(bytes.Buffer)
	err = tmpl.Execute(b, data)
	if err != nil {
		log.Fatal(err)
	}

	return b
}

func main() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(),
			`Usage: determined-gotmpl [-i data.yaml] [-o output.txt] [key=value] template [template ...]

Based on https://github.com/NateScarlet/gotmpl
Adds:
 - input yaml context support.
 - toYaml and reflectKind helpers.
`)
	}
	var output string
	var input string

	flag.StringVar(&output, "o", "", "output file path")
	flag.StringVar(&input, "i", "", "input yaml data file path")
	flag.Parse()

	data := map[string]interface{}{} // sprig dict functions require map[string]interface{}
	files := []string{}

	if output != "" {
		p, err := filepath.Abs(output)
		if err != nil {
			log.Fatal(err)
		}
		data["Name"] = strings.TrimSuffix(filepath.Base(p), filepath.Ext(filepath.Base(p)))
	}

	for k, v := range stdinData() {
		data[k] = v
	}

	if input != "" {
		d, err := ioutil.ReadFile(filepath.Clean(input))
		if err != nil {
			log.Fatal(err)
		}
		err = yaml.Unmarshal(d, &data)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, i := range flag.Args() {
		if strings.Contains(i, "=") {
			kv := strings.SplitN(i, "=", 2)
			if kv[0] == "" {
				files = append(files, kv[1])
				continue
			}
			data[kv[0]] = kv[1]
		} else {
			files = append(files, i)
		}
	}

	if len(files) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	b := RunTmpl(data, files)

	if output != "" {
		var err = ioutil.WriteFile(output, b.Bytes(), 0644) //nolint: gosec
		if err != nil {
			log.Fatal(err)
		}
	} else {
		fmt.Print(b.String())
	}
}
