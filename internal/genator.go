package internal

import (
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"regexp"
	"strings"
)

var (
	reName  = regexp.MustCompile(`[A-Z][^A-Z]*`)
	strconv = false
)

const (
	commandName        = `_command`
	valuesVariableName = `values`
)

func Generate(command *Command) error {
	strconv = false
	argumentCountName := `argumentCount`
	file := &ast.File{}
	file.Name = &ast.Ident{
		NamePos: 0,
		Name:    command.Package,
		Obj:     nil,
	}
	importDecl := &ast.GenDecl{
		Tok: token.IMPORT,
		Specs: []ast.Spec{
			&ast.ImportSpec{
				Path: &ast.BasicLit{
					Value: `"fmt"`,
				},
			},
			&ast.ImportSpec{
				Path: &ast.BasicLit{
					Value: `"strings"`,
				},
			},
		},
	}
	forBodyStmt := &ast.BlockStmt{
		List: []ast.Stmt{

		},
	}
	funcBodyStmt := &ast.BlockStmt{
		List: []ast.Stmt{
			&ast.AssignStmt{
				Lhs: []ast.Expr{
					ast.NewIdent(commandName),
				},
				Tok: token.DEFINE,
				Rhs: []ast.Expr{
					&ast.CompositeLit{
						Type: ast.NewIdent(command.Name),
					},
				},
			},
		},
	}
	if len(command.Arguments) > 0 {
		funcBodyStmt.List = append(funcBodyStmt.List, &ast.AssignStmt{
			Lhs: []ast.Expr{
				ast.NewIdent(argumentCountName),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.INT,
					Value: fmt.Sprintf(`%d`, len(command.Arguments)),
				},
			},
		})
	}
	funcBodyStmt.List = append(funcBodyStmt.List, &ast.RangeStmt{
		Key:   ast.NewIdent(`_`),
		Value: ast.NewIdent(`item`),
		Tok:   token.DEFINE,
		X:     ast.NewIdent(`items`),
		Body:  forBodyStmt,
	})
	funcBodyStmt.List = append(funcBodyStmt.List, &ast.ReturnStmt{
		Results: []ast.Expr{
			&ast.UnaryExpr{
				Op: token.AND,
				X:  ast.NewIdent(commandName),
			},
			ast.NewIdent(`nil`),
		},
	})
	file.Decls = []ast.Decl{
		importDecl,
		&ast.FuncDecl{
			Name: &ast.Ident{
				Name: `New` + strings.Title(command.Name),
			},
			Type: &ast.FuncType{
				Params: &ast.FieldList{
					List: []*ast.Field{
						{
							Names: []*ast.Ident{
								{
									Name: `items`,
								},
							},
							Type: &ast.Ellipsis{
								Elt: ast.NewIdent(`string`),
							},
						},
					},
				},
				Results: &ast.FieldList{
					List: []*ast.Field{
						{
							Type: &ast.StarExpr{
								X: ast.NewIdent(command.Name),
							},
						},
						{
							Type: ast.NewIdent(`error`),
						},
					},
				},
			},
			Body: funcBodyStmt,
		},
	}
	if len(command.LongOptions) > 0 {
		caseTag := func(item *Field) string {
			return fmt.Sprintf("`%s`", formatLongOption(item.Name))
		}
		forBodyStmt.List = append(forBodyStmt.List, generateOptionCase("`--`", `long`, command.LongOptions, caseTag))
	}
	if len(command.ShortOptions) > 0 {
		caseTag := func(item *Field) string {
			return fmt.Sprintf("`%s`", item.Short)
		}
		ifStmt := generateOptionCase("`-`", `short`, command.ShortOptions, caseTag)
		if len(forBodyStmt.List) == 0 {
			forBodyStmt.List = append(forBodyStmt.List, ifStmt)
		} else {
			//TODO: Сделать проверки что бы избежать паники
			forBodyStmt.List[len(forBodyStmt.List)-1].(*ast.IfStmt).Else = ifStmt
		}
	}
	if len(command.Arguments) > 0 {
		body := &ast.BlockStmt{
			List: make([]ast.Stmt, len(command.Arguments)+1),
		}
		block := &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ForStmt{
					Cond: &ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent(`strings`),
							Sel: ast.NewIdent(`HasPrefix`),
						},
						Args: []ast.Expr{
							ast.NewIdent(`item`),
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: "`\\`",
							},
						},
					},
					Body: &ast.BlockStmt{
						List: []ast.Stmt{
							&ast.AssignStmt{
								Lhs: []ast.Expr{ast.NewIdent(`item`),},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{
									&ast.SliceExpr{
										X: ast.NewIdent(`item`),
										Low: &ast.BasicLit{
											Kind:  token.INT,
											Value: `1`,
										},
									},
								},
							},
						},
					},
				},
				&ast.SwitchStmt{
					Tag:  ast.NewIdent(argumentCountName),
					Body: body,
				},
			},
		}
		if len(forBodyStmt.List) == 0 {
			forBodyStmt.List = append(forBodyStmt.List, block)
		} else if err := appendToEndElse(forBodyStmt.List[len(forBodyStmt.List)-1].(*ast.IfStmt), block); err != nil {
			return err
		}
		for index, item := range command.Arguments {
			body.List[index] = &ast.CaseClause{
				List: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.INT,
						Value: fmt.Sprintf(`%d`, len(command.Arguments)-index),
					},
				},
				Body: append(generateFormatVariable(item, ast.NewIdent(`item`)), &ast.IncDecStmt{
					X:      ast.NewIdent(argumentCountName),
					Tok:    token.DEC,
				}),
			}
		}
		body.List[len(body.List)-1] = &ast.CaseClause{
			Body: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						ast.NewIdent(`nil`),
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent(`fmt`),
								Sel: ast.NewIdent(`Errorf`),
							},
							Args: []ast.Expr{
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: "`wrong argument '%s'`",
								},
								ast.NewIdent(`item`),
							},
						},
					},
				},
			},
		}
	}
	if strconv {
		importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
			Path: &ast.BasicLit{
				Value: `"strconv"`,
			},
		})
	}

	return printer.Fprint(command.Writer, command.FileSet, file)
}

