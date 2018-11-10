package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"log"
	"strings"
	"text/template"

	"github.com/Joker/jade"
	"golang.org/x/tools/imports"
)

const (
	file_bgn = `// Code generated by "jade.go"; DO NOT EDIT.

package {{.Package}}

import (
{{- range .Import}}
	{{.}}
{{- end}}
)

{{- range .Def}}
	{{.}}
{{- end}}

{{.Func}} {
	{{.Before}}
`
	file_end = `
	{{.After}}
}
`
)

type layout struct {
	Package string
	Import  []string
	Def     []string
	Bbuf    string
	Func    string
	Before  string
	After   string
}

func (data *layout) writeBefore(wr io.Writer) {
	t := template.Must(template.New("file_bgn").Parse(file_bgn))
	err := t.Execute(wr, data)
	if err != nil {
		log.Fatalln("executing template: ", err)
	}
}
func (data *layout) writeAfter(wr *bytes.Buffer) {
	t := template.Must(template.New("file_end").Parse(file_end))
	err := t.Execute(wr, struct{ After string }{data.After})
	if err != nil {
		log.Fatalln("executing template: ", err)
	}
}

func newLayout(constName string) layout {
	var tpl layout
	tpl.Package = pkg_name
	tpl.Import = []string{`"bytes"`, `"fmt"`, `"html"`, `"strconv"`, `pool "github.com/valyala/bytebufferpool"`, `"github.com/Joker/hpp"`}

	if !inline {
		tpl.Def = []string{"const ()"}
	}
	if stdbuf {
		tpl.Bbuf = "*bytes.Buffer"
	} else {
		tpl.Bbuf = "*pool.ByteBuffer"
	}
	if format {
		tpl.After = constName + `__buffer := hpp.Print(bytes.NewReader(buffer.Bytes()))
		buffer.Reset()
		buffer.Write(` + constName + `__buffer)
		`
	}

	if jade.Go.Name != "" {
		tpl.Func = "func " + jade.Go.Name
		jade.Go.Name = ""
	} else {
		tpl.Func = `func tpl_` + constName
	}
	if jade.Go.Args != "" {
		args := strings.Split(jade.Go.Args, ",")
		buffer := true
		for k, v := range args {
			args[k] = strings.Trim(v, " \t\n")
			if strings.HasPrefix(args[k], "buffer ") {
				args[k] = "buffer " + tpl.Bbuf
				buffer = false
			}
		}
		if buffer {
			args = append(args, "buffer "+tpl.Bbuf)
		}
		tpl.Func += "(" + strings.Join(args, ",") + ")"
		jade.Go.Args = ""
	} else {
		tpl.Func += `(buffer ` + tpl.Bbuf + `) `
	}

	if jade.Go.Import != "" {
		imp := strings.Split(jade.Go.Import, "\n")
		for k, v := range imp {
			str := strings.Trim(v, " \t")
			if v[len(v)-1:] != `"` { // lastChar != `"`
				imp[k] = `"` + str + `"`
			} else {
				imp[k] = str
			}
		}
		tpl.Import = append(tpl.Import, imp...)
		jade.Go.Import = ""
	}
	return tpl
}

//

type goAST struct {
	node *ast.File
	fset *token.FileSet
}

func parseGoSrc(fileName string, GoSrc interface{}) (out goAST, err error) {
	out.fset = token.NewFileSet()
	out.node, err = parser.ParseFile(out.fset, fileName, GoSrc, parser.ParseComments)
	return
}

func (a *goAST) bytes(bb *bytes.Buffer) []byte {
	printer.Fprint(bb, a.fset, a.node)
	return bb.Bytes()
}

func goImports(absPath string, src []byte) []byte {
	fmtOut, err := imports.Process(absPath, src, &imports.Options{TabWidth: 4, TabIndent: true, Comments: true, Fragment: true})
	if err != nil {
		log.Fatalln("goImports(): ", err)
	}

	return fmtOut
}
