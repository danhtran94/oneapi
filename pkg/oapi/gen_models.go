package oapi

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func oapiType(t string) (string, bool) {
	basicTypeToOpenAPIType := map[string]string{
		// basic types
		"string":    "string",
		"int":       "integer",
		"int32":     "integer",
		"int64":     "integer",
		"float32":   "number",
		"float64":   "number",
		"bool":      "boolean",
		"time.Time": "string",
		"[]byte":    "string",
		// opt package
		"null.Val[string]":    "string",
		"null.Val[int]":       "integer",
		"null.Val[int64]":     "integer",
		"null.Val[int32]":     "integer",
		"null.Val[float]":     "number",
		"null.Val[float64]":   "number",
		"null.Val[float32]":   "number",
		"null.Val[bool]":      "boolean",
		"null.Val[time.Time]": "string",
	}

	if openAPIType, ok := basicTypeToOpenAPIType[t]; ok {
		return openAPIType, true
	}

	return "", false
}

func GenerateSchemas(path string) (*orderedmap.Map[string, *base.SchemaProxy], error) {
	if path == "" {
		path = "./models/models.go"
	}

	return globToSchemas(path)
}

func globToSchemas(globPath string) (*orderedmap.Map[string, *base.SchemaProxy], error) {
	schemas := orderedmap.New[string, *base.SchemaProxy]()

	files, err := filepath.Glob(globPath)
	if err != nil {
		return schemas, err
	}

	for _, file := range files {
		fschemas, err := fileToSchemas(file)
		if err != nil {
			return schemas, err
		}

		for pairs := fschemas.First(); pairs != nil; pairs = pairs.Next() {
			schemas.Set(pairs.Key(), pairs.Value())
		}
	}

	return schemas, nil
}

func fileToSchemas(filePath string) (*orderedmap.Map[string, *base.SchemaProxy], error) {
	schemas := orderedmap.New[string, *base.SchemaProxy]()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return schemas, err
	}

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			schemas.Set(typeSpec.Name.Name, exprToProp(structType))
		}
	}

	return schemas, nil
}

func exprToString(expr ast.Expr) string {
	var buf bytes.Buffer
	err := printer.Fprint(&buf, token.NewFileSet(), expr)
	if err != nil {
		panic(err)
	}
	return buf.String()
}

func exprToProp(expr ast.Expr) *base.SchemaProxy {
	switch t := expr.(type) {
	case *ast.Ident:
		// The field is a basic type
		if tp, ok := oapiType(t.Name); ok {
			return base.CreateSchemaProxy(&base.Schema{
				Type:        []string{tp},
				Description: t.Name,
			})
		}

		// The field is a custom type
		return base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", t.Name))

		// TODO: handle struct types recursively
		// switch decl := t.Obj.Decl.(type) {
		// case *ast.TypeSpec:
		// 	if structType, ok := decl.Type.(*ast.StructType); ok {
		// 		p := &OpenAPIProperty{
		// 			Type:       "object",
		// 			GoType:     t.Name,
		// 			Properties: make(map[string]*OpenAPIProperty),
		// 		}

		// 		for _, field := range structType.Fields.List {
		// 			for _, name := range field.Names {
		// 				if prop := exprToProperty(field.Type); prop != nil {
		// 					p.Properties[name.Name] = prop
		// 				}
		// 			}
		// 		}

		// 		return p
		// 	}
		// }
	case *ast.StarExpr:
		// The field is a pointer to another type
		return exprToProp(t.X)
	case *ast.SelectorExpr:
		// The field is a type from another package
		pkg := exprToString(t.X)
		str := fmt.Sprintf("%s.%s", pkg, t.Sel.Name)

		if tp, ok := oapiType(str); ok {
			return base.CreateSchemaProxy(&base.Schema{
				Type:        []string{tp},
				Description: str,
			})
		}
	case *ast.IndexExpr:
		// The field is an generic type or array or slice
		typestr := exprToString(t.X)
		str := fmt.Sprintf("%s[%s]", typestr, t.Index)

		if tp, ok := oapiType(str); ok {
			return base.CreateSchemaProxy(&base.Schema{
				Type:        []string{tp},
				Description: str,
			})
		}
	case *ast.ArrayType:
		// The field is an array or slice
		return base.CreateSchemaProxy(&base.Schema{
			Type: []string{"array"},
			Items: &base.DynamicValue[*base.SchemaProxy, bool]{
				N: 0,
				A: exprToProp(t.Elt),
			},
			Description: fmt.Sprintf("[]%s", exprToString(t.Elt)),
		})
	case *ast.MapType:
		return base.CreateSchemaProxy(&base.Schema{
			Type: []string{"object"},
			AdditionalProperties: &base.DynamicValue[*base.SchemaProxy, bool]{
				N: 0,
				A: exprToProp(t.Value),
			},
			Description: fmt.Sprintf("map[string]%s", exprToString(t.Value)),
		})
	case *ast.StructType:
		propMap := orderedmap.New[string, *base.SchemaProxy]()

		for _, field := range t.Fields.List {
			for _, name := range field.Names {
				if prop := exprToProp(field.Type); prop != nil {
					propMap.Set(name.Name, prop)
				}
			}
		}

		return base.CreateSchemaProxy(&base.Schema{
			Type:        []string{"object"},
			Properties:  propMap,
			Description: "struct",
		})
	}

	return base.CreateSchemaProxy(&base.Schema{
		Type:        []string{"null"},
		Description: "unknown",
	})
}