func appendToEndElse(stmt *ast.IfStmt, block *ast.BlockStmt) error {
	if stmt.Else == nil {
		stmt.Else = block
		return nil
	}

	if ifStmt, ok := stmt.Else.(*ast.IfStmt); ok {
		return appendToEndElse(ifStmt, block)
	}

	return fmt.Errorf(`samething wrong`)
}

func generateOptionCase(optionPrefix, optionType string, options Fields, caseTag func(item *Field) string) *ast.IfStmt {
	switchBodyStmt := make([]ast.Stmt, len(options)+1)
	var ifStmt = &ast.IfStmt{
		Cond: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   ast.NewIdent(`strings`),
				Sel: ast.NewIdent(`HasPrefix`),
			},
			Args: []ast.Expr{
				ast.NewIdent(`item`),
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: optionPrefix,
				},
			},
			Ellipsis: 0,
			Rparen:   0,
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.AssignStmt{
					Lhs: []ast.Expr{
						ast.NewIdent(valuesVariableName),
					},
					Tok: token.DEFINE,
					Rhs: []ast.Expr{
						&ast.CallExpr{
							Fun: &ast.SelectorExpr{
								X:   ast.NewIdent(`strings`),
								Sel: ast.NewIdent(`SplitN`),
							},
							Args: []ast.Expr{
								&ast.SliceExpr{
									X: ast.NewIdent(`item`),
									Low: &ast.BasicLit{
										Kind:  token.INT,
										Value: fmt.Sprintf(`%d`, len(optionPrefix)-2),
									},
								},
								&ast.BasicLit{
									Kind:  token.STRING,
									Value: "`=`",
								},
								&ast.BasicLit{
									Kind:  token.INT,
									Value: `2`,
								},
							},
							Ellipsis: 0,
							Rparen:   0,
						},
					},
				},
				&ast.SwitchStmt{
					Tag: &ast.IndexExpr{
						X: ast.NewIdent(valuesVariableName),
						Index: &ast.BasicLit{
							Kind:  token.INT,
							Value: `0`,
						},
					},
					Body: &ast.BlockStmt{
						List: switchBodyStmt,
					},
				},
			},
		},
		Else: nil,
	}
	for index, item := range options {
		switchBodyStmt[index] = &ast.CaseClause{
			List: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: caseTag(item),
				},
			},
			Body: generateFormatVariable(item, &ast.IndexExpr{
				X: ast.NewIdent(valuesVariableName),
				Index: &ast.BasicLit{
					Kind:  token.INT,
					Value: `1`,
				},
			}),
		}
	}
	switchBodyStmt[len(options)] = &ast.CaseClause{
		Body: []ast.Stmt{
			&ast.ReturnStmt{
				Results: []ast.Expr{
					ast.NewIdent(`nil`),
					&ast.CallExpr{
						Fun: &ast.SelectorExpr{
							X:   ast.NewIdent(`fmt`),
							Sel: ast.NewIdent(`Errorf`),
						},
						Args: []ast.Expr{
							&ast.BasicLit{
								Kind:  token.STRING,
								Value: fmt.Sprintf("`wrong %s option '%%s' from '%%s'`", optionType),
							},
							&ast.IndexExpr{
								X: ast.NewIdent(valuesVariableName),
								Index: &ast.BasicLit{
									Kind:  token.INT,
									Value: `0`,
								},
							},
							ast.NewIdent(`item`),
						},
					},
				},
			},
		},
	}
	return ifStmt
}

