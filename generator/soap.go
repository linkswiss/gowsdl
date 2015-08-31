// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"gopkg.in/inconshreveable/log15.v2"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
)

var _ os.File

var Log = log15.New()

func init() {
	Log.SetHandler(log15.DiscardHandler())
//	handler := log15.StreamHandler(os.Stdout, log15.LogfmtFormat())
//	Log.SetHandler(handler)
}

type SoapEnvelope struct {
	XMLName xml.Name    `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  *SoapHeader `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header,omitempty"`
	Body    SoapBody    `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

type SoapHeader struct {
	ReqHeader interface{}
	RespHeader interface{}
	BodyAttributes interface{}
	Content string     `xml:",innerxml"`
}

type SoapBody struct {
	Fault   *SoapFault `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty"`
	Content string     `xml:",innerxml"`

	RequestId string `xml:"RequestId,attr,omitempty"`
	Transaction string `xml:"Transaction,attr,omitempty"`
}

type SoapFault struct {
	Faultcode   string `xml:"faultcode,omitempty"`
	Faultstring string `xml:"faultstring,omitempty"`
	Faultactor  string `xml:"faultactor,omitempty"`
	Detail      string `xml:"detail,omitempty"`
}

type SoapClient struct {
	url string
	tls bool
}

func (f *SoapFault) Error() string {
	return f.Faultstring
}

func NewSoapClient(url string, tls bool) *SoapClient {
	return &SoapClient{
		url: url,
		tls: tls,
	}
}

func (s *SoapClient) Call(soapAction string, request, response interface{}, header *SoapHeader, configureRequest func(*http.Request)) error {
	envelope := SoapEnvelope{
		//Header: header,
	}

	//TODO VERIFY HEADER AND BODY ATTRIBUTES!
//	envelope.Body.RequestId = "TEST"
//	envelope.Body.Transaction = "asdfasdf"

	if(header.BodyAttributes != nil){
//		bodyAttributes := header.BodyAttributes




		header.BodyAttributes = nil
	}

	envelope.Header = header

	if request != nil {
		reqXml, err := xml.Marshal(request)
		if err != nil {
			return err
		}

		envelope.Body.Content = string(reqXml)
//		envelope.Body.Attributes = bodyAttributes
	}
	buffer := &bytes.Buffer{}

	encoder := xml.NewEncoder(buffer)
	encoder.Indent("  ", "    ")

	err := encoder.Encode(envelope)
	if err == nil {
		err = encoder.Flush()
	}
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", s.url, buffer)

	req.Header.Add("Content-Type", "text/xml; charset=\"utf-8\"")
	if soapAction != "" {
		req.Header.Add("SOAPAction", soapAction)
	}

	if configureRequest != nil {
		configureRequest(req)
	}

	Log.Debug("request", "request", req,
		"Header", log15.Lazy{func() string { r, _ := httputil.DumpRequestOut(req, true); return string(r) }},
	)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: s.tls,
		},
		Dial: dialTimeout,
	}

	client := &http.Client{Transport: tr}
	res, err := client.Do(req)
	if err != nil {
		Log.Debug("Client error", "err", err)
		return err
	}
	defer res.Body.Close()

	rawbody, err := ioutil.ReadAll(res.Body)
	if len(rawbody) == 0 {
		Log.Warn("empty response")
		return nil
	}
	Log.Debug("Raw response", "url", s.url, "rawbody", log15.Lazy{func() string { return string(rawbody) }})

	respEnvelope := &SoapEnvelope{}

	err = xml.Unmarshal(rawbody, respEnvelope)
	if err != nil {
		return err
	}

	body := respEnvelope.Body.Content
	fault := respEnvelope.Body.Fault
	if(respEnvelope.Header != nil){
		header.Content = respEnvelope.Header.Content
	}

	if body == "" {
		Log.Warn("empty response body", "envelope", respEnvelope, "body", body)
		return nil
	}

	Log.Debug("response", "envelope", respEnvelope, "body", body)
	if fault != nil {
		return fault
	}

	if(header.Content != ""){
		Log.Debug("Header","content",header.Content)
		err = xml.Unmarshal([]byte(header.Content), header.RespHeader)
		if err != nil {
			return err
		}
		Log.Debug("Header","head",header.RespHeader)
	}

	err = xml.Unmarshal([]byte(body), response)
	if err != nil {
		return err
	}

	return nil
}
