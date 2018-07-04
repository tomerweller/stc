package main

import "fmt"
import "io"
import "io/ioutil"
import "os"
import "strings"
import "strconv"

//go:generate goyacc -o parse.go parse.y

func capitalize(s string) string {
	if len(s) > 0 && s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0] &^ 0x20) + s[1:]
	}
	return s
}

func uncapitalize(s string) string {
	if len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z' {
		return string(s[0] | 0x20) + s[1:]
	}
	return s
}

func underscore(s string) string {
	if len(s) > 0 && s[0] == '_' {
		return s
	}
	return "_" + s
}

func parseXDR(out *rpc_syms, file string) {
	src, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		out.Failed = true
		return
	}
	l := NewLexer(out, file, string(src))
	yyParse(l)
}


type emitter struct {
	syms *rpc_syms
	declarations []string
	emitted map[string]struct{}
}

func (e *emitter) append(out interface{}) {
	var s string
	switch t := out.(type) {
	case string:
		s = t
	case fmt.Stringer:
		s = t.String()
	default:
		panic("emitter append non-String")
	}
	e.declarations = append(e.declarations, s)
}

func (e *emitter) printf(str string, args ...interface{}) {
	e.append(fmt.Sprintf(str, args...))
}

func (e *emitter) chase_typedef(id string) string {
	if d, ok := e.syms.SymbolMap[id]; ok {
		if td, ok := d.(*rpc_typedef);
		ok && td.qual == SCALAR && td.inline_decl == nil {
			return e.chase_typedef(td.typ)
		}
	}
	return id
}

func (e *emitter) chase_bound(d *rpc_decl) string {
	b := d.bound
	for {
		if s, ok := e.syms.SymbolMap[b]; !ok {
			break
		} else if d2, ok := s.(*rpc_const); !ok {
			break
		} else {
			b = d2.val
		}
	}
	if b != "" {
		i32, err := strconv.ParseInt(b, 0, 32)
		if err != nil {
			return b
		} else if i32 != 0xffffffff {
			return fmt.Sprintf("%d", i32)
		}
	}
	return ""
}

func (e *emitter) decltype(parent rpc_sym, d *rpc_decl) string {
	out := &strings.Builder{}
	var bound string
	switch d.qual {
	case SCALAR:
		if d.typ == "string" {
			bound = e.chase_bound(d)
		}
	case PTR:
		fmt.Fprintf(out, "*");
	case ARRAY:
		fmt.Fprintf(out, "[%s]", d.bound)
	case VEC:
		if bound = e.chase_bound(d); bound == "" {
			fmt.Fprintf(out, "[]")
		}
	}
	if d.typ == "" {
		if _, isTypedef := parent.(*rpc_typedef); isTypedef {
			d.typ = underscore(d.id)
		} else {
			d.typ = underscore(*parent.symid()) + "_" + d.id
		}
		*d.inline_decl.symid() = d.typ
		e.emit(d.inline_decl)
	}
	if (bound == "") {
		fmt.Fprintf(out, "%s", d.typ)
		return out.String()
	}
	typ := underscore(e.chase_typedef(d.typ)) + "_" + bound
	if _, ok := e.emitted[typ]; !ok {
		d1 := *d
		d1.id = typ
		d1.bound = ""
		e.emit(&d1)
		e.printf("func (*%s) XdrBound() uint32 {\n" +
			"\treturn %s\n" +
			"}\n", typ, bound)
		e.emitted[typ] = struct{}{}
	}
	return typ
}

func (e *emitter) emit(sym rpc_sym) {
	sym.(Emittable).emit(e)
}


type Emittable interface {
	emit(e *emitter)
}

func (r *rpc_const) emit(e *emitter) {
	e.printf("const %s = %s\n", r.id, r.val)
}

func (r *rpc_decl) emit(e *emitter) {
	e.printf("type %s %s\n", r.id, e.decltype(r, r))
}

func (r *rpc_typedef) emit(e *emitter) {
	e.printf("type %s = %s\n", r.id, e.decltype(r, (*rpc_decl)(r)))
}

func (r *rpc_enum) emit(e *emitter) {
	out := &strings.Builder{}
	fmt.Fprintf(out, "type %s int32\nconst (\n", r.id);
	for _, tag := range r.tags {
		fmt.Fprintf(out, "\t%s = %s(%s)\n", tag.id, r.id, tag.val)
	}
	fmt.Fprintf(out, ")\n");
	fmt.Fprintf(out, "var _%s_names = map[int32]string{\n", r.id);
	for _, tag := range r.tags {
		fmt.Fprintf(out, "\tint32(%s): \"%s\",\n", tag.id, tag.id);
	}
	fmt.Fprintf(out, "}\n");
	fmt.Fprintf(out, "func (*%s) EnumNames() map[int32]string {\n" +
		"\treturn _%s_names\n}\n", r.id, r.id)
	fmt.Fprintf(out, "func (v *%s) EnumVal() *int32 {\n" +
		"\treturn (*int32)(v)\n" +
		"}\n", r.id)
	fmt.Fprintf(out, "func (v *%s) String() string {\n" +
		"\tif s, ok := _%s_names[int32(*v)]; ok {\n" +
		"\t\treturn s\n\t}\n" +
		"\treturn \"unknown_%s\"\n}\n",
		r.id, r.id, r.id)
	fmt.Fprintf(out, "func (v *%s) Value() interface{} {\n" +
		"\treturn *v\n" +
		"}\n", r.id)
	e.append(out)
}

