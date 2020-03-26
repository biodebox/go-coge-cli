package founder

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

func NewFounder(path string) (Founder, error) {
	f := founder{}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	f.fileSet = token.NewFileSet()
	flags := parser.AllErrors
	if info.IsDir() {
		if f.packages, err = parser.ParseDir(f.fileSet, path, filterGoFile, flags); err != nil {
			return nil, err
		}
	} else {
		if f.file, err = parser.ParseFile(f.fileSet, path, nil, flags); err != nil {
			return nil, err
		}
	}
	return &f, nil
}

func hasName(name string, names []string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func filterGoFile(info os.FileInfo) bool {
	return !strings.HasSuffix(info.Name(), `_generated.go`) &&
		!strings.HasSuffix(info.Name(), `_test.go`)
}

func foundTypesInFile(file *ast.File, names []string) (res []*ast.TypeSpec, err error) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !hasName(typeSpec.Name.Name, names) {
				continue
			}
			res = append(res, typeSpec)
		}
	}

	return res, nil
}