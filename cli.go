// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"go/format"
	"log"
	"os"
	"runtime"
	"path/filepath"
	"path"
	"strings"

	gen "github.com/hooklift/gowsdl/generator"
	flags "github.com/jessevdk/go-flags"
)

const version = "v0.0.1"

var opts struct {
	Version    bool   `short:"v" long:"version" description:"Shows gowsdl version"`
	Package    string `short:"p" long:"package" description:"Package under which code will be generated - use the path to ensure imports and features under sub packages" default:"myservice"`
	OutputFile string `short:"o" long:"output" description:"File where the generated code will be saved" default:"myservice.go"`
	IgnoreTls  bool   `short:"i" long:"ignore-tls" description:"Ignores invalid TLS certificates. It is not recomended for production. Use at your own risk" default:"false"`
	ProcessXsd bool  `short:"x" long:"process-xsd" description:"Process only xsd. it will process the file as xsd or the folder if specified in is-folder" default:"false"`
	XsdFolder  bool   `short:"f" long:"is-folder" description:"Process only xsd. used by process xsd. It'll go recursively in the folder and process all xsd files" default:"false"`
}

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	log.SetFlags(0)
	log.SetOutput(os.Stdout)
	log.SetPrefix("🍀  ")
}

func main() {
	args, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if opts.Version {
		log.Println(version)
		os.Exit(0)
	}

	if len(args) == 0 {
		log.Fatalln("WSDL or XSD file is required to start the party")
	}

	if opts.OutputFile == args[0] {
		log.Fatalln("Output file cannot be the same as Input file")
	}

	if(opts.ProcessXsd){
		log.Printf("Process XSDs")
		processXSD(opts.Package,opts.IgnoreTls,opts.XsdFolder,args)
	}else{
		log.Printf("Process WSDL")
		processWSDL(opts.Package,opts.IgnoreTls,opts.OutputFile,args)
	}


//	gowsdl, err := gen.NewGoWsdl(args[0], opts.Package, opts.IgnoreTls)
//	if err != nil {
//		log.Fatalln(err)
//	}
//
//	gocode, gotypes, err := gowsdl.Start()
//	if err != nil {
//		log.Fatalln(err)
//	}
//
//	pkg := "./" + opts.Package
//	err = os.Remove(pkg)
//	err = os.Mkdir(pkg, 0744)
//
//	if perr, ok := err.(*os.PathError); ok && os.IsExist(perr.Err) {
//		log.Printf("Package directory %s already exist, skipping creation\n", pkg)
//	} else {
//		if err != nil {
//			log.Fatalln(err)
//		}
//	}
//
//	fd, err := os.Create(pkg + "/" + opts.OutputFile)
//	if err != nil {
//		log.Fatalln(err)
//	}
//	defer fd.Close()
//
//	data := new(bytes.Buffer)
//	data.Write(gocode["header"])
////	data.Write(gocode["types"])
//	data.Write(gocode["operations"])
//
//	source, err := format.Source(data.Bytes())
//	if err != nil {
//		fd.Write(data.Bytes())
//		log.Fatalln(err)
//	}
//
//	fd.Write(source)
//
//	for key, gotype := range gotypes {
//		fd, err := os.Create(pkg + "/" + key + ".go")
//		if err != nil {
//			log.Fatalln(err)
//		}
//		defer fd.Close()
//
//		data := new(bytes.Buffer)
//		data.Write(gotype)
//
//		source, err := format.Source(data.Bytes())
//		if err != nil {
//			fd.Write(data.Bytes())
//			log.Fatalln(err)
//		}
//
//		fd.Write(source)
//	}

	log.Println("Done 💩")
}

func processWSDL(packageOpt string, IgnoreTls bool, outputFile string, args []string){
	gowsdl, err := gen.NewGoWsdl(args[0], packageOpt, IgnoreTls)
	if err != nil {
		log.Fatalln(err)
	}

	gocode, gotypes, err := gowsdl.Start()
	if err != nil {
		log.Fatalln(err)
	}

	pkgName := path.Base(packageOpt)
	pkg := "./" + pkgName

	err = os.Remove(pkg)
	err = os.Mkdir(pkg, 0744)

	if perr, ok := err.(*os.PathError); ok && os.IsExist(perr.Err) {
		log.Printf("Package directory %s already exist, skipping creation\n", pkg)
	} else {
		if err != nil {
			log.Fatalln(err)
		}
	}

	fd, err := os.Create(pkg + "/" + outputFile)
	if err != nil {
		log.Fatalln(err)
	}
	defer fd.Close()

	data := new(bytes.Buffer)
	data.Write(gocode["header"])
	//	data.Write(gocode["types"])
	data.Write(gocode["operations"])

	source, err := format.Source(data.Bytes())
	if err != nil {
		fd.Write(data.Bytes())
		log.Fatalln(err)
	}

	fd.Write(source)

	for key, gotype := range gotypes {
		currPkg := pkg + "/" + strings.Replace(key, "_", "", -1)
		err = os.Mkdir(currPkg, 0744)

		fd, err := os.Create(currPkg + "/" + key + ".go")
		if err != nil {
			log.Fatalln(err)
		}
		defer fd.Close()

		data := new(bytes.Buffer)
		data.Write(gotype)

		source, err := format.Source(data.Bytes())
		if err != nil {
			fd.Write(data.Bytes())
			log.Fatalln(err)
		}

		fd.Write(source)
	}
}

func processXSD(packageOpt string, IgnoreTls bool, xsdFolder bool, args []string){

	pkgName := path.Base(packageOpt)
	pkg := "./" + pkgName

	err := os.Remove(pkg)
	err = os.Mkdir(pkg, 0744)

	if perr, ok := err.(*os.PathError); ok && os.IsExist(perr.Err) {
		log.Printf("Package directory %s already exist, skipping creation\n", pkg)
	} else {
		if err != nil {
			log.Fatalln(err)
		}
	}

	if(xsdFolder){
		fileList := []string{}
		err = filepath.Walk(args[0], func(path string, f os.FileInfo, err error) error {
			fileList = append(fileList, path)
			return nil
		})

		for _, file := range fileList {
			if(filepath.Ext(file) == ".xsd"){
				processSingleXSD(packageOpt,IgnoreTls,file,pkg)
			}
		}
	}else{
		if(filepath.Ext(args[0]) == ".xsd"){
			processSingleXSD(packageOpt,IgnoreTls,args[0],pkg)
		}
	}
}


func processSingleXSD(packageOpt string, IgnoreTls bool, file string, pkg string) {
//	log.Printf("File %s",file)

	goxsd, err := gen.NewGoXsd(file, packageOpt, IgnoreTls)
	if err != nil {
		log.Fatalln(err)
	}

	gotypes, err := goxsd.Start()
	if err != nil {
		log.Fatalln(err)
	}

	for key, gotype := range gotypes {
		currPkg := pkg + "/" + strings.Replace(key, "_", "", -1)
		err = os.Mkdir(currPkg, 0744)

		filename := currPkg + "/" + key + ".go"
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			//output file is missing we can proceed
			fd, err := os.Create(filename)
			if err != nil {
				log.Fatalln(err)
			}
			defer fd.Close()

			data := new(bytes.Buffer)
			data.Write(gotype)

			source, err := format.Source(data.Bytes())
			if err != nil {
				fd.Write(data.Bytes())
				log.Fatalln(err)
			}

			fd.Write(source)
		}else{
//			log.Printf("File %s already created",filename)
		}
	}
}
