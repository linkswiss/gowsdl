// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

import (
	"bytes"
//	"crypto/tls"
	"encoding/xml"
//	"errors"
//	"fmt"
	"io/ioutil"
//	"net"
//	"net/http"
	"net/url"
	"os"
//	"path/filepath"
	"strings"
	"sync"
	"text/template"
//	"time"
//	"unicode"
//	"strconv"
	"github.com/davecgh/go-spew/spew"
)

//const maxRecursion uint8 = 20

type GoXsd struct {
	file,pkg              string
	ignoreTls             bool
	xsd                   *XsdSchema
	resolvedXsdExternals  map[string]*XsdSchema
	importsNeeded		  map[string]bool
	currentRecursionLevel uint8
	processedComplexTypes map[string]map[string]bool
	processedSimpleTypes  map[string]map[string]bool
	packagesTypes 	  	  map[string]map[string]bool
	currentSchema	      *XsdSchema
}

func NewGoXsd(file, pkg string, ignoreTls bool) (*GoXsd, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		Log.Crit("XSD file is required to generate Go classes")
		os.Exit(2)
	}

	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		pkg = "myservice"
	}

	return &GoXsd{
		file:      file,
		pkg:       pkg,
		ignoreTls: ignoreTls,
	}, nil
}

func (g *GoXsd) Start() (map[string][]byte, error) {
	var gotypes map[string][]byte

	err := g.unmarshal()
	if err != nil {
		return nil, err
	}

	g.fillPackagesTypes()

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

	wg.Wait()

	return gotypes, nil
}

func (g *GoXsd) unmarshal() error {
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

	g.xsd = &XsdSchema{}
	err = xml.Unmarshal(data, g.xsd)
	if err != nil {
		return err
	}

//	spew.Dump(g.wsdl.Types.Schemas)

//	for _, schema := range g.xsd.Types.Schemas {
		err = g.resolveXsdExternals(g.xsd, parsedUrl)
		if err != nil {
			return err
		}
//	}

	return nil
}

func (g *GoXsd) resolveXsdExternals(schema *XsdSchema, url *url.URL) error {
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

//		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, newschema)

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
		var data []byte

		if !location.IsAbs() && !url.IsAbs() {
			if !url.IsAbs() {
//				Log.Info("Open local external schema", "location", schemaLocation)
				data, err = ioutil.ReadFile(location.Path)
				if err != nil {
					return err
				}
			}
		}else{
			if !location.IsAbs(){
				schemaLocation = url.Scheme + "://" + url.Host + schemaLocation
			}
//			Log.Info("Downloading external schema", "location", schemaLocation)
			data, err = downloadFile(schemaLocation, g.ignoreTls)
		}

		newschema := &XsdSchema{}

		err = xml.Unmarshal(data, newschema)
		if err != nil {
			return err
		}

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

//		g.wsdl.Types.Schemas = append(g.wsdl.Types.Schemas, newschema)
	}

//	keys := make([]string, 0, len(g.resolvedXsdExternals))
//	for k := range g.resolvedXsdExternals {
//		keys = append(keys, k)
//	}
//	spew.Dump(keys)

	return nil
}

func (g *GoXsd) addPackageType(elType string) {
	if g.packagesTypes[g.currentSchema.Parent] != nil && g.packagesTypes[g.currentSchema.Parent][elType] {
		return
	}else {
		if g.packagesTypes[g.currentSchema.Parent] == nil{
			g.packagesTypes[g.currentSchema.Parent] = make(map[string]bool, 10000)
		}
		g.packagesTypes[g.currentSchema.Parent][elType] = true
	}
}

func (g *GoXsd) fillComplexTypesLocal(elementType *XsdElement) {
	elType := makePublic(replaceReservedWords(elementType.Name))
	g.addPackageType(elType)

	if(elementType.ComplexType != nil){
		if(elementType.ComplexType.ComplexContent.Extension.Base != ""){
			for _, el := range elementType.ComplexType.ComplexContent.Extension.Sequence {
				if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
					g.fillComplexTypesLocal(&el)
				}
			}
		}else if(elementType.ComplexType.SimpleContent.Extension.Base != ""){
			for _, el := range elementType.ComplexType.SimpleContent.Extension.Sequence {
				if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
					g.fillComplexTypesLocal(&el)
				}
			}
		}else{
			for _, el := range elementType.ComplexType.Sequence {
				if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
					g.fillComplexTypesLocal(&el)
				}
			}
			for _, el := range elementType.ComplexType.SubSequence {
				if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
					g.fillComplexTypesLocal(&el)
				}
			}
			for _, el := range elementType.ComplexType.Choice {
				if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
					g.fillComplexTypesLocal(&el)
				}
			}
			for _, el := range elementType.ComplexType.All {
				if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
					g.fillComplexTypesLocal(&el)
				}
			}
		}
	}
	return
}