func (r *rpc_struct) emit(e *emitter) {
	out := &strings.Builder{}
	fmt.Fprintf(out, "type %s struct {\n", r.id);
	for _, decl := range r.decls {
		fmt.Fprintf(out, "\t%s %s\n", decl.id, e.decltype(r, &decl))
	}
	fmt.Fprintf(out, "}\n")
	e.append(out)
}

func (r *rpc_union) emit(e *emitter) {
	out := &strings.Builder{}
	fmt.Fprintf(out, "type %s struct {\n", r.id);
	fmt.Fprintf(out, "\t%s %s\n", r.tagid, r.tagtype);
	fmt.Fprintf(out, "\t_u interface{}\n");
	fmt.Fprintf(out, "}\n");
	for _, u := range r.fields {
		if u.decl.id == "" || u.decl.typ == "void" {
			continue
		}
		ret := e.decltype(r, &u.decl)
		fmt.Fprintf(out, "func (u *%s) %s() *%s {\n", r.id, u.decl.id, ret)
		goodcase := fmt.Sprintf("\t\tif v, ok := u._u.(*%s); ok {\n" +
			"\t\t\treturn v\n" +
			"\t\t} else {\n" +
			"\t\t\tvar zero %s\n" +
			"\t\t\tu._u = &zero\n" +
			"\t\t\treturn &zero\n" +
			"\t\t}\n", ret, ret)
		badcase := fmt.Sprintf(
			"\t\tpanic(\"%s accessed when not selected\")\n", u.decl.id)
		fmt.Fprintf(out, "\tswitch u.%s {\n", r.tagid);
		if u.hasdefault && len(r.fields) > 1 {
			needcomma := false
			fmt.Fprintf(out, "\tcase ");
			for _, u1 := range r.fields {
				if r.hasdefault {
					continue
				}
				if needcomma {
					fmt.Fprintf(out, ",")
				} else {
					needcomma = true
				}
				fmt.Fprintf(out, "%s", strings.Join(u1.cases, ","))
			}
			fmt.Fprintf(out, ":\n%s\tdefault:\n%s", badcase, goodcase)
		} else {
			if u.hasdefault {
				fmt.Fprintf(out, "default:\n")
			} else {
				fmt.Fprintf(out, "\tcase %s:\n", strings.Join(u.cases, ","))
			}
			fmt.Fprintf(out, "%s", goodcase)
			if !u.hasdefault {
				fmt.Fprintf(out, "\tdefault:\n%s", badcase)
			}
		}
		fmt.Fprintf(out, "\t}\n");
		fmt.Fprintf(out, "}\n")
	}
	fmt.Fprintf(out, "func (u *%s) XdrUnionTag() interface{} {\n" +
		"\treturn &u.%s\n}\n", r.id, r.tagid)
	fmt.Fprintf(out, "func (u *%s) XdrUnionValid() bool {\n", r.id)
	if r.hasdefault {
		fmt.Fprintf(out, "\treturn true\n")
	} else {
		fmt.Fprintf(out, "\tswitch u.%s {\n" + "\tcase ", r.tagid);
		needcomma := false
		for _, u1 := range r.fields {
			if needcomma {
				fmt.Fprintf(out, ",")
			} else {
				needcomma = true
			}
			fmt.Fprintf(out, "%s", strings.Join(u1.cases, ","))
		}
		fmt.Fprintf(out, ":\n\t\treturn true\n\t}\n\treturn false\n")
	}
	fmt.Fprintf(out, "}\n")
	fmt.Fprintf(out, "func (u *%s) XdrUnionBody() interface{} {\n" +
		"\tswitch u.%s {\n", r.id, r.tagid)
	for _, u := range r.fields {
		if u.hasdefault {
			fmt.Fprintf(out, "\tdefault:\n")
		} else {
			fmt.Fprintf(out, "\tcase %s:\n", strings.Join(u.cases, ","))
		}
		if u.decl.id == "" || u.decl.typ == "void" {
			fmt.Fprintf(out, "\t\treturn nil\n")
		} else {
			fmt.Fprintf(out, "\t\treturn u.%s()\n", u.decl.id)
		}
	}
	fmt.Fprintf(out, "\t}\n" +
		"\treturn nil\n" +
		"}\n")
	e.append(out)
}

func (r *rpc_program) emit(e *emitter) {
	// Do something?
}

func emit(syms *rpc_syms) {
	e := emitter{
		declarations: []string{},
		syms: syms,
		emitted: map[string]struct{}{},
	}

	e.declarations = append(e.declarations, fmt.Sprintf("package main\n"))
	for _, s := range syms.Symbols  {
		e.declarations = append(e.declarations, "\n")
		e.emit(s)
	}
	for _, d := range e.declarations {
		io.WriteString(os.Stdout, d)
	}
}

func main() {
	args := os.Args
	if len(args) <= 1 { return }
	args = args[1:]
	var syms rpc_syms
	for _, arg := range args {
		parseXDR(&syms, arg)
	}
	if syms.Failed {
		os.Exit(1)
	} else {
		emit(&syms)
	}
}