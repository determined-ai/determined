package main

import (
	"io"
	"fmt"
	"os"
	"strings"
	"strconv"
	"go/token"
	"go/parser"
	"go/ast"
	"io/fs"
	"bytes"
	"time"

	"github.com/pkg/errors"
)

type Streamable struct {
	Name string
	Fields []Field
}

type Field struct {
	Name string
	Type string
	JsonTag string
}

// RootVisitor is the Visitor for the top-level go document.
type RootVisitor struct {
	src []byte
	out *[]Streamable
}

func (x RootVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	return DeclFinder{x.src, x.out}
}

// DeclFinder discards any top-level definitions which can't be a type declaration.
type DeclFinder struct {
	src []byte
	out *[]Streamable
}

func (x DeclFinder) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	_, ok := node.(*ast.GenDecl)
	if !ok {
		return nil
	}
	return &StreamableFinder{src: x.src, out: x.out}
}

// StreamableFinder seeks `type Thing struct` definitions with `determined:streamable` comments,
// builds an associated Streamable object, and adds it to the out slice.
type StreamableFinder struct {
	src []byte
	out *[]Streamable
	expectStreamable bool
}

func (x *StreamableFinder) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}
	if !x.expectStreamable {
		// is this a comment group containing the string "determined:streamable"?
		cmntgrp, ok := node.(*ast.CommentGroup)
		if ok {
			if strings.Contains(cmntgrp.Text(), "determined:streamable") {
				// it is! The next node hsould be our StructType.
				x.expectStreamable = true
			}
		}
		return nil
	}

	// expectstreamable is only valid once.
	x.expectStreamable = false

	// This should be a TypeSpec with .Type that is a StructType.
	typ, ok := node.(*ast.TypeSpec)
	if !ok {
		fmt.Fprintf(os.Stderr, "found special 'determined:streamable' comment on non-struct\n")
		os.Exit(1)
	}
	strct, ok := typ.Type.(*ast.StructType)
	if !ok {
		fmt.Fprintf(os.Stderr, "found special 'determined:streamable' comment on non-struct\n")
		os.Exit(1)
	}

	// Build our Streamable from this struct.
	result := Streamable{Name: typ.Name.String()}
	for _, field := range strct.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		if field.Tag == nil {
			continue
		}
		// The field tag comes as a literal; so unquote it to get the string
		tags, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse tag: %v\n", field.Tag.Value)
			os.Exit(7)
		}
		// Use strings.Fields to split tags by non-empty space-separated individual tags.
		for _, tag := range strings.Fields(tags) {
			// Let each individual tag be KEY:VALUE, where VALUE can be anything.
			splits := strings.SplitN(tag, ":", 2)
			if len(splits) != 2 {
				fmt.Fprintf(os.Stderr, "failed to parse tag: %v\n", field.Tag.Value)
				os.Exit(7)
			}
			// Now Unquote each VALUE as if it were another string literal.
			k := splits[0]
			v, err := strconv.Unquote(splits[1])
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to parse tag: %v\n", field.Tag.Value)
				os.Exit(7)
			}
			// Detect the json= tag to figure out the name of this field.
			if k != "json" {
				continue
			}
			// Pick out the first comma-separated value from tag values like "since,omit_empty".
			v = strings.SplitN(v, ",", 2)[0]
			// Get the string representing this type.  We use the string because the ast
			// representation of the type is a PITA to work with.
			typestr := string(x.src[field.Type.Pos()-1:field.Type.End()-1])
			result.Fields = append(result.Fields, Field{field.Names[0].String(), typestr, v})
		}
	}

	// extend output
	*x.out = append(*x.out, result)

	return nil
}

func parseFiles(files []string) ([]Streamable, error) {
	var results []Streamable

	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return nil, errors.Wrapf(err, "reading file: %v\n", src)
		}
		fs := token.NewFileSet()
		opts := parser.ParseComments | parser.SkipObjectResolution
		file, err := parser.ParseFile(fs, f, src, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "in file: %v\n", f)
		}

		ast.Walk(RootVisitor{src, &results}, file)
	}

	return results, nil
}

// Builder wraps strings.Builder but doesn't return a nil error like strings.Builder.
type Builder struct {
	builder strings.Builder
}

func (b *Builder) Writef(fstr string, args... interface{}) {
	if len(args) == 0 {
		_, _ = b.builder.WriteString(fstr)
		return
	}
	_, _ = b.builder.WriteString(fmt.Sprintf(fstr, args...))
}

func (b *Builder) String() string {
	return b.builder.String()
}

