//// This Source Code Form is subject to the terms of the Mozilla Public
//// License, v. 2.0. If a copy of the MPL was not distributed with this
//// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//package generator
//
//var typesTmpl = `
//{{define "SimpleType"}}
//	{{$type := replaceReservedWords .Name | makePublic}}
//	{{if processSimpleType $type}}
//		type {{$type}} {{toGoType .Restriction.Base}}
//		const (
//			{{with .Restriction}}
//				{{range .Enumeration}}
//					{{if .Doc}} {{.Doc | comment}} {{end}}
//					{{$type}}{{$value := replaceReservedWords .Value}}{{$value | makePublic}} {{$type}} = "{{$value}}" {{end}}
//			{{end}}
//		)
//	{{end}}
//{{end}}
//
//{{define "ComplexContent"}}
//	{{$baseType := toGoType .Extension.Base}}
//	{{ if $baseType }}
//		{{$baseType}}
//	{{end}}
//
//	{{template "Elements" .Extension.Sequence}}
//	{{template "Attributes" .Extension.Attributes}}
//{{end}}
//
//{{define "Attributes"}}
//	{{range .}}
//		{{if .Doc}} {{.Doc | comment}} {{end}} {{if not .Type}}
//			{{ .Name | makePublic}} {{toGoType .SimpleType.Restriction.Base}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
//		{{else}}
//			{{ .Name | makePublic}} {{toGoType .Type}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
//		{{end}}
//	{{end}}
//{{end}}
//
//{{define "SimpleContent"}}
//	Value {{toGoType .Extension.Base}}{{template "Attributes" .Extension.Attributes}}
//{{end}}
//
//{{define "ComplexTypeGlobal"}}
//	{{$name := replaceReservedWords .Name | makePublic}}
//	{{if processComplexType $name}}
//		type {{$name}} struct {
//			//XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
//			{{if ne .ComplexContent.Extension.Base ""}}
//				{{template "ComplexContent" .ComplexContent}}
//			{{else if ne .SimpleContent.Extension.Base ""}}
//				{{template "SimpleContent" .SimpleContent}}
//			{{else}}
//				{{template "Elements" .Sequence}}
//				{{template "Elements" .Choice}}
//				{{template "Elements" .All}}
//				{{template "Attributes" .Attributes}}
//			{{end}}
//		}
//
//		{{if ne .ComplexContent.Extension.Base ""}}
//		{{else if ne .SimpleContent.Extension.Base ""}}
//		{{else}}
//			{{template "ElementsTypes" .Sequence}}
//			{{template "ElementsTypes" .Choice}}
//			{{template "ElementsTypes" .All}}
//		{{end}}
//	{{end}}
//{{end}}
//
//{{define "ComplexTypeLocal"}}
//	{{$name := .Name}}
//	{{if processComplexType $name}}
//		type {{$name | replaceReservedWords | makePublic}} struct {
//			//XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{$name}}\"`" + `
//			{{with .ComplexType}}
//				{{if ne .ComplexContent.Extension.Base ""}}
//					{{template "ComplexContent" .ComplexContent}}
//				{{else if ne .SimpleContent.Extension.Base ""}}
//					{{template "SimpleContent" .SimpleContent}}
//				{{else}}
//					{{template "Elements" .Sequence}}
//					{{template "Elements" .SubSequence}}
//					{{template "Elements" .Choice}}
//					{{template "Elements" .All}}
//					{{template "Attributes" .Attributes}}
//				{{end}}
//			{{end}}
//		}
//
//		{{with .ComplexType}}
//			{{if ne .ComplexContent.Extension.Base ""}}
//			{{else if ne .SimpleContent.Extension.Base ""}}
//			{{else}}
//				{{template "ElementsTypes" .Sequence}}
//				{{template "ElementsTypes" .SubSequence}}
//				{{template "ElementsTypes" .Choice}}
//				{{template "ElementsTypes" .All}}
//			{{end}}
//		{{end}}
//	{{end}}
//{{end}}
//
//{{define "ComplexTypeInline"}}
//	{{replaceReservedWords .Name | makePublic}} struct {
//	{{with .ComplexType}}
//		{{if ne .ComplexContent.Extension.Base ""}}
//			{{template "ComplexContent" .ComplexContent}}
//		{{else if ne .SimpleContent.Extension.Base ""}}
//			{{template "SimpleContent" .SimpleContent}}
//		{{else}}
//			{{template "Elements" .Sequence}}
//			{{template "Elements" .Choice}}
//			{{template "Elements" .All}}
//			{{template "Attributes" .Attributes}}
//		{{end}}
//	{{end}}
//	} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
//{{end}}
//
//{{define "ElementsTypes"}}
//	{{range .}}
//		{{if not .SimpleType}}
//			{{template "ComplexTypeLocal" .}}
//		{{end}}
//	{{end}}
//{{end}}
//
//{{define "Elements"}}
//	{{range .}}
//		{{if .Doc}} {{.Doc | comment}} {{end}}
//		{{if not .Type}}
//			{{if .SimpleType}}
//				{{if .SimpleType.Doc}} {{.SimpleType.Doc | comment}} {{end}}
//				{{ .Name | makePublic}} {{toGoType .SimpleType.Restriction.Base}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
//			{{else}}
//				{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
//				{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{replaceReservedWords .Name | toGoType}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
//			{{end}}
//		{{else}}
//			{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
//			{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{replaceReservedWords .Type | toGoType}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
//		{{end}}
//	{{end}}
//{{end}}
//
//{{define "ElementsTypesOLD"}}
//{{end}}
//
//{{define "ElementsOLD"}}
//	{{range .}}
//		{{if not .Type}}
//			{{template "ComplexTypeInline" .}}
//		{{else}}
//			{{if .Doc}} {{.Doc | comment}} {{end}}
//			{{replaceReservedWords .Name | makePublic}} {{if eq .MaxOccurs "unbounded"}}[]{{end}}{{replaceReservedWords .Type | toGoType}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
//		{{end}}
//	{{end}}
//{{end}}
//
//{{range .Schemas}}
//	{{range .SimpleType}}
//		{{template "SimpleType" .}}
//	{{end}}
//	{{range .Elements}}
//		{{if not .Type}}
//			{{template "ComplexTypeLocal" .}}
//		{{end}}
//	{{end}}
//	{{range .ComplexTypes}}
//		{{template "ComplexTypeGlobal" .}}
//	{{end}}
//{{end}}
//`


// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

var typesTmpl = `
{{define "SimpleType"}}
	{{$type := replaceReservedWords .Name | makePublic}}
	{{if processSimpleType $type}}
		type {{$type}} {{toGoType .Restriction.Base}}
		const (
			{{with .Restriction}}
				{{range .Enumeration}}
					{{if .Doc}} {{.Doc | comment}} {{end}}
					{{$type}}{{$value := replaceReservedWords .Value}}{{$value | makePublic}} {{$type}} = "{{$value}}" {{end}}
			{{end}}
		)
	{{end}}
{{end}}

{{define "ComplexContent"}}
	{{$baseType := toGoType .Extension.Base}}
	{{ if $baseType }}
		{{$baseType}}
	{{end}}

	{{template "Elements" dictValues "ParentName" "" "Values" .Extension.Sequence}}
	{{template "Attributes" .Extension.Attributes}}
{{end}}

{{define "Attributes"}}
	{{range .}}
		{{if .Doc}} {{.Doc | comment}} {{end}} {{if not .Type}}
			{{ .Name | makePublic}} {{toGoType .SimpleType.Restriction.Base}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
		{{else}}
			{{ .Name | makePublic}} {{toGoType .Type}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
		{{end}}
	{{end}}
{{end}}

{{define "SimpleContent"}}
	Value {{toGoType .Extension.Base}}{{template "Attributes" .Extension.Attributes}}
{{end}}

{{define "ComplexTypeGlobal"}}
	{{$name := replaceReservedWords .Name | makePublic}}
	{{if processComplexType $name}}
		type {{$name}} struct {
			XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
			{{if ne .ComplexContent.Extension.Base ""}}
				{{template "ComplexContent" .ComplexContent}}
			{{else if ne .SimpleContent.Extension.Base ""}}
				{{template "SimpleContent" .SimpleContent}}
			{{else}}
				{{template "Elements" dictValues "ParentName" $name "Values" .Sequence}}
				{{template "Elements" dictValues "ParentName" $name "Values" .SubSequence}}
				{{template "Elements" dictValues "ParentName" $name "Values" .Choice}}
				{{template "Elements" dictValues "ParentName" $name "Values" .All}}
				{{template "Attributes" .Attributes}}
			{{end}}
		}

		{{if ne .ComplexContent.Extension.Base ""}}
		{{else if ne .SimpleContent.Extension.Base ""}}
		{{else}}
			{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Sequence}}
			{{template "ElementsTypes" dictValues "ParentName" $name "Values" .SubSequence}}
			{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Choice}}
			{{template "ElementsTypes" dictValues "ParentName" $name "Values" .All}}
		{{end}}
	{{end}}
{{end}}

{{define "ComplexTypeLocal"}}
	{{ $parent := .ParentName }}
	{{with .Value}}
		{{$name := title .Name | print $parent | replaceReservedWords | makePublic}}
		{{if processComplexType $name}}
			type {{ $name }} struct {
				XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
				{{with .ComplexType}}
					{{if ne .ComplexContent.Extension.Base ""}}
						{{template "ComplexContent" .ComplexContent}}
					{{else if ne .SimpleContent.Extension.Base ""}}
						{{template "SimpleContent" .SimpleContent}}
					{{else}}
						{{template "Elements" dictValues "ParentName" $name "Values" .Sequence}}
						{{template "Elements" dictValues "ParentName" $name "Values" .SubSequence}}
						{{template "Elements" dictValues "ParentName" $name "Values" .Choice}}
						{{template "Elements" dictValues "ParentName" $name "Values" .All}}
						{{template "Attributes" .Attributes}}
					{{end}}
				{{end}}
			}

			{{with .ComplexType}}
				{{if ne .ComplexContent.Extension.Base ""}}
				{{else if ne .SimpleContent.Extension.Base ""}}
				{{else}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Sequence}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .SubSequence}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Choice}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .All}}
				{{end}}
			{{end}}
		{{end}}
	{{end}}

{{end}}

{{define "ElementsTypes"}}
	{{ $parent := .ParentName }}

	{{range .Values}}
		{{if not .SimpleType}}
			{{template "ComplexTypeLocal" dictValues "ParentName" $parent "Value" .}}
		{{end}}
	{{end}}
{{end}}

{{define "Elements"}}
	{{ $parent := .ParentName }}

	{{range .Values}}
		{{if .Doc}} {{.Doc | comment}} {{end}}
		{{if not .Type}}
			{{if .SimpleType}}
				{{if .SimpleType.Doc}} {{.SimpleType.Doc | comment}} {{end}}
				{{ replaceReservedWords .Name | makePublic}} {{toGoType .SimpleType.Restriction.Base}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{else}}
				{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
				{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ title .Name | print $parent | replaceReservedWords | toGoType}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{end}}
		{{else}}
			{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
			{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ title .Name | print $parent |replaceReservedWords | toGoType}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
		{{end}}
	{{end}}
{{end}}

{{range .Schemas}}
	{{ setCurrentSchema . }}
	{{range .SimpleType}}
		{{template "SimpleType" .}}
	{{end}}
	{{range .Elements}}
		{{if not .Type}}
			{{template "ComplexTypeLocal" dictValues "ParentName" "" "Value" .}}
		{{end}}
	{{end}}
	{{range .ComplexTypes}}
		{{template "ComplexTypeGlobal" .}}
	{{end}}
{{end}}
`
