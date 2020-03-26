package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"reflect"
	"strings"
)

type (
	FieldType    int8
	VariableType int8
	Field        struct {
		Name         string
		Short        string
		VariableType VariableType
		Type         FieldType
		Default      string
	}
	Fields  []*Field
	Command struct {
		Package, Name   string
		FileSet *token.FileSet
		Writer io.Writer
		ShortOptions Fields
		LongOptions Fields
		Arguments Fields
	}
	Commands []*Command
)

const (
	FieldOption FieldType = iota + 1
	FieldArgument
)

const (
	VariableString VariableType = iota + 1
	VariableInt
	VariableInt8
	VariableInt16
	VariableInt32
	VariableInt64
	VariableUint
	VariableUint8
	VariableUint16
	VariableUint32
	VariableUint64
	VariableFloat32
	VariableFloat64
	VariableBool
)

func ParseCommands(packageName string, tt []*ast.TypeSpec) (Commands, error) {
	var err error
	commands := make(Commands, len(tt))
	for i, t := range tt {
		commands[i], err = ParseCommand(packageName, t)
		if err != nil {
			return nil, err
		}
	}
	return commands, nil
}

func ParseCommand(packageName string, t *ast.TypeSpec) (*Command, error) {
	st, ok := t.Type.(*ast.StructType)
	if !ok {
		return nil, fmt.Errorf(`wrong struct type for '%s'`, t.Name.Name)
	}

	c := Command{
		Package: packageName,
		Name:   t.Name.Name,
		ShortOptions: make(Fields, 0, st.Fields.NumFields()),
		LongOptions: make(Fields, 0, st.Fields.NumFields()),
		Arguments: make(Fields, 0, st.Fields.NumFields()),
	}

	for _, field := range st.Fields.List {
		f, err := parseField(field)
		if err != nil {
			return nil, fmt.Errorf(`error of parsing %s:%s: %s`, t.Name.Name, field.Names[0].Name, err)
		}
		switch f.Type {
		case FieldOption:
			if len(f.Name) > 0 {
				c.LongOptions = append(c.LongOptions, f)
			}

			if len(f.Short) > 0 {
				c.ShortOptions = append(c.ShortOptions, f)
			}
		case FieldArgument:
			c.Arguments = append(c.Arguments, f)
		}
	}
	return &c, nil

}

func parseField(field *ast.Field) (*Field, error) {
	if len(field.Names) > 1 {
		return nil, fmt.Errorf(`multiple names for field`)
	}
	var err error
	f := Field{
		Name: field.Names[0].Name,
		Type: FieldArgument,
	}
	f.VariableType, err = parseVariableType(field)
	if err != nil {
		return nil, fmt.Errorf(`error parsing variable type: %s`, err)
	}
	props, err := parseProps(field)
	if err != nil {
		return nil, err
	}
	for key, value := range props {
		switch key {
		case `short`:
			f.Short = value
		case `default`:
			f.Default = value
		case `type`:
			t, err := parseType(value)
			if err != nil {
				return nil, err
			}
			f.Type = t
		case `name`:
			f.Name = value
		default:
			return nil, fmt.Errorf(`undefined property '%s' of tag`, key)
		}
	}
	return &f, nil
}

func parseVariableType(field *ast.Field) (VariableType, error) {
	switch field.Type.(type) {
	case *ast.Ident:
		v := field.Type.(*ast.Ident)
		switch v.Name {
		case `string`:
			return VariableString, nil
		case `int`:
			return VariableInt, nil
		case `int8`:
			return VariableInt8, nil
		case `int16`:
			return VariableInt16, nil
		case `int32`:
			return VariableInt32, nil
		case `int64`:
			return VariableInt64, nil
		case `uint`:
			return VariableUint, nil
		case `uint8`:
			return VariableUint8, nil
		case `uint16`:
			return VariableUint16, nil
		case `uint32`:
			return VariableUint32, nil
		case `uint64`:
			return VariableUint64, nil
		case `float32`:
			return VariableFloat32, nil
		case `float64`:
			return VariableFloat64, nil
		case `bool`:
			return VariableBool, nil
		default:
			return 0, fmt.Errorf(`undefined type: %s`, v.Name)
		}
	default:
		return 0, fmt.Errorf(`unknowed type %v`, field.Type)
	}
}

func parseType(value string) (FieldType, error) {
	switch value {
	case `option`:
		return FieldOption, nil
	case `argument`:
		return FieldArgument, nil
	default:
		return 0, fmt.Errorf(`undefined type '%s'`, value)
	}
}

func parseProps(field *ast.Field) (map[string]string, error) {
	props := map[string]string{}
	if field.Tag != nil {
		switch field.Tag.Kind {
		case token.STRING:
			cli, ok := reflect.StructTag(strings.Trim(field.Tag.Value, "`")).Lookup(`cli`)
			if ok {
				var key string
				var value string
				var isValue bool
				var initCollect bool
				var border rune
				for _, item := range cli {
					if item == ':' {
						isValue = true
						continue
					}
					if item == ' ' && border == 0 {
						if initCollect {
							props[key] = value
							isValue = false
							initCollect = false
							border = 0
							key = ``
							value = ``
						}
						continue
					}
					if item == '\'' || item == '"' || item == '`' {
						if border > 0 {
							border = 0
							continue
						}
						border = item
						continue
					}
					initCollect = true
					if isValue {
						value += string(item)
					} else {
						key += string(item)
					}
				}
				if initCollect && len(key) > 0 {
					props[key] = value
				}
			}
		default:
			return nil, fmt.Errorf(`unknowned type for tag '%v'`, field.Tag.Kind)
		}
	}

	return props, nil
}
