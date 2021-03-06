// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"
	"strconv"
	"path"
//	"github.com/davecgh/go-spew/spew"
)

const maxRecursion uint8 = 100

type GoWsdl struct {
	file, pkg             string
	ignoreTls             bool
	wsdl                  *Wsdl
	resolvedXsdExternals  map[string]*XsdSchema
	importsNeeded		  map[string]bool
	currentRecursionLevel uint8
	processedComplexTypes map[string]map[string]bool
	processedSimpleTypes  map[string]map[string]bool
	currentSchema         *XsdSchema
}

type HeaderElements struct {
	Pkg 					string
	PkgBase					string
	ImportsNeeded			map[string]bool
	ResolvedXsdExternals  	map[string]*XsdSchema
}

var cacheDir = filepath.Join(os.TempDir(), "gowsdl-cache")

func init() {
	err := os.MkdirAll(cacheDir, 0700)
	if err != nil {
		Log.Crit("Create cache directory", "error", err)
		os.Exit(1)
	}
}

var timeout = time.Duration(30 * time.Second)

func dialTimeout(network, addr string) (net.Conn, error) {
	return net.DialTimeout(network, addr, timeout)
}

func downloadFile(url string, ignoreTls bool) ([]byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: ignoreTls,
		},
		Dial: dialTimeout,
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func NewGoWsdl(file, pkg string, ignoreTls bool) (*GoWsdl, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		Log.Crit("WSDL file is required to generate Go proxy")
		os.Exit(2)
	}

	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		pkg = "myservice"
	}

	return &GoWsdl{
		file:      file,
		pkg:       pkg,
		ignoreTls: ignoreTls,
	}, nil
}