func (g *GoXsd) fillComplexTypesGlobal(complexType *XsdComplexType) {
	elType := makePublic(replaceReservedWords(complexType.Name))
	g.addPackageType(elType)

	if(complexType.ComplexContent.Extension.Base != ""){
		for _, el := range complexType.ComplexContent.Extension.Sequence {
			if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
				g.fillComplexTypesLocal(&el)
			}
		}
	}else if(complexType.SimpleContent.Extension.Base != ""){
		for _, el := range complexType.SimpleContent.Extension.Sequence {
			if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
				g.fillComplexTypesLocal(&el)
			}
		}
	}else{
		for _, el := range complexType.Sequence {
			if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
				g.fillComplexTypesLocal(&el)
			}
		}
		for _, el := range complexType.SubSequence {
			if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
				g.fillComplexTypesLocal(&el)
			}
		}
		for _, el := range complexType.Choice {
			if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
				g.fillComplexTypesLocal(&el)
			}
		}
		for _, el := range complexType.All {
			if(el.Ref == "" && el.SimpleType == nil && el.Type == ""){
				g.fillComplexTypesLocal(&el)
			}
		}
	}
	return
}

func (g *GoXsd) fillSchemaTypes(schema *XsdSchema) {
	for _, el := range schema.SimpleType {
		elType := makePublic(replaceReservedWords(el.Name))
		g.addPackageType(elType)
	}

	for _, el := range schema.ComplexTypes {
		g.fillComplexTypesGlobal(el)
	}

	for _, el := range schema.Elements {
		if (el.Type == "") {
			g.fillComplexTypesLocal(el)
		}else {
			elType := makePublic(replaceReservedWords(strings.Title(el.Name)))
			g.addPackageType(elType)
		}
	}

	for _, el := range schema.AttributeGoups {
		if (el.Ref == "") {
			elType := makePublic(replaceReservedWords(el.Name))
			g.addPackageType(elType)
		}
	}
	return
}


func (g *GoXsd) fillPackagesTypes() {

	schemaName := getSchemaName(g.file)
	g.xsd.Parent = schemaName
	g.setCurrentSchema(g.xsd)

	g.packagesTypes = make(map[string]map[string]bool, 10000)

	g.fillSchemaTypes(g.xsd)

	for key, schema := range g.resolvedXsdExternals {
		schema.Parent = key
		g.setCurrentSchema(schema)
		g.fillSchemaTypes(schema)
	}

//	spew.Dump(g.packagesTypes["OTA_SimpleTypes"])
	return
}


//Generate types, included and imported schemas are under it's own namespaces, others under basetypes
func (g *GoXsd) genTypes() (map[string][]byte, error) {
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
		"title":				strings.Title,
		"isArrayElement": 		isArrayElement,
		"dictValues":			dictValues,
		"setCurrentSchema":     g.setCurrentSchema,
		"targetNamespace":      g.targetNamspace,
		"getSchemaName":		getSchemaName,
		"dump":					dump,
//		"targetNamespace":      func() string { return g.wsdl.TargetNamespace },
	}

	gotypes := make(map[string][]byte)
	g.importsNeeded = make(map[string]bool,100)

	schemaName := getSchemaName(g.file)
	g.xsd.Parent = schemaName

//	for _, schema := range g.wsdl.Types.Schemas {
		data := new(bytes.Buffer)
		tmpl := template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
		err := tmpl.Execute(data, g.xsd)
		if err != nil {
			return nil, err
		}

		schemaBytes := bytes.TrimSpace(data.Bytes())
		if(len(schemaBytes) > 0){
			gotypes[schemaName] = append(gotypes[schemaName],data.Bytes()...)
		}
