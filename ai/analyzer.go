package ai

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

var tableRegex = regexp.MustCompile(`(?i)\b(?:FROM|INTO|UPDATE|JOIN|TABLE)\s+(\w+)`)

var sqlFuncNames = map[string]bool{
	"Exec":     true,
	"Query":    true,
	"QueryRow": true,
}

var httpMethods = map[string]string{
	"Get":    "GET",
	"Post":   "POST",
	"Put":    "PUT",
	"Patch":  "PATCH",
	"Delete": "DELETE",
}

func Analyze(projectPath string) (*ProjectContext, error) {
	var endpoints []EndpointInfo
	var models []ModelInfo
	tableSet := map[string]bool{}
	topicSet := map[string]bool{}

	err := filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // skip unparseable files
		}

		ast.Inspect(f, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.GenDecl:
				for _, spec := range node.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					info := ModelInfo{Name: ts.Name.Name}
					for _, field := range st.Fields.List {
						typeName := typeString(field.Type)
						if len(field.Names) == 0 {
							info.Fields = append(info.Fields, typeName)
						}
						for _, name := range field.Names {
							info.Fields = append(info.Fields, name.Name+" "+typeName)
						}
					}
					models = append(models, info)
				}

			case *ast.CallExpr:
				sel, ok := node.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				methodName := sel.Sel.Name

				// HTTP endpoints
				if httpMethod, isHTTP := httpMethods[methodName]; isHTTP && len(node.Args) >= 2 {
					if path := stringLit(node.Args[0]); path != "" {
						endpoints = append(endpoints, EndpointInfo{
							Method:  httpMethod,
							Path:    path,
							Handler: handlerName(node.Args[1]),
						})
					}
				}

				// http.HandleFunc
				if methodName == "HandleFunc" && len(node.Args) >= 2 {
					if path := stringLit(node.Args[0]); path != "" {
						endpoints = append(endpoints, EndpointInfo{
							Method:  "ANY",
							Path:    path,
							Handler: handlerName(node.Args[1]),
						})
					}
				}

				// SQL tables
				if sqlFuncNames[methodName] && len(node.Args) >= 1 {
					for _, arg := range node.Args {
						if sql := stringLit(arg); sql != "" {
							for _, match := range tableRegex.FindAllStringSubmatch(sql, -1) {
								if len(match) > 1 {
									tableSet[strings.ToLower(match[1])] = true
								}
							}
							break
						}
					}
				}

				// Kafka topics
				if (methodName == "CreateTopic" || methodName == "Publish") && len(node.Args) >= 1 {
					if topic := stringLit(node.Args[0]); topic != "" {
						topicSet[topic] = true
					}
				}
			}
			return true
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("analyze %s: %w", projectPath, err)
	}

	tables := keys(tableSet)
	topics := keys(topicSet)

	ctx := &ProjectContext{
		Endpoints: endpoints,
		Models:    models,
		Tables:    tables,
		Topics:    topics,
	}
	ctx.Summary = buildSummary(ctx)
	return ctx, nil
}

func stringLit(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	return strings.Trim(lit.Value, "`\"")
}

func handlerName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.FuncLit:
		return "<anonymous>"
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", identName(e.X), e.Sel.Name)
	case *ast.Ident:
		return e.Name
	}
	return "<unknown>"
}

func identName(expr ast.Expr) string {
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return "_"
}

func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.SelectorExpr:
		return identName(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + typeString(t.Key) + "]" + typeString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	}
	return "unknown"
}

func keys(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

func buildSummary(ctx *ProjectContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Project has %d HTTP endpoint(s)", len(ctx.Endpoints)))
	if len(ctx.Tables) > 0 {
		sb.WriteString(fmt.Sprintf(", uses DB tables: %s", strings.Join(ctx.Tables, ", ")))
	}
	if len(ctx.Topics) > 0 {
		sb.WriteString(fmt.Sprintf(", has Kafka topics: %s", strings.Join(ctx.Topics, ", ")))
	}
	sb.WriteString(".")
	return sb.String()
}