func (g *GoWsdl) Start() (map[string][]byte, map[string][]byte, error) {
	gocode := make(map[string][]byte)
	var gotypes map[string][]byte

	err := g.unmarshal()
	if err != nil {
		return nil, nil, err
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error

		gotypes, err = g.genTypes()

		//		gocode["types"], err = g.genTypes()
		if err != nil {
			Log.Error("genTypes", "error", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error

		gocode["operations"], err = g.genOperations()
		if err != nil {
			Log.Error("genOperations", "error", err)
		}
	}()

	wg.Wait()

	gocode["header"], err = g.genHeader()
	if err != nil {
		Log.Error("genHeader", "error", err)
	}

	return gocode, gotypes, nil
}

func (g *GoWsdl) unmarshal() error {
	var data []byte

	parsedUrl, err := url.Parse(g.file)
	if parsedUrl.Scheme == "" {
		Log.Info("Reading", "file", g.file)

		data, err = ioutil.ReadFile(g.file)
		if err != nil {
			return err
		}
	} else {
		Log.Info("Downloading", "file", g.file)

		data, err = downloadFile(g.file, g.ignoreTls)
		if err != nil {
			return err
		}
	}

	g.wsdl = &Wsdl{}
	err = xml.Unmarshal(data, g.wsdl)
	if err != nil {
		return err
	}

	//	spew.Dump(g.wsdl.Types.Schemas)

	for _, schema := range g.wsdl.Types.Schemas {
		err = g.resolveXsdExternals(schema, parsedUrl)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *GoWsdl) resolveXsdExternals(schema *XsdSchema, url *url.URL) error {
	for _, impor := range schema.Imports {
		location, err := url.Parse(impor.SchemaLocation)
		if err != nil {
			return err
		}

		schemaName := getSchemaName(location.Path)
		if g.resolvedXsdExternals[schemaName] != nil {
			continue
		}

		data, err := ioutil.ReadFile(location.Path)
		if err != nil {
			return err
		}
		newschema := &XsdSchema{}

		err = xml.Unmarshal(data, newschema)
		if err != nil {
			return err
		}

		//		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, newschema)

		if g.resolvedXsdExternals == nil {
			g.resolvedXsdExternals = make(map[string]*XsdSchema, maxRecursion)
		}
		g.resolvedXsdExternals[schemaName] = newschema

		if len(newschema.Includes) > 0 &&
		maxRecursion > g.currentRecursionLevel {

			g.currentRecursionLevel++

			//log.Printf("Entering recursion %d\n", g.currentRecursionLevel)
			g.resolveXsdExternals(newschema, url)
		}
	}

	for _, incl := range schema.Includes {
		location, err := url.Parse(incl.SchemaLocation)
		if err != nil {
			return err
		}

		schemaName := getSchemaName(location.Path)
		if g.resolvedXsdExternals[schemaName] != nil {
			continue
		}

		schemaLocation := location.String()
		if !location.IsAbs() {
			if !url.IsAbs() {
				return errors.New(fmt.Sprintf("Unable to resolve external schema %s through WSDL URL %s", schemaLocation, url))
			}
			schemaLocation = url.Scheme + "://" + url.Host + schemaLocation
		}

		Log.Info("Downloading external schema", "location", schemaLocation)

		data, err := downloadFile(schemaLocation, g.ignoreTls)
		newschema := &XsdSchema{}

		err = xml.Unmarshal(data, newschema)
		if err != nil {
			return err
		}

		//		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, newschema)

		if g.resolvedXsdExternals == nil {
			g.resolvedXsdExternals = make(map[string]*XsdSchema, maxRecursion)
		}
		g.resolvedXsdExternals[schemaName] = newschema


		if len(newschema.Includes) > 0 &&
		maxRecursion > g.currentRecursionLevel {

			g.currentRecursionLevel++

			//log.Printf("Entering recursion %d\n", g.currentRecursionLevel)
			g.resolveXsdExternals(newschema, url)
		}
	}

	return nil
}

//Generate types, included and imported schemas are under it's own namespaces, others under basetypes
func (g *GoWsdl) genTypes() (map[string][]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"toGoUnionType":        toGoUnionType,
		"isBaseType":			isBaseType,
		"findType":             g.findType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"processComplexType" :  g.processComplexType,
		"processSimpleType" :   g.processSimpleType,
		"makePublic":           makePublic,
		"comment":              comment,
		"title":                strings.Title,
		"isArrayElement":        isArrayElement,
		"dictValues":            dictValues,
		"setCurrentSchema":     g.setCurrentSchema,
		"targetNamespace":      g.targetNamspace,
		"getSchemaName":		getSchemaName,
		//		"targetNamespace":      func() string { return g.wsdl.TargetNamespace },
	}

	//TODO resolve element refs in place.
	//g.resolveElementsRefs()

	gotypes := make(map[string][]byte)
	g.importsNeeded = make(map[string]bool,100)

//	pkgName := path.Base(g.pkg)
//	headerElem := HeaderElements{
//		Pkg: "basetypes",
//		PkgBase: g.pkg,
//		Imports: nil,
//		Includes: nil,
//	}
//
//	data := new(bytes.Buffer)
//	tmpl := template.Must(template.New("includetHeader").Funcs(funcMap).Parse(includeHeaderTmpl))
//	err := tmpl.Execute(data, headerElem)
//	if err != nil {
//		return nil, err
//	}
//	gotypes["basetypes"] = data.Bytes()

	for _, schema := range g.wsdl.Types.Schemas {
		schema.Parent = "basetypes"

		data := new(bytes.Buffer)
		tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
		err := tmpl.Execute(data, schema)
		if err != nil {
			return nil, err
		}

		schemaBytes := bytes.TrimSpace(data.Bytes())
		if (len(schemaBytes) > 0) {
			gotypes["basetypes"] = append(gotypes["basetypes"], data.Bytes()...)
		}
	}

	headerElem := HeaderElements{
		Pkg: "basetypes",
		PkgBase: g.pkg,
		ImportsNeeded: g.importsNeeded,
	}

	headerData := new(bytes.Buffer)
	tmpl := template.Must(template.New("includetHeader").Funcs(funcMap).Parse(includeHeaderTmpl))
	err  := tmpl.Execute(headerData, headerElem)
	if err != nil {
		return nil, err
	}

	content := gotypes["basetypes"]
	gotypes["basetypes"] = headerData.Bytes()
	gotypes["basetypes"] = append(gotypes["basetypes"],content...)


	for key, schema := range g.resolvedXsdExternals {
		name := key
		schema.Parent = key
		g.importsNeeded = make(map[string]bool,100)

//		namespace := path.Base(schema.TargetNamespace)
//		namespace = replaceReservedWords(namespace)
//		schema.Parent = namespace
//		pkg := replaceReservedWords(name)
//		headerElem := HeaderElements{
//			Pkg: pkg,
//			PkgBase: g.pkg,
//			Imports: schema.Imports,
//			Includes: schema.Includes,
//		}
//
//		hederdata := new(bytes.Buffer)
//		tmplhead := template.Must(template.New("includetHeader").Funcs(funcMap).Parse(includeHeaderTmpl))
//		err := tmplhead.Execute(hederdata, headerElem)
//		if err != nil {
//			return nil, err
//		}
//		gotypes[name] = hederdata.Bytes()

		data := new(bytes.Buffer)
		tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
		err := tmpl.Execute(data, schema)
		if err != nil {
			return nil, err
		}

		schemaBytes := bytes.TrimSpace(data.Bytes())
		if (len(schemaBytes) > 0) {
			gotypes[name] = append(gotypes[name], data.Bytes()...)
		}

		pkg := replaceReservedWords(name)
		headerElem := HeaderElements{
			Pkg: pkg,
			PkgBase: g.pkg,
			ImportsNeeded: g.importsNeeded,
		}

		headerData := new(bytes.Buffer)
		tmplhead := template.Must(template.New("includetHeader").Funcs(funcMap).Parse(includeHeaderTmpl))
		err = tmplhead.Execute(headerData, headerElem)
		if err != nil {
			return nil, err
		}

		content := gotypes[name]
		gotypes[name] = headerData.Bytes()
		gotypes[name] = append(gotypes[name],content...)
	}

	return gotypes, nil
}

// func (g *GoWsdl) resolveElementsRefs() error {
// 	for _, schema := range g.wsdl.Types.Schemas {
// 		for _, globalEl := range schema.Elements {
// 			for _, localEl := range globalEl.ComplexType.Sequence.Elements {

// 			}
// 		}
// 	}
// }

func (g *GoWsdl) genOperations() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           makePublic,
		"findType":             g.findMessageType,
		"findSoapAction":       g.findSoapAction,
		"findServiceAddress":   g.findServiceAddress,
		"replaceStar":			replaceStar,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("operations").Funcs(funcMap).Parse(opsTmpl))
	err := tmpl.Execute(data, g.wsdl.PortTypes)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (g *GoWsdl) genHeader() ([]byte, error) {
	funcMap := template.FuncMap{
		"toGoType":             toGoType,
		"stripns":              stripns,
		"replaceReservedWords": replaceReservedWords,
		"makePublic":           makePublic,
		"findType":             g.findType,
		"comment":              comment,
		"getSchemaName":		getSchemaName,
	}

	pkgName := path.Base(g.pkg)
	headerElem := HeaderElements{
		Pkg: pkgName,
		PkgBase: g.pkg,
		ResolvedXsdExternals: g.resolvedXsdExternals,
	}

	data := new(bytes.Buffer)
	tmpl := template.Must(template.New("header").Funcs(funcMap).Parse(headerTmpl))
	err := tmpl.Execute(data, headerElem)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

var reservedWords = map[string]string{
	"break":       "break_",
	"default":     "default_",
	"func":        "func_",
	"interface":   "interface_",
	"select":      "select_",
	"case":        "case_",
	"defer":       "defer_",
	"go":          "go_",
	"map":         "map_",
	"struct":      "struct_",
	"chan":        "chan_",
	"else":        "else_",
	"goto":        "goto_",
	"package":     "package_",
	"switch":      "switch_",
	"const":       "const_",
	"fallthrough": "fallthrough_",
	"if":          "if_",
	"range":       "range_",
	"type":        "type_",
	"continue":    "continue_",
	"for":         "for_",
	"import":      "import_",
	"return":      "return_",
	"var":         "var_",
}

func getSchemaName(schemalocation string) string{
	_, schemaName := filepath.Split(schemalocation)
	schemaName = strings.Replace(schemaName, ".", "", -1)
	schemaName = strings.Replace(schemaName, "xsd", "", -1)
	return schemaName
}

func replaceStar(identifier string) string {
	return strings.Replace(identifier, "*", "", -1)
}

// Replaces Go reserved keywords to avoid compilation issues
func replaceReservedWords(identifier string) string {
	//Rplace _ to be consistent in the element pointer definition
	identifier = strings.Replace(identifier, "_", "", -1)
	value := reservedWords[identifier]
	if value != "" {
		return value
	}

	r := strings.Split(identifier, ":")
	if len(r) == 2 {
		return identifier
	}

	return normalize(identifier)
}

// Normalizes value to be used as a valid Go identifier, avoiding compilation issues
func normalize(value string) string {
	mapping := func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}

	return strings.Map(mapping, value)
}

var xsd2GoTypes = map[string]string{
	"string":        "string",
	"token":         "string",
	"NMTOKEN": 		 "string",
	"language": 	 "string",
	"anyURI":        "string",
	"float":         "float32",
	"double":        "float64",
	"decimal":       "float64",
	"integer":       "int32",
	"int":           "int32",
	"short":         "int16",
	"byte":          "int8",
	"long":          "int64",
	"boolean":       "bool",
	"dateTime":      "time.Time",
	"date":          "time.Time",
	"time":          "time.Time",
	"duration":      "time.Time",
	"gYear":      	 "time.Time",
	"base64Binary":  "[]byte",
	"hexBinary":     "[]byte",
	"positiveInteger": "uint32",
	"nonNegativeInteger": "uint32",
	"unsignedInt":   "uint32",
	"unsignedShort": "uint16",
	"unsignedByte":  "byte",
	"unsignedLong":  "uint64",
	"anyType":       "*interface{}",

	"StringLength1to16": "string",
	"StringLength1to32": "string",
}

func isBaseType(xsdType string) bool {
	// Handles name space, ie. xsd:string, xs:string
	r := strings.Split(xsdType, ":")
	type_ := r[0]

	if len(r) == 2 {
		type_ = r[1]
	}

	value := xsd2GoTypes[type_]
	if value != "" {
		return true
	}

	return false
}

func toGoType(xsdType string) string {
	// Handles name space, ie. xsd:string, xs:string
	r := strings.Split(xsdType, ":")
	type_ := r[0]

	if len(r) == 2 {
		type_ = r[1]
	}

	value := xsd2GoTypes[type_]
	if value != "" {
		return value
	}

	return "*" + makePublic(type_)
}

func toGoUnionType(xsdType string) string {
	// Handles name space, ie. xsd:string, xs:string
	return "string"
}

// Check if the ComplexType is already been processed
func (g *GoWsdl) processComplexType(complexType string) bool {
//	return true
	if g.processedComplexTypes[g.currentSchema.Parent] != nil && g.processedComplexTypes[g.currentSchema.Parent][complexType] {
		return false
	}else {
		if g.processedComplexTypes == nil {
			g.processedComplexTypes = make(map[string]map[string]bool, 10000)
		}
		if g.processedComplexTypes[g.currentSchema.Parent] == nil{
			g.processedComplexTypes[g.currentSchema.Parent] = make(map[string]bool, 10000)
		}
		g.processedComplexTypes[g.currentSchema.Parent][complexType] = true
		return true
	}
}

// Check if the SimpleType is already been processed
func (g *GoWsdl) processSimpleType(simpleType string) bool {
//	return true
	if g.processedSimpleTypes[g.currentSchema.Parent] != nil && g.processedSimpleTypes[g.currentSchema.Parent][simpleType] {
		return false
	}else {
		if g.processedSimpleTypes == nil {
			g.processedSimpleTypes = make(map[string]map[string]bool, 10000)
		}
		if g.processedSimpleTypes[g.currentSchema.Parent] == nil{
			g.processedSimpleTypes[g.currentSchema.Parent] = make(map[string]bool, 10000)
		}
		g.processedSimpleTypes[g.currentSchema.Parent][simpleType] = true
		return true
	}
}

// Check if the SimpleType is already been processed
func (g *GoWsdl) setCurrentSchema(schema *XsdSchema) string {
	g.currentSchema = schema
	return ""
}

// Check if the SimpleType is already been processed
func (g *GoWsdl) targetNamspace() string {
	if (g.currentSchema != nil && g.currentSchema.TargetNamespace != "") {
		return g.currentSchema.TargetNamespace
	}else {
		return g.wsdl.TargetNamespace
	}
}

// Given a message, finds its type.
//
// I'm not very proud of this function but
// it works for now and performance doesn't
// seem critical at this point
func (g *GoWsdl) findType(xmlType string) string {
	elRefName := stripns(xmlType)
	//	Log.Info(elRef)

	for _, msg := range g.wsdl.Messages {
		if msg.Name != elRefName {
			continue
		}

		// Assumes document/literal wrapped WS-I
		if len(msg.Parts) == 0 {
			// Message does not have parts. This could be a Port
			// with HTTP binding or SOAP 1.2 binding, which are not currently
			// supported.
			Log.Warn("WSDL does seem to have HTTP or SOAP 1.2 binding which is not currently supported.")
			continue
		}
		part := msg.Parts[0]
		if part.Type != "" {
			return stripns(part.Type)
		}

		elRef := stripns(part.Element)

		for _, schema := range g.wsdl.Types.Schemas {
			for _, el := range schema.Elements {
				if strings.EqualFold(elRef, el.Name) {
					if el.Type != "" {
						return stripns(el.Type)
					}

					if(!g.importsNeeded["basetypes"]){
						g.importsNeeded["basetypes"] = true
					}

					fullname := "basetypes."+makePublic(replaceReservedWords(el.Name))
					return fullname
					//					return el.Name
				}
			}
		}

		//Inserted here in the Messages for in order to avoid reference not corrected (Amadeus for example!)
		//TODO Check if it works for all the wsdl -> if there are some as the OTA xsd could be a problem... to solve move this for out of the messages for
		for key, schema := range g.resolvedXsdExternals {
			fullname := ""
			for _, el := range schema.SimpleType {
				if strings.EqualFold(elRef, el.Name) {
					if(key == g.currentSchema.Parent){
						fullname = "*" +makePublic(replaceReservedWords(el.Name))
						//						fullname = "*" +makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
					}else{
						pkg := makePublic(replaceReservedWords(key))
						if(!g.importsNeeded[pkg]){
							g.importsNeeded[pkg] = true
						}
						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
					}
					return fullname
				}
			}
			for _, el := range schema.ComplexTypes {
				if strings.EqualFold(elRef, el.Name) {
					if(key == g.currentSchema.Parent){
						fullname = "*" +makePublic(replaceReservedWords(el.Name))
						//						fullname = "*" +makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
					}else{
						pkg := makePublic(replaceReservedWords(key))
						if(!g.importsNeeded[pkg]){
							g.importsNeeded[pkg] = true
						}
						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
					}
					return fullname
				}			}
			for _, el := range schema.AttributeGoups {
				if strings.EqualFold(elRef, el.Name) {
					if(key == g.currentSchema.Parent){
						fullname = "*" +makePublic(replaceReservedWords(el.Name))
						//						fullname = "*" +makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
					}else{
						pkg := makePublic(replaceReservedWords(key))
						if(!g.importsNeeded[pkg]){
							g.importsNeeded[pkg] = true
						}
						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
					}
					return fullname
				}
			}

			for _, el := range schema.Elements {
				if strings.EqualFold(elRef, el.Name) {
					elName := ""
					if el.Type != "" {
						elName = stripns(el.Type)
					}else{
						elName = makePublic(replaceReservedWords(el.Name))
					}

					if(key == g.currentSchema.Parent){
						fullname = "*" +makePublic(replaceReservedWords(elName))
						//						fullname = "*" +makePublic(replaceReservedWords(key))+"."+elName
					}else{
						pkg := makePublic(replaceReservedWords(key))
						if(!g.importsNeeded[pkg]){
							g.importsNeeded[pkg] = true
						}
						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(elName))
					}
					//					Log.Info(fullname)
					return fullname
				}
			}
		}
	}


	if(isBaseType(xmlType)){
		return toGoType(replaceReservedWords(xmlType))
	}else{
		return toGoType(replaceReservedWords(strings.Title(xmlType)))
	}
}