func genPython(streamables []Streamable) ([]byte, error) {
	b := Builder{}
	typeAnno := func(f Field) (string, error) {
		x := map[string]string {
			"JsonB": "typing.Any",
			"string": "str",
			"int": "int",
			"int64": "int",
			"[]int": "typing.List[int]",
			"time.Time": "float",
			"*time.Time": "typing.Optional[float]",
			"model.TaskID": "str",
			"model.RequestID": "int",
			"*model.RequestID": "typing.Optional[int]",
			"model.State": "str",
		}
		out, ok := x[f.Type]
		if !ok {
			return "", fmt.Errorf("no type annotation matches %q", f.Type)
		}
		return out, nil
	}
	b.Writef("# Code generated by stream-gen. DO NOT EDIT.\n")
	b.Writef("\n")
	b.Writef("\"\"\"Wire formats for the determined streaming updates subsystem\"\"\"\n")
	b.Writef("\n")
	b.Writef("import typing\n")
	b.Writef("\n")
	b.Writef("\n")
	b.Writef("class MsgBase:\n")
	b.Writef("    @classmethod\n")
	b.Writef("    def from_json(cls, obj: typing.Any):\n")
	b.Writef("        return cls(**obj)\n")
	b.Writef("\n")
	b.Writef("    def __repr__(self) -> str:\n")
	b.Writef("        body = \", \".join(f\"{k}={v}\" for k, v in vars(self).items())\n")
	b.Writef("        return f\"{type(self).__name__}({body})\"\n")

	for _, s := range streamables {
		b.Writef("\n\n")
		b.Writef("class %v(MsgBase):\n", s.Name)
		b.Writef("    def __init__(\n")
		b.Writef("        self,\n")
		for _, f := range s.Fields {
			anno, err := typeAnno(f)
			if err != nil {
				return nil, errors.Wrapf(err, "struct %v, field %v", s.Name, f.Name)
			}
			b.Writef("        %v: %q,\n", f.JsonTag, anno)
		}
		b.Writef("    ) -> None:\n")
		for _, f := range s.Fields {
			b.Writef("        self.%v = %v\n", f.JsonTag, f.JsonTag)
		}
	}
	return []byte(b.String()), nil
}

func printHelp(output io.Writer) {
	fmt.Fprintf(
		output,
		`stream-gen generates bindings for determined streaming updates.

usage: stream-gen IN.GO... --python [--output OUTPUT] [--stamp STAMP]

All structs in the input files IN.GO... which contain special 'determined:streamable' comments will
be included in the generated output.

Presently the only output language is --python.

Output will be written to stdout, or a location specified by --output.  The OUTPUT will only be
overwritten if it would be modified.

If --stamp is provided, the STAMP file will be touched after a successful run, which is useful for
integration with build systems.
`)
}

func main() {
	// Parse commandline options manually because built-in flag library is junk.
	if len(os.Args) == 1 {
		// no args provided
		printHelp(os.Stdout)
		os.Exit(0)
	}
	output := ""
	lang := ""
	stamp := ""
	gofiles := []string{}
	for i := 1 ; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "-h" || arg == "--help" {
			printHelp(os.Stdout)
			os.Exit(0)
		}
		if arg == "--python" {
			lang = "python"
			continue
		}
		if arg == "-o" || arg == "--output" {
			if i + 1 >= len(os.Args) {
				fmt.Fprintf(os.Stderr, "Missing --output parameter.\nTry --help.\n")
				os.Exit(2)
			}
			i++
			output = os.Args[i]
			continue
		}
		if arg == "-s" || arg == "--stamp" {
			if i + 1 >= len(os.Args) {
				fmt.Fprintf(os.Stderr, "Missing --stamp parameter.\nTry --help.\n")
				os.Exit(2)
			}
			i++
			stamp = os.Args[i]
			continue
		}
		if strings.HasPrefix(arg, "-") {
			fmt.Fprintf(os.Stderr, "Unrecognized option: %v.\nTry --help.\n", arg)
			os.Exit(2)
		}
		gofiles = append(gofiles, arg)
	}
	if len(gofiles) == 0 {
		fmt.Fprintf(os.Stderr, "No input files.\nTry --help.\n")
		os.Exit(2)
	}
	if lang == "" {
		fmt.Fprintf(os.Stderr, "No language specifier.\nTry --help.\n")
		os.Exit(2)
	}

	// read input files
	results, err := parseFiles(gofiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// generate the language bindings
	content, err := genPython(results)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// write to output
	if output == "" {
		// write to stdout
		_, err := os.Stderr.Write(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed writing to stdout: %v\n", err.Error())
			os.Exit(1)
		}
	} else {
		old, err := os.ReadFile(output)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "failed reading old content of %v: %v\n", output, err.Error())
			os.Exit(1)
		}
		if bytes.Equal(old, content) {
			// old output is already up-to-date
			fmt.Fprintf(os.Stderr, "output is up-to-date\n")
		} else {
			// write a new output
			err := os.WriteFile(output, content, 0666)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed writing to %v: %v\n", output, err.Error())
				os.Exit(1)
			}
		}
	}

	// touch stamp file
	if stamp != "" {
		err := os.Chtimes(stamp, time.Time{}, time.Now())
		if errors.Is(err, fs.ErrNotExist) {
			// file doesn't exist, create it instead
			var f *os.File
			f, err = os.Create(stamp)
			if f != nil {
				f.Close()
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "error touching stampfile (%v): %v\n", stamp, err.Error())
			os.Exit(1)
		}
	}
}
