// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
package generator

var opsTmpl = `
{{range .}}
	{{$portType := .Name | makePublic}}
	type {{$portType}} struct {
		client *gowsdl.SoapClient
	}

	func New{{$portType}}(url string, tls bool) *{{$portType}} {
		if url == "" {
			url = {{findServiceAddress .Name | printf "%q"}}
		}
		client := gowsdl.NewSoapClient(url, tls)

		return &{{$portType}}{
			client: client,
		}
	}

	{{range .Operations}}
		{{$faults := len .Faults}}
		{{$requestType := findType .Input.Message }}
		{{$soapAction := findSoapAction .Name $portType}}
		{{$responseType := findType .Output.Message }}

		{{/*if ne $soapAction ""*/}}
		{{if gt $faults 0}}
		// Error can be either of the following types:
		// {{range .Faults}}
		//   - {{.Name}} {{.Doc}}{{end}}{{end}}
		{{if ne .Doc ""}}/* {{.Doc}} */{{end}}
		func (service *{{$portType}}) {{makePublic .Name | replaceReservedWords}} ({{if ne $requestType ""}}request {{$requestType}}{{end}}, header *gowsdl.SoapHeader, configureRequest func(*http.Request)) ({{$responseType}}, error) {
			response := &{{replaceStar $responseType}}{}
			err := service.client.Call("{{$soapAction}}", {{if ne $requestType ""}}request{{else}}nil{{end}}, response, header, configureRequest)
			if err != nil {
				return nil, err
			}

			return response, nil
		}
		{{/*end*/}}
	{{end}}
{{end}}
`