func (g *GoWsdl) findMessageType(xmlType string) string {
	elRefName := stripns(xmlType)
	//	Log.Info(elRef)

	for _, msg := range g.wsdl.Messages {
		if msg.Name != elRefName {
			continue
		}

		// Assumes document/literal wrapped WS-I
		if len(msg.Parts) == 0 {
			// Message does not have parts. This could be a Port
			// with HTTP binding or SOAP 1.2 binding, which are not currently
			// supported.
			Log.Warn("WSDL does seem to have HTTP or SOAP 1.2 binding which is not currently supported.")
			continue
		}
		part := msg.Parts[0]
		if part.Type != "" {
			return stripns(part.Type)
		}

		elRef := stripns(part.Element)

		for _, schema := range g.wsdl.Types.Schemas {
			for _, el := range schema.Elements {
				if strings.EqualFold(elRef, el.Name) {
					if el.Type != "" {
						return stripns(el.Type)
					}

					if(!g.importsNeeded["basetypes"]){
						g.importsNeeded["basetypes"] = true
					}

					fullname := "*basetypes."+makePublic(replaceReservedWords(el.Name))
					return fullname
					//					return el.Name
				}
			}
		}

		for key, schema := range g.resolvedXsdExternals {
			fullname := ""
			for _, el := range schema.SimpleType {
				if strings.EqualFold(elRef, el.Name) {
					pkg := makePublic(replaceReservedWords(key))
					if(!g.importsNeeded[pkg]){
						g.importsNeeded[pkg] = true
					}
					fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))

					return fullname
				}
			}
			for _, el := range schema.ComplexTypes {
				if strings.EqualFold(elRef, el.Name) {
					pkg := makePublic(replaceReservedWords(key))
					if(!g.importsNeeded[pkg]){
						g.importsNeeded[pkg] = true
					}
					fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
					return fullname
				}			}
			for _, el := range schema.AttributeGoups {
				if strings.EqualFold(elRef, el.Name) {
					pkg := makePublic(replaceReservedWords(key))
					if(!g.importsNeeded[pkg]){
						g.importsNeeded[pkg] = true
					}
					fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
					return fullname
				}
			}

			for _, el := range schema.Elements {
				if strings.EqualFold(elRef, el.Name) {
					elName := ""
					if el.Type != "" {
						elName = stripns(el.Type)
					}else{
						elName = makePublic(replaceReservedWords(el.Name))
					}

					pkg := makePublic(replaceReservedWords(key))
					if(!g.importsNeeded[pkg]){
						g.importsNeeded[pkg] = true
					}
					fullname = "*" +pkg+"."+makePublic(replaceReservedWords(elName))

					return fullname
				}
			}
		}
	}

	if(isBaseType(xmlType)){
		return toGoType(replaceReservedWords(xmlType))
	}else{
		return toGoType(replaceReservedWords(strings.Title(xmlType)))
	}
}
//func (g *GoWsdl) findType(message string) string {
//	message = stripns(message)
//
//	for _, msg := range g.wsdl.Messages {
//		if msg.Name != message {
//			continue
//		}
//
//		// Assumes document/literal wrapped WS-I
//		if len(msg.Parts) == 0 {
//			// Message does not have parts. This could be a Port
//			// with HTTP binding or SOAP 1.2 binding, which are not currently
//			// supported.
//			Log.Warn("WSDL does seem to have HTTP or SOAP 1.2 binding which is not currently supported.")
//			continue
//		}
//		part := msg.Parts[0]
//		if part.Type != "" {
//			return stripns(part.Type)
//		}
//
//		elRef := stripns(part.Element)
//
//		for _, schema := range g.wsdl.Types.Schemas {
//			for _, el := range schema.Elements {
//				if strings.EqualFold(elRef, el.Name) {
//					if el.Type != "" {
//						return stripns(el.Type)
//					}
//					fullname := "basetypes."+makePublic(replaceReservedWords(el.Name))
//					return fullname
////					return el.Name
//				}
//			}
//		}
//
//		for key, schema := range g.resolvedXsdExternals {
//			for _, el := range schema.Elements {
//				if strings.EqualFold(elRef, el.Name) {
//					if el.Type != "" {
//						return stripns(el.Type)
//					}
//
////					pkg := replaceReservedWords(key)
////					namespace := path.Base(schema.TargetNamespace)
////					namespace = replaceReservedWords(namespace)
////					schema.Parent = namespace
////
////					title := strings.Title(el.Name)
////					fullname := namespace+title
//////					return makePublic(replaceReservedWords(fullname))
//					fullname := makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
////					Log.Info(fullname)
//					return fullname
//
////					return el.Name
//				}
//			}
//		}
//	}
//	return ""
//}