func generateFormatVariable(item *Field, value ast.Expr) []ast.Stmt {
	var body []ast.Stmt
	switch item.VariableType {
	case VariableString:
		body = make([]ast.Stmt, 1)
		body[0] = &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X:   ast.NewIdent(commandName),
					Sel: ast.NewIdent(item.Name),
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				value,
			},
		}
	default:
		strconv = true
		body = make([]ast.Stmt, 3)
		strConvFunctionIdent := ast.NewIdent(``)
		assingFuncNameIdent := ast.NewIdent(``)
		body[0] = &ast.AssignStmt{
			Lhs: []ast.Expr{
				ast.NewIdent(`value`),
				ast.NewIdent(`err`),
			},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   ast.NewIdent(`strconv`),
						Sel: strConvFunctionIdent,
					},
					Args: []ast.Expr{
						value,
					},
				},
			},
		}
		body[1] = &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  ast.NewIdent(`err`),
				Op: token.NEQ,
				Y:  ast.NewIdent(`nil`),
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.ReturnStmt{
						Results: []ast.Expr{
							ast.NewIdent(`nil`),
							ast.NewIdent(`err`),
						},
					},
				},
			},
		}
		body[2] = &ast.AssignStmt{
			Lhs: []ast.Expr{
				&ast.SelectorExpr{
					X:   ast.NewIdent(commandName),
					Sel: ast.NewIdent(item.Name),
				},
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{
				&ast.CallExpr{
					Fun: assingFuncNameIdent,
					Args: []ast.Expr{
						ast.NewIdent(`value`),
					},
				},
			},
		}
		switch item.VariableType {
		case VariableBool:
			strConvFunctionIdent.Name = `ParseBool`
			body[2].(*ast.AssignStmt).Rhs[0] = ast.NewIdent(`value`)
		case VariableInt, VariableInt8, VariableInt16, VariableInt32, VariableInt64:
			strConvFunctionIdent.Name = `ParseInt`
			callExpr := body[0].(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)
			callExpr.Args = append(callExpr.Args, &ast.BasicLit{
				Kind:  token.INT,
				Value: `10`,
			})
			switch item.VariableType {
			case VariableInt:
				assingFuncNameIdent.Name = `int`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `64`,
				})
			case VariableInt8:
				assingFuncNameIdent.Name = `int8`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `8`,
				})
			case VariableInt16:
				assingFuncNameIdent.Name = `int16`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `16`,
				})
			case VariableInt32:
				assingFuncNameIdent.Name = `int32`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `32`,
				})
			case VariableInt64:
				assingFuncNameIdent.Name = `int64`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `64`,
				})
			}
		case VariableUint, VariableUint8, VariableUint16, VariableUint32, VariableUint64:
			strConvFunctionIdent.Name = `ParseUint`
			callExpr := body[0].(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)
			callExpr.Args = append(callExpr.Args, &ast.BasicLit{
				Kind:  token.INT,
				Value: `10`,
			})
			switch item.VariableType {
			case VariableUint:
				assingFuncNameIdent.Name = `uint`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `64`,
				})
			case VariableUint8:
				assingFuncNameIdent.Name = `uint8`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `8`,
				})
			case VariableUint16:
				assingFuncNameIdent.Name = `uint16`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `16`,
				})
			case VariableUint32:
				assingFuncNameIdent.Name = `uint32`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `32`,
				})
			case VariableUint64:
				assingFuncNameIdent.Name = `uint64`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `64`,
				})
			}
		case VariableFloat32, VariableFloat64:
			strConvFunctionIdent.Name = `ParseFloat`
			callExpr := body[0].(*ast.AssignStmt).Rhs[0].(*ast.CallExpr)
			switch item.VariableType {
			case VariableFloat32:
				assingFuncNameIdent.Name = `float32`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `32`,
				})
			case VariableFloat64:
				assingFuncNameIdent.Name = `float64`
				callExpr.Args = append(callExpr.Args, &ast.BasicLit{
					Kind:  token.INT,
					Value: `64`,
				})
			}
		}
	}
	return body
}

func formatLongOption(name string) string {
	res := ``
	for index, item := range reName.FindAllString(name, -1) {
		if index > 0 {
			res += `-`
		}
		res += strings.ToLower(item)
	}
	return res
}