//	}

	pkg := replaceReservedWords(schemaName)
	headerElem := HeaderElements{
		Pkg: pkg,
		PkgBase: g.pkg,
		ImportsNeeded: g.importsNeeded,
	}

	headerData := new(bytes.Buffer)
	tmpl = template.Must(template.New("includetHeader").Funcs(funcMap).Parse(includeHeaderTmpl))
	err  = tmpl.Execute(headerData, headerElem)
	if err != nil {
		return nil, err
	}

	content := gotypes[schemaName]
	gotypes[schemaName] = headerData.Bytes()
	gotypes[schemaName] = append(gotypes[schemaName],content...)

//	Log.Info("Base")
//	spew.Dump(g.importsNeeded)

	for key, schema := range g.resolvedXsdExternals {
		name := key
		schema.Parent = key
		g.importsNeeded = make(map[string]bool,100)

		data = new(bytes.Buffer)
		tmpl = template.Must(template.New("types").Funcs(funcMap).Parse(typesTmpl))
		err = tmpl.Execute(data, schema)
		if err != nil {
			return nil, err
		}

		schemaBytes := bytes.TrimSpace(data.Bytes())
		if(len(schemaBytes) > 0){
			gotypes[name] = append(gotypes[name],data.Bytes()...)
		}

		pkg := replaceReservedWords(name)
		headerElem := HeaderElements{
			Pkg: pkg,
			PkgBase: g.pkg,
			ImportsNeeded: g.importsNeeded,
		}

		headerData = new(bytes.Buffer)
		tmplhead := template.Must(template.New("includetHeader").Funcs(funcMap).Parse(includeHeaderTmpl))
		err := tmplhead.Execute(headerData, headerElem)
		if err != nil {
			return nil, err
		}

		content := gotypes[name]
		gotypes[name] = headerData.Bytes()
		gotypes[name] = append(gotypes[name],content...)

		//		gotypes[name] = append(headerData.Bytes(),gotypes[name])
//		gotypes[name] = append([]byte{headerData.Bytes()},gotypes[name]...)
	}

	return gotypes, nil
}

func dump(obj interface {}) string{
	spew.Dump(obj)
	return "dumped"
}