// TODO(c4milo): Add support for namespaces instead of striping them out
// TODO(c4milo): improve runtime complexity if performance turns out to be an issue.
func (g *GoWsdl) findSoapAction(operation, portType string) string {
	for _, binding := range g.wsdl.Binding {
		if stripns(binding.Type) != portType {
			continue
		}

		for _, soapOp := range binding.Operations {
			if soapOp.Name == operation {
				return soapOp.SoapOperation.SoapAction
			}
		}
	}
	return ""
}

func (g *GoWsdl) findServiceAddress(name string) string {
	for _, service := range g.wsdl.Service {
		for _, port := range service.Ports {
			if port.Name == name {
				return port.SoapAddress.Location
			}
		}
	}
	return ""
}

// TODO(c4milo): Add namespace support instead of stripping it
func stripns(xsdType string) string {
	r := strings.Split(xsdType, ":")
	type_ := r[0]

	if len(r) == 2 {
		type_ = r[1]
	}

	return type_
}

func makePublic(field_ string) string {
	field := []rune(field_)
	if len(field) == 0 {
		return field_
	}

	field[0] = unicode.ToUpper(field[0])
	return string(field)
}

func comment(text string) string {
	lines := strings.Split(text, "\n")

	var output string
	if len(lines) == 1 && lines[0] == "" {
		return ""
	}

	// Helps to determine if
	// there is an actual comment
	// without screwing newlines
	// in real comments.
	hasComment := false

	for _, line := range lines {
		line = strings.TrimLeftFunc(line, unicode.IsSpace)
		if line != "" {
			hasComment = true
		}
		output += "\n// " + line
	}

	if hasComment {
		return output
	}
	return ""
}

//Check if the maxoccur of the element means that is an array or a single instance
func isArrayElement(maxOccur string) bool {
	if (maxOccur == "unbounded") {
		return true
	}

	i, err := strconv.Atoi(maxOccur)
	if (err != nil) {
		return false
	}

	return i > 1
}

//dictionary map to pass multiple params to a template
func dictValues(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i+=2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}
