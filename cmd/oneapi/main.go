package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/danhtran94/oneapi/pkg/oapi"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

func main() {
	path := flag.String("path", "./models/models.go", "path to the models file")
	flag.Parse()

	schemas, err := oapi.GenerateSchemas(*path)
	if err != nil {
		log.Fatalln(fmt.Errorf("[oapi] generate error: %w", err))
	}

	doc := &v3.Document{
		Version: "3.1.0",
		Info: &base.Info{
			Version: "1.0.0",
			Title:   "OneAPI",
			Contact: &base.Contact{
				Name:  "danhtran94",
				Email: "danh.tt1294@gmail.com",
			},
		},
		Servers: []*v3.Server{
			{
				URL: "http://localhost:3000",
			},
		},
		Components: &v3.Components{
			Schemas: schemas,
		},
	}

	rawdoc, err := doc.Render()
	if err != nil {
		log.Fatalln(fmt.Errorf("[oapi] render error: %w", err))
	}

	fmt.Println(string(rawdoc))
}