// Check if the ComplexType is already been processed
func (g *GoXsd) processComplexType(complexType string) bool {
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
func (g *GoXsd) processSimpleType(simpleType string) bool {
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
func (g *GoXsd) setCurrentSchema(schema *XsdSchema) string {
	g.currentSchema = schema
	return ""
}

// Check if the SimpleType is already been processed
func (g *GoXsd) targetNamspace() string {
	if(g.currentSchema != nil && g.currentSchema.TargetNamespace != ""){
		return g.currentSchema.TargetNamespace
	}else{
		return g.xsd.TargetNamespace
	}
}

// Given a message, finds its type.
//
// I'm not very proud of this function but
// it works for now and performance doesn't
// seem critical at this point
func (g *GoXsd) findType(xmlType string) string {
	elRef := makePublic(replaceReservedWords(stripns(xmlType)))

//	Log.Info(elRef)
//	spew.Dump(g.processedComplexTypes);

//	schemaName := getSchemaName(g.file)
//	Log.Info(schemaName)

	if(isBaseType(xmlType)){
//		if(xmlType == "RPH_Type"){
//			Log.Info("BASETYPE")
//		}
		return toGoType(replaceReservedWords(xmlType))
	}

	for keyType, _ := range g.packagesTypes[g.currentSchema.Parent] {
		if(elRef == keyType){
//			if(xmlType == "RPH_Type"){
//				Log.Info("INNER")
//			}
			//Log.Info("FOUND INNER TYPE "+elRef)
			fullname := "*"+makePublic(replaceReservedWords(elRef))
			return fullname
		}
	}


	for keyPkg, elPkg := range g.packagesTypes {
		pkg := replaceReservedWords(keyPkg)
		for keyType, _ := range elPkg {
			if(elRef == keyType){
//				if(xmlType == "RPH_Type"){
//					Log.Info("PKG")
//				}
				//Log.Info("FOUND TYPE "+pkg+"."+elRef)
				if(!g.importsNeeded[pkg]){
					g.importsNeeded[pkg] = true
				}
				fullname := "*"+pkg+"."+makePublic(replaceReservedWords(elRef))
				return fullname
			}
		}
	}

//	if(xmlType == "RPH_Type"){
//		Log.Info("NONE")
//	}

	return toGoType(replaceReservedWords(strings.Title(xmlType)))


//	for _, el := range g.xsd.SimpleType {
//		if strings.EqualFold(elRef, el.Name) {
//			schemaName := getSchemaName(g.file)
//			g.xsd.Parent = schemaName
//			pkg := replaceReservedWords(schemaName)
//
//			if(!g.importsNeeded[pkg]){
//				g.importsNeeded[pkg] = true
//			}
//
//			fullname := "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//			return fullname
//			//					return el.Name
//		}
//	}
//	for _, el := range g.xsd.ComplexTypes {
//		if strings.EqualFold(elRef, el.Name) {
//			schemaName := getSchemaName(g.file)
//			g.xsd.Parent = schemaName
//			pkg := replaceReservedWords(schemaName)
//
//			if(!g.importsNeeded[pkg]){
//				g.importsNeeded[pkg] = true
//			}
//
//			fullname := "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//			return fullname
//			//					return el.Name
//		}
//	}
//	for _, el := range g.xsd.AttributeGoups {
//		if strings.EqualFold(elRef, el.Name) {
//			schemaName := getSchemaName(g.file)
//			g.xsd.Parent = schemaName
//			pkg := replaceReservedWords(schemaName)
//
//			if(!g.importsNeeded[pkg]){
//				g.importsNeeded[pkg] = true
//			}
//
//			fullname := "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//			return fullname
//			//					return el.Name
//		}
//	}
//		for _, el := range g.xsd.Elements {
//			if strings.EqualFold(elRef, el.Name) {
//				if el.Type != "" {
//					return "*" +stripns(el.Type)
//				}
//				schemaName := getSchemaName(g.file)
//				g.xsd.Parent = schemaName
//				pkg := replaceReservedWords(schemaName)
//
//				if(!g.importsNeeded[pkg]){
//					g.importsNeeded[pkg] = true
//				}
//
//				fullname := "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//				return fullname
//				//					return el.Name
//			}
//		}
//
//		for key, schema := range g.resolvedXsdExternals {
//			fullname := ""
//			for _, el := range schema.SimpleType {
//				if strings.EqualFold(elRef, el.Name) {
//					if(key == g.currentSchema.Parent){
//						fullname = "*" +makePublic(replaceReservedWords(el.Name))
////						fullname = "*" +makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
//					}else{
//						pkg := makePublic(replaceReservedWords(key))
//						if(!g.importsNeeded[pkg]){
//							g.importsNeeded[pkg] = true
//						}
//						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//					}
//					return fullname
//				}
//			}
//			for _, el := range schema.ComplexTypes {
//				if strings.EqualFold(elRef, el.Name) {
//					if(key == g.currentSchema.Parent){
//						fullname = "*" +makePublic(replaceReservedWords(el.Name))
////						fullname = "*" +makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
//					}else{
//						pkg := makePublic(replaceReservedWords(key))
//						if(!g.importsNeeded[pkg]){
//							g.importsNeeded[pkg] = true
//						}
//						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//					}
//					return fullname
//				}			}
//			for _, el := range schema.AttributeGoups {
//				if strings.EqualFold(elRef, el.Name) {
//					if(key == g.currentSchema.Parent){
//						fullname = "*" +makePublic(replaceReservedWords(el.Name))
////						fullname = "*" +makePublic(replaceReservedWords(key))+"."+makePublic(replaceReservedWords(el.Name))
//					}else{
//						pkg := makePublic(replaceReservedWords(key))
//						if(!g.importsNeeded[pkg]){
//							g.importsNeeded[pkg] = true
//						}
//						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(el.Name))
//					}
//					return fullname
//				}
//			}
//
//			for _, el := range schema.Elements {
//				if strings.EqualFold(elRef, el.Name) {
//					elName := ""
//					if el.Type != "" {
//						elName = stripns(el.Type)
//					}else{
//						elName = makePublic(replaceReservedWords(el.Name))
//					}
//
//					if(key == g.currentSchema.Parent){
//						fullname = "*" +makePublic(replaceReservedWords(elName))
////						fullname = "*" +makePublic(replaceReservedWords(key))+"."+elName
//					}else{
//						pkg := makePublic(replaceReservedWords(key))
//						if(!g.importsNeeded[pkg]){
//							g.importsNeeded[pkg] = true
//						}
//						fullname = "*" +pkg+"."+makePublic(replaceReservedWords(elName))
//					}
//					//					Log.Info(fullname)
//					return fullname
//				}
//			}
//		}

//	if(isBaseType(xmlType)){
//		return toGoType(replaceReservedWords(xmlType))
//	}else{
//		return toGoType(replaceReservedWords(strings.Title(xmlType)))
//	}
}
