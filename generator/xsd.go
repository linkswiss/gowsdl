// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

import (
	"encoding/xml"
)

type XsdSchema struct {
	Parent			   string
	XMLName            xml.Name          `xml:"schema"`
	Tns                string            `xml:"xmlns tns,attr"`
	Xs                 string            `xml:"xmlns xs,attr"`
	Version            string            `xml:"version,attr"`
	TargetNamespace    string            `xml:"targetNamespace,attr"`
	ElementFormDefault string            `xml:"elementFormDefault,attr"`
	Includes           []*XsdInclude     `xml:"include"`
	Imports            []*XsdImport      `xml:"import"`
	Elements           []*XsdElement     `xml:"element"`
	ComplexTypes       []*XsdComplexType `xml:"complexType"` //global
	SimpleType         []*XsdSimpleType  `xml:"simpleType"`
	AttributeGoups	   []*XsdAttributeGroup	 `xml:"attributeGroup"`
}

type XsdInclude struct {
	SchemaLocation string `xml:"schemaLocation,attr"`
}

type XsdImport struct {
	XMLName        xml.Name `xml:"import"`
	SchemaLocation string   `xml:"schemaLocation,attr"`
	Namespace      string   `xml:"namespace,attr"`
}

type XsdElement struct {
	XMLName     xml.Name        `xml:"element"`
	Name        string          `xml:"name,attr"`
	Doc         string          `xml:"annotation>documentation"`
	Nillable    bool            `xml:"nillable,attr"`
	Type        string          `xml:"type,attr"`
	Ref         string          `xml:"ref,attr"`
	MinOccurs   string          `xml:"minOccurs,attr"`
	MaxOccurs   string          `xml:"maxOccurs,attr"`
	ComplexType *XsdComplexType `xml:"complexType"` //local
	SimpleType  *XsdSimpleType  `xml:"simpleType"`
	Groups      []*XsdGroup     `xml:"group"`
	Any			[]XsdAny		`xml:"sequence>any"`
}

type XsdAny struct {
	XMLName xml.Name            `xml:"any"`
	ProcessContents    string   `xml:"processContents,attr"`
	MinOccurs   string          `xml:"minOccurs,attr"`
	MaxOccurs   string          `xml:"maxOccurs,attr"`
}

type XsdComplexType struct {
	XMLName        xml.Name          `xml:"complexType"`
	Doc        	   string      		 `xml:"annotation>documentation"`
	Abstract       bool              `xml:"abstract,attr"`
	Name           string            `xml:"name,attr"`
	Mixed          bool              `xml:"mixed,attr"`
	Sequence       []XsdElement      `xml:"sequence>element"`
	SubSequence    []XsdElement      `xml:"sequence>sequence>element"`
	Any			   []XsdAny		     `xml:"sequence>any"`
	Choice         []XsdElement      `xml:"choice>element"`
	All            []XsdElement      `xml:"all>element"`
	ComplexContent XsdComplexContent `xml:"complexContent"`
	SimpleContent  XsdSimpleContent  `xml:"simpleContent"`
	SimpleType     *XsdSimpleType    `xml:"simpleType"`
	Attributes     []*XsdAttribute   `xml:"attribute"`
	AttributeGoups []*XsdAttributeGroup	`xml:"attributeGroup"`
}

type XsdAttributeGroup struct {
	Name       string          `xml:"name,attr"`
	Ref        string          `xml:"ref,attr"`
	Doc        string          `xml:"annotation>documentation"`
	Attributes []*XsdAttribute `xml:"attribute"`
}

type XsdGroup struct {
	Name     string       `xml:"name,attr"`
	Ref      string       `xml:"ref,attr"`
	Sequence []XsdElement `xml:"sequence>element"`
	Choice   []XsdElement `xml:"choice>element"`
	All      []XsdElement `xml:"all>element"`
}

type XsdComplexContent struct {
	XMLName   xml.Name     `xml:"complexContent"`
	Doc        string      `xml:"annotation>documentation"`
	Extension XsdExtension `xml:"extension"`
}

type XsdSimpleContent struct {
	XMLName   xml.Name     `xml:"simpleContent"`
	Extension XsdExtension `xml:"extension"`
}

type XsdExtension struct {
	XMLName    xml.Name        `xml:"extension"`
	Base       string          `xml:"base,attr"`
	Attributes []*XsdAttribute `xml:"attribute"`
	Sequence   []XsdElement    `xml:"sequence>element"`
}

type XsdAttribute struct {
	Name       string         `xml:"name,attr"`
	Doc        string         `xml:"annotation>documentation"`
	Type       string         `xml:"type,attr"`
	SimpleType *XsdSimpleType `xml:"simpleType"`
}

type XsdSimpleType struct {
	Name        string         `xml:"name,attr"`
	Doc         string         `xml:"annotation>documentation"`
	Restriction XsdRestriction `xml:"restriction"`
	UnionType    XsdUnion      `xml:"union"`
}

type XsdRestriction struct {
	Base         string                `xml:"base,attr"`
	Doc          string                `xml:"annotation>documentation"`
	Enumeration  []*XsdRestrictionValue `xml:"enumeration"`
	Pattern      XsdRestrictionValue   `xml:"pattern"`
	MinInclusive XsdRestrictionValue   `xml:"minInclusive"`
	MaxInclusive XsdRestrictionValue   `xml:"maxInclusive"`
	WhiteSpace   XsdRestrictionValue   `xml:"whitespace"`
	Length       XsdRestrictionValue   `xml:"length"`
	MinLength    XsdRestrictionValue   `xml:"minLength"`
	MaxLength    XsdRestrictionValue   `xml:"maxLength"`
}

type XsdRestrictionValue struct {
	Doc   string `xml:"annotation>documentation"`
	Value string `xml:"value,attr"`
}

type XsdUnion struct {
	MemberType string `xml:"memberTypes,attr"`
	SimpleType  []*XsdSimpleType  `xml:"simpleType"`
}
