package apidoc

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"path/filepath"
	"strings"

	. "github.com/danhtran94/xdot"

	base "github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

type APIDoc struct{}

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
		path = "./models/*.go"
	}

	return globToSchemas(path)
}

var ResponseKinds map[*ast.TypeSpec][]any = map[*ast.TypeSpec][]any{}

func globToSchemas(globPath string) (schemas *orderedmap.Map[string, *base.SchemaProxy], err error) {
	try, pipe := TryPipe()
	defer try(func(_err error) {
		err = _err
	})

	schemas = orderedmap.New[string, *base.SchemaProxy]()
	files := Must(filepath.Glob(globPath))(pipe)

	for _, file := range files {
		fschemas := Must(fileToSchemas(file))(pipe)

		for pairs := fschemas.First(); pairs != nil; pairs = pairs.Next() {
			schemas.Set(pairs.Key(), pairs.Value())
		}
	}

	return schemas, nil
}

func fileToSchemas(filePath string) (schemas *orderedmap.Map[string, *base.SchemaProxy], err error) {
	try, pipe := TryPipe()
	defer try(func(_err error) {
		err = _err
	})

	schemas = orderedmap.New[string, *base.SchemaProxy]()

	fset := token.NewFileSet()
	f := Must(parser.ParseFile(fset, filePath, nil, parser.ParseComments))(pipe)

	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok != token.TYPE {
			continue
		}

		comment := genDecl.Doc.Text()
		oapiDef := getOAPIDef(comment)
		isResponseKind := oapiDef.Get(DefKind) == KindResponse

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			if isResponseKind {
				ResponseKinds[typeSpec] = []any{structType, oapiDef}
				continue
			}

			schemas.Set(typeSpec.Name.Name, exprToProp(structType))
		}
	}

	modelNames := []string{}
	for pairs := schemas.First(); pairs != nil; pairs = pairs.Next() {
		modelNames = append(modelNames, pairs.Key())
	}

	for _, vals := range ResponseKinds {
		structType, oapiDef := vals[0].(*ast.StructType), vals[1].(OAPIDef)

		for _, model := range modelNames {
			placeholder := oapiDef.Get(ResponsePlaceholder)
			name := oapiDef.Get(ResponseName)

			prop := exprToProp(structType, map[string]string{placeholder: model})
			schemaName := fmt.Sprintf(name, model)
			schemas.Set(schemaName, prop)
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

func exprToProp(expr ast.Expr, genericSchemas ...map[string]string) *base.SchemaProxy {
	switch t := expr.(type) {
	case *ast.Ident:
		// The field is a basic type
		if tp, ok := oapiType(t.Name); ok {
			return base.CreateSchemaProxy(&base.Schema{
				Type:        []string{tp},
				Description: t.Name,
			})
		}

		if len(genericSchemas) > 0 {
			if tp, ok := genericSchemas[0][t.Name]; ok {
				return base.CreateSchemaProxyRef(fmt.Sprintf("#/components/schemas/%s", tp))
			}
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
		return exprToProp(t.X, genericSchemas...)
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
				A: exprToProp(t.Elt, genericSchemas...),
			},
			Description: fmt.Sprintf("[]%s", exprToString(t.Elt)),
		})
	case *ast.MapType:
		return base.CreateSchemaProxy(&base.Schema{
			Type: []string{"object"},
			AdditionalProperties: &base.DynamicValue[*base.SchemaProxy, bool]{
				N: 0,
				A: exprToProp(t.Value, genericSchemas...),
			},
			Description: fmt.Sprintf("map[string]%s", exprToString(t.Value)),
		})
	case *ast.StructType:
		propMap := orderedmap.New[string, *base.SchemaProxy]()
		jsonNameByFields := map[string]string{}

		// embedstructs := []string{}
		// read embed struct
		// for _, field := range t.Fields.List {
		// 	if len(field.Names) == 0 {
		// 		if sel, ok := field.Type.(*ast.SelectorExpr); ok {
		// 			embedstructs = append(embedstructs, exprToString(sel))
		// 			if field.Tag != nil {
		// 				oapiTag := field.Tag.Value
		// 				fmt.Println(oapiTag)
		// 			}
		// 		}
		// 	}
		// }

		// read field json tag value
		for _, field := range t.Fields.List {
			for _, name := range field.Names {
				if field.Tag != nil {
					tag := field.Tag.Value
					slices := strings.Split(tag, `json:"`)
					if len(slices) > 1 {
						jsonTag := strings.Split(slices[1], `"`)
						jsonName := strings.ReplaceAll(jsonTag[0], "omitempty", "")
						jsonName = strings.ReplaceAll(jsonName, ",", "")
						jsonName = strings.ReplaceAll(jsonName, " ", "")
						jsonNameByFields[name.Name] = jsonName
					}
				}
			}
		}

		for _, field := range t.Fields.List {
			for _, name := range field.Names {
				if prop := exprToProp(field.Type, genericSchemas...); prop != nil {
					jsonName := name.Name
					if jn, ok := jsonNameByFields[name.Name]; ok {
						jsonName = jn
					}

					propMap.Set(jsonName, prop)
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
