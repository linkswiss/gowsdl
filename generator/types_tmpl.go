// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

var typesTmpl = `
{{define "SimpleType"}}
	//SimpleType
	{{$type := replaceReservedWords .Name | makePublic}}
	{{if processSimpleType $type}}
		{{if .Doc}} {{.Doc | comment}} {{end}}
		{{if .Restriction.Base}}
			type {{$type}} {{toGoType .Restriction.Base}}
			const (
				{{with .Restriction}}
					{{range .Enumeration}}
						{{if .Doc}} {{.Doc | comment}} {{end}}
						{{$type}}{{$value := replaceReservedWords .Value}}{{$value | makePublic}} {{$type}} = "{{$value}}"
					{{end}}
				{{end}}
			)
		{{else}}
		 	{{.UnionType.MemberType | comment}}
			type {{$type}} {{replaceReservedWords .UnionType.MemberType | toGoUnionType}}
			const (
				{{range .UnionType.SimpleType}}
					{{with .Restriction}}
						{{range .Enumeration}}
							{{if .Doc}} {{.Doc | comment}} {{end}}
							{{$type}}{{$value := replaceReservedWords .Value}}{{$value | makePublic}} {{$type}} = "{{$value}}"
						{{end}}
					{{end}}
				{{end}}
			)
		{{end}}
	{{end}}
{{end}}

{{define "Attributes"}}
	//Attributes
	{{range .}}
		{{if .Doc}} {{.Doc | comment}} {{end}}
		{{if .Type}}
			//type
			{{$attributeType := findType .Type }}
			{{ replaceReservedWords .Name | makePublic}} {{$attributeType}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
		{{else if .SimpleType}}
			{{ if .SimpleType.Restriction.Base }}
				//restriction
				{{$attributeType := findType .SimpleType.Restriction.Base }}
				{{ replaceReservedWords .Name | makePublic}} {{$attributeType}} ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
			{{else}}
				//uniontype
				{{.SimpleType.UnionType.MemberType | comment}}
				{{ replaceReservedWords .Name | makePublic}} string ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
			{{end}}
		{{ else }}
			{{ replaceReservedWords .Name | makePublic}} string ` + "`" + `xml:"{{.Name}},attr,omitempty"` + "`" + `
		{{end}}
	{{end}}
{{end}}

{{define "AttributeGroups"}}
	//AttributeGroups
	{{range .}}
		{{if .Doc}} {{.Doc | comment}} {{end}}
		{{if .Ref}}
			{{$attributeType := findType .Ref }}
			{{ replaceReservedWords .Ref | makePublic}} {{$attributeType}}
		{{else}}
			{{$name := replaceReservedWords .Name | makePublic}}
			{{if processSimpleType $name}}
				type {{$name}} struct {
					{{if targetNamespace}}
						XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
					{{else}}
						XMLName xml.Name ` + "`xml:\"{{.Name}}\"`" + `
					{{end}}

					{{template "Attributes" .Attributes}}
				}
			{{end}}
		{{end}}
	{{end}}
{{end}}

{{define "SimpleContent"}}
	//SimpleContent
	{{if .Extension.Attributes}}
		//extension
		{{template "Attributes" .Extension.Attributes}}
	{{ else }}
		{{ $isBaseType := isBaseType .Extension.Base }}
		{{if not $isBaseType}}
			{{$elementType := findType .Extension.Base }}
			Value {{ $elementType }}{{template "Attributes" .Extension.Attributes}}
		{{else}}
			//base
			Value {{toGoType .Extension.Base}}
		{{end}}
	{{ end }}
{{end}}

{{define "ComplexTypeGlobal"}}
	//ComplexTypeGlobal
	{{/* $parent := .ParentName */}}
	{{with .Value}}
		{{/* $name := title .Name | print $parent | replaceReservedWords | makePublic */}}
		{{ $name := replaceReservedWords .Name | makePublic }}
		{{ if processComplexType $name }}
			{{if .Doc}} {{.Doc | comment}} {{end}}
			type {{$name}} struct {
				{{if targetNamespace}}
					XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
				{{else}}
					XMLName xml.Name ` + "`xml:\"{{.Name}}\"`" + `
				{{end}}
				{{ if .Any }}
					Any interface{}
				{{end}}

				{{if ne .ComplexContent.Extension.Base ""}}
					{{with .ComplexContent}}
						//ComplexContent
						{{$baseType := findType .Extension.Base }}
						{{if .Doc}} {{.Doc | comment}} {{end}}
						{{/*$baseType := title .Extension.Base| replaceReservedWords | toGoType*/}}
						{{ if $baseType }}
							//Etension Base
							{{ if eq $baseType "*interface{}"}}
								//{{$baseType}}
							{{else}}
								{{$baseType}}
							{{end}}
						{{end}}

						{{template "Elements" dictValues "ParentName" $name "Values" .Extension.Sequence}}
						{{template "Attributes" .Extension.Attributes}}
					{{end}}
				{{else if ne .SimpleContent.Extension.Base ""}}
					{{template "SimpleContent" .SimpleContent}}
				{{else}}
					{{template "AttributeGroups" .AttributeGoups}}
					{{template "Elements" dictValues "ParentName" $name "Values" .Sequence}}
					{{template "Elements" dictValues "ParentName" $name "Values" .SubSequence}}
					{{template "Elements" dictValues "ParentName" $name "Values" .Choice}}
					{{template "Elements" dictValues "ParentName" $name "Values" .All}}
					{{template "Attributes" .Attributes}}
				{{end}}
			}
			{{if ne .ComplexContent.Extension.Base ""}}
				{{template "ElementsTypes" dictValues "ParentName" $name "Values" .ComplexContent.Extension.Sequence}}
			{{else if ne .SimpleContent.Extension.Base ""}}
				{{template "ElementsTypes" dictValues "ParentName" $name "Values" .SimpleContent.Extension.Sequence}}
			{{else}}
				{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Sequence}}
				{{template "ElementsTypes" dictValues "ParentName" $name "Values" .SubSequence}}
				{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Choice}}
				{{template "ElementsTypes" dictValues "ParentName" $name "Values" .All}}
			{{end}}
		{{ end }}
	{{ end }}
{{end}}

{{define "ComplexTypeLocal"}}
	//ComplexTypeLocal
	{{/* $parent := .ParentName */}}
	{{with .Value}}
		{{/* $name := title .Name | print $parent | replaceReservedWords | makePublic */}}
		{{ $name := title .Name | replaceReservedWords | makePublic }}
		{{ if processComplexType $name }}
			{{if .Doc}} {{.Doc | comment}} {{end}}
			type {{ $name }} struct {
				{{if targetNamespace}}
					XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
				{{else}}
					XMLName xml.Name ` + "`xml:\"{{.Name}}\"`" + `
				{{end}}
				{{ if .Any }}
					Any interface{}
				{{end}}

				{{with .ComplexType}}
					{{if ne .ComplexContent.Extension.Base ""}}
						{{with .ComplexContent}}
							//ComplexContent
							{{$baseType := findType .Extension.Base }}
							{{if .Doc}} {{.Doc | comment}} {{end}}
							{{/*$baseType := title .Extension.Base| replaceReservedWords | toGoType*/}}
							{{ if $baseType }}
								//Etension Base
								{{ if eq $baseType "*interface{}"}}
									//{{$baseType}}
								{{else}}
									{{$baseType}}
								{{end}}
							{{end}}

							{{template "Elements" dictValues "ParentName" $name "Values" .Extension.Sequence}}
							{{template "Attributes" .Extension.Attributes}}
						{{end}}
					{{else if ne .SimpleContent.Extension.Base ""}}
						{{template "SimpleContent" .SimpleContent}}
					{{else}}
						{{template "AttributeGroups" .AttributeGoups}}
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
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .ComplexContent.Extension.Sequence}}
				{{else if ne .SimpleContent.Extension.Base ""}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .SimpleContent.Extension.Sequence}}
				{{else}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Sequence}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .SubSequence}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .Choice}}
					{{template "ElementsTypes" dictValues "ParentName" $name "Values" .All}}
				{{end}}
			{{end}}
		{{ end }}
	{{end}}
{{end}}

{{define "ElementsTypes"}}
	//ElementsTypes
	{{/* $parent := .ParentName */}}
	{{range .Values}}
		{{if not .Ref}}
			{{if not .SimpleType}}
				{{ if not .Type }}
					{{template "ComplexTypeLocal" dictValues "ParentName" "" "Value" .}}
				{{end}}
			{{end}}
		{{end}}
	{{end}}
{{end}}

{{define "Elements"}}
	//Elements
	{{/* $parent := .ParentName */}}
	{{range .Values}}
		{{if .Doc}} {{.Doc | comment}} {{end}}
		{{if not .Type}}
			//not type
			{{if .SimpleType}}
				//simple
				{{if .SimpleType.Doc}} {{.SimpleType.Doc | comment}} {{end}}
				{{ replaceReservedWords .Name | makePublic}} {{toGoType .SimpleType.Restriction.Base}} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{else if .Ref}}
				//ref
				{{$elementType := findType .Ref }}
				{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
				{{ if .Name }}
					{{replaceReservedWords .Ref | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ $elementType }} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
				{{else}}
					{{replaceReservedWords .Ref | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ $elementType }}
				{{end}}
			{{else}}
				//else
				{{$elementType := findType .Name }}
				{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
				{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ $elementType }} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{end}}
		{{else}}
			//type
			{{ $isBaseType := isBaseType .Type }}
			{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
			{{ if $isBaseType }}
				//basetype
				{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ toGoType .Type }} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{ else }}
				//else
				{{$elementType := findType .Type }}
				{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ $elementType }} ` + "`" + `xml:"{{.Name}},omitempty"` + "`" + `
			{{ end }}
		{{end}}
	{{end}}
{{end}}

	{{ setCurrentSchema . }}
	{{ $parent := .Parent }}
	{{range .SimpleType}}
		{{template "SimpleType" .}}
	{{end}}
	{{range .ComplexTypes}}
		{{template "ComplexTypeGlobal" dictValues "ParentName" "" "Value" .}}
	{{end}}
	{{range .Elements}}
		{{if not .Type}}
			{{template "ComplexTypeLocal" dictValues "ParentName" "" "Value" .}}
		{{else}}
			//ELEMENT TYPE
			{{$name := title .Name | replaceReservedWords | makePublic}}
			{{if processComplexType $name}}
				{{if .Doc}} {{.Doc | comment}} {{end}}
				type {{ $name }} struct {
					{{if targetNamespace}}
						XMLName xml.Name ` + "`xml:\"{{targetNamespace}} {{.Name}}\"`" + `
					{{else}}
						XMLName xml.Name ` + "`xml:\"{{.Name}}\"`" + `
					{{end}}

					{{if isArrayElement .MaxOccurs }}//MAX OCCUR {{ .MaxOccurs }}{{end}}
					{{ $isBaseType := isBaseType .Type }}
					{{if not $isBaseType}}
						{{replaceReservedWords .Type | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ title .Type | replaceReservedWords | toGoType}}
					{{else}}
						{{replaceReservedWords .Name | makePublic}} {{if isArrayElement .MaxOccurs }}[]{{end}}{{ .Type | replaceReservedWords | toGoType}}
					{{end}}
				}
			{{end}}
		{{end}}
	{{end}}

	{{template "AttributeGroups" .AttributeGoups}}
`
