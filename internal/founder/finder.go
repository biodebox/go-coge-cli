package founder

import (
	"go/ast"
	"go/token"
)

type (
	Founder interface {
		GetTypes(names ...string) ([]*ast.TypeSpec, error)
		GetFunctions(names ...string) ([]*ast.FuncType, error)
		GetMethodsForStruct(structName string, methodsName ...string) ([]*ast.FuncType, error)
		GetPackage() string
		GetFileSet() *token.FileSet
	}
	founder struct {
	file     *ast.File
	packages map[string]*ast.Package
	fileSet  *token.FileSet
}
)

func (f *founder) GetFileSet() *token.FileSet {
	return f.fileSet
}

func (f *founder) GetPackage() string {
	return f.file.Name.Name
}

func (f *founder) GetTypes(names ...string) ([]*ast.TypeSpec, error) {
	if f.file != nil {
		return foundTypesInFile(f.file, names)
	} else {
		return f.findTypesInPackages(names)
	}
}

func (f *founder) GetFunctions(names ...string) ([]*ast.FuncType, error) {
	if f.file != nil {
		return f.findFunctionsInFile(names)
	} else {
		return f.findFunctionsInPackages(names)
	}
}

func (f *founder) findTypesInPackages(names []string) (res []*ast.TypeSpec, err error) {
	for _, p := range f.packages {
		for _, file := range p.Files {
			fr, err := foundTypesInFile(file, names)
			if err != nil {
				return nil, err
			}
			res = append(res, fr...)
		}
	}

	return res, nil
}

func (f *founder) findFunctionsInFile(names []string) ([]*ast.FuncType, error) {
	panic("implement me")
}

func (f *founder) findFunctionsInPackages(names []string) ([]*ast.FuncType, error) {
	panic("implement me")
}

func (f *founder) GetMethodsForStruct(structName string, methodsName ...string) ([]*ast.FuncType, error) {
	panic("implement me")
}
