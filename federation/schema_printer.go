package federation

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/graphql-go/graphql"
)

func printDescription(desc string, indent int, out *strings.Builder) {
	if desc == "" {
		return
	}

	if indent > 0 {
		out.WriteString(strings.Repeat(" ", indent))
	}

	if !strings.Contains(desc, "\n") {
		out.WriteString("\"\"\"")
		out.WriteString(desc)
		out.WriteString("\"\"\"\n")
	} else {
		out.WriteString("\"\"\"\n")
		for _, d := range strings.Split(desc, "\n") {
			out.WriteString(strings.Repeat(" ", indent))
			out.WriteString(d)
			out.WriteString("\n")
		}
		out.WriteString(strings.Repeat(" ", indent))
		out.WriteString("\"\"\"\n")
	}
}

// schema

func printSchemaDefinition(schema graphql.Schema, out *strings.Builder) {
	out.WriteString("schema")
	printAppliedDirectives(schema.AppliedDirectives(), "", out)

	if schema.QueryType() != nil {
		out.WriteString(" {\n")
		fmt.Fprintf(out, "  query: %v\n", schema.QueryType().Name())
	} else {
		panic("invalid schema - schema requires valid query type")
	}

	if schema.MutationType() != nil {
		fmt.Fprintf(out, "  mutation: %v\n", schema.MutationType().Name())
	}

	if schema.SubscriptionType() != nil {
		fmt.Fprintf(out, "  subscription: %v\n", schema.SubscriptionType().Name())
	}
	out.WriteString("}\n\n")
}

// directives

func printDirectiveDefinitions(directives []*graphql.Directive, out *strings.Builder) {
	for _, directive := range sortDirectiveDefinitions(directives) {
		printDirectiveDefinition(directive, out)
	}
}

func sortDirectiveDefinitions(directives []*graphql.Directive) []*graphql.Directive {
	sorted := make([]*graphql.Directive, 0, len(directives))
	for _, v := range directives {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

// Example
// "Marks an element of a GraphQL schema as no longer supported."
// directive @deprecated(reason: String) on FIELD_DEFINITION | ENUM_VALUE
func printDirectiveDefinition(directive *graphql.Directive, out *strings.Builder) {
	printDescription(directive.Description, 0, out)
	out.WriteString("directive @")
	out.WriteString(directive.Name)
	if len(directive.Args) > 0 {
		out.WriteString("(")

		args := make([]string, 0, len(directive.Args))
		for _, arg := range directive.Args {
			switch arg.Type.(type) {
			case *graphql.List:
				args = append(args, fmt.Sprintf("%s: [%s]", arg.Name(), arg.Type.Name()))
			default:
				var defaultValue string
				if arg.DefaultValue != nil {
					switch arg.DefaultValue.(type) {
					case string:
						defaultValue = fmt.Sprintf(" = \"%v\"", arg.DefaultValue)
					default:
						defaultValue = fmt.Sprintf(" = %v", arg.DefaultValue)
					}
				}
				args = append(args, fmt.Sprintf("%s: %s%s", arg.Name(), arg.Type.Name(), defaultValue))
			}
		}
		out.WriteString(strings.Join(args, ", "))
		out.WriteString(")")
	}

	if directive.Repeatable {
		out.WriteString(" repeatable")
	}

	out.WriteString(" on ")
	out.WriteString(strings.Join(directive.Locations, " | "))
	out.WriteString("\n\n")
}

func printAppliedDirectives(appliedDirectives []*graphql.AppliedDirective, deprecationReason string, out *strings.Builder) {
	if deprecationReason != "" {
		fmt.Fprintf(out, " @deprecated(reason: %q)", deprecationReason)
	}

	if len(appliedDirectives) == 0 {
		return
	}
	for _, appliedDirective := range sortAppliedDirectives(appliedDirectives) {
		out.WriteString(" ")
		printAppliedDirective(appliedDirective, out)
	}
}

func sortAppliedDirectives(appliedDirectives []*graphql.AppliedDirective) []*graphql.AppliedDirective {
	sorted := make([]*graphql.AppliedDirective, 0, len(appliedDirectives))
	for _, v := range appliedDirectives {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}

func printAppliedDirective(applied *graphql.AppliedDirective, out *strings.Builder) {
	fmt.Fprintf(out, "@%s", applied.Name)
	if len(applied.Args) > 0 {
		out.WriteString("(")

		args := []string{}
		for _, arg := range applied.Args {
			value := printAppliedDirectiveArgumentValue(arg.Value)
			if len(value) > 1 {
				args = append(args, fmt.Sprintf("%s: [%s]", arg.Name, strings.Join(value, ", ")))
			} else {
				args = append(args, fmt.Sprintf("%s: %v", arg.Name, value[0]))
			}
		}
		out.WriteString(strings.Join(args, ", "))
		out.WriteString(")")
	}
}

func printAppliedDirectiveArgumentValue(arg interface{}) []string {
	printedValues := []string{}

	switch reflect.TypeOf(arg).Kind() {
	case reflect.Array, reflect.Slice:
		array := reflect.ValueOf(arg)
		for i := 0; i < array.Len(); i++ {
			values := printAppliedDirectiveArgumentValue(array.Index(i).Interface())
			printedValues = append(printedValues, values...)
		}
	case reflect.String:
		printedValues = append(printedValues, fmt.Sprintf("%q", arg))
	default:
		printedValues = append(printedValues, fmt.Sprintf("%v", arg))
	}
	return printedValues
}

// enums

func printEnumDefinitions(enums []*graphql.Enum, out *strings.Builder) {
	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name() < enums[j].Name()
	})

	for _, enum := range enums {
		printDescription(enum.Description(), 0, out)
		fmt.Fprintf(out, "enum %s", enum.Name())
		printAppliedDirectives(enum.AppliedDirectives, "", out)
		out.WriteString(" {\n")

		// enum values
		sortedValues := make([]*graphql.EnumValueDefinition, len(enum.Values()))
		copy(sortedValues, enum.Values())
		sort.Slice(sortedValues, func(i, j int) bool {
			return sortedValues[i].Name < sortedValues[j].Name
		})
		for _, enumValue := range sortedValues {
			printDescription(enumValue.Description, 2, out)
			out.WriteString("  ")
			out.WriteString(enumValue.Name)
			printAppliedDirectives(enumValue.AppliedDirectives, enumValue.DeprecationReason, out)
			out.WriteString("\n")
		}

		out.WriteString("}\n\n")
	}
}

// input object

func printInputObjectDefinitions(inputObjects []*graphql.InputObject, out *strings.Builder) {
	sort.Slice(inputObjects, func(i, j int) bool {
		return inputObjects[i].Name() < inputObjects[j].Name()
	})

	for _, inputObject := range inputObjects {
		printDescription(inputObject.Description(), 0, out)
		fmt.Fprintf(out, "input %s", inputObject.Name())
		printAppliedDirectives(inputObject.AppliedDirectives, "", out)
		out.WriteString(" {\n")
		printInputObjectFieldDefinitions(inputObject.Fields(), out)
		out.WriteString("}\n\n")
	}
}

// input object fields

func printInputObjectFieldDefinitions(inputFields graphql.InputObjectFieldMap, out *strings.Builder) {
	keys := make([]string, 0, len(inputFields))
	for k := range inputFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		field := inputFields[key]
		printDescription(field.Description(), 2, out)
		fmt.Fprintf(out, "  %s: %s", field.Name(), field.Type.String())
		printAppliedDirectives(field.AppliedDirectives, "", out)
		out.WriteString("\n")
	}
}

// interfaces

func printInterfaceDefinitions(interfaces []*graphql.Interface, out *strings.Builder) {
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Name() < interfaces[j].Name()
	})

	for _, intf := range interfaces {
		printDescription(intf.Description(), 0, out)
		fmt.Fprintf(out, "interface %s", intf.Name())
		printAppliedDirectives(intf.AppliedDirectives, "", out)
		out.WriteString(" {\n")
		printFieldDefinitions(intf.Fields(), out)
		out.WriteString("}\n\n")
	}
}

// objects

func printObjectDefinitions(objects []*graphql.Object, out *strings.Builder) {
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Name() < objects[j].Name()
	})

	for _, object := range objects {
		printDescription(object.Description(), 0, out)
		fmt.Fprintf(out, "type %s", object.Name())
		if len(object.Interfaces()) > 0 {
			interfaces := make([]string, 0, len(object.Interfaces()))
			for _, i := range object.Interfaces() {
				interfaces = append(interfaces, i.Name())
			}
			out.WriteString(" implements ")
			out.WriteString(strings.Join(interfaces, ", "))
		}
		printAppliedDirectives(object.AppliedDirectives, "", out)
		out.WriteString(" {\n")
		printFieldDefinitions(object.Fields(), out)
		out.WriteString("}\n\n")
	}
}

func printFieldDefinitions(fieldDefinitionMap graphql.FieldDefinitionMap, out *strings.Builder) {
	keys := make([]string, 0, len(fieldDefinitionMap))
	for k := range fieldDefinitionMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		field := fieldDefinitionMap[key]
		printDescription(field.Description, 2, out)
		fmt.Fprintf(out, "  %s", field.Name)

		if len(field.Args) > 0 {
			out.WriteString("(")
			args := make([]string, 0, len(field.Args))
			for _, arg := range field.Args {
				args = append(args, fmt.Sprintf("%s: %s", arg.Name(), arg.Type.Name()))
			}
			out.WriteString(strings.Join(args, ", "))
			out.WriteString(")")
		}

		fmt.Fprintf(out, ": %s", field.Type.Name())
		printAppliedDirectives(field.AppliedDirectives, field.DeprecationReason, out)
		out.WriteString("\n")
	}
}

// unions

func printUnionDefinitions(unions []*graphql.Union, out *strings.Builder) {
	sort.Slice(unions, func(i, j int) bool {
		return unions[i].Name() < unions[j].Name()
	})

	for _, union := range unions {
		printDescription(union.Description(), 0, out)
		fmt.Fprintf(out, "union %s", union.Name())
		printAppliedDirectives(union.AppliedDirectives, "", out)
		typeNames := make([]string, 0, len(union.Types()))
		for _, t := range union.Types() {
			typeNames = append(typeNames, t.Name())
		}
		sort.Slice(typeNames, func(i, j int) bool {
			return typeNames[i] < typeNames[j]
		})
		out.WriteString(" = ")
		out.WriteString(strings.Join(typeNames, " | "))
		out.WriteString("\n\n")
	}
}

// scalars

func printCustomScalars(scalars []*graphql.Scalar, out *strings.Builder) {
	sort.Slice(scalars, func(i, j int) bool {
		return scalars[i].Name() < scalars[j].Name()
	})

	for _, scalar := range scalars {
		printDescription(scalar.Description(), 0, out)
		fmt.Fprintf(out, "scalar %s", scalar.Name())
		printAppliedDirectives(scalar.AppliedDirectives, "", out)
		out.WriteString("\n\n")
	}
}

// utils

func isSchemaDefinitionNeeded(schema graphql.Schema) bool {
	if schema.QueryType() != nil && schema.QueryType().Name() != "Query" {
		return true
	}
	if schema.MutationType() != nil && schema.MutationType().Name() != "Mutation" {
		return true
	}
	if schema.SubscriptionType() != nil && schema.SubscriptionType().Name() != "Subscription" {
		return true
	}
	return false
}

// public API

type PrinterOptions struct {
	IncludeDirectiveDefinition bool
	IncludeSchemaDefinition    bool
}

var DefaultPrinterOptions = PrinterOptions{
	IncludeDirectiveDefinition: true,
	IncludeSchemaDefinition:    true,
}

func PrintSchema(schema graphql.Schema, options PrinterOptions) string {
	enums := make([]*graphql.Enum, 0, 0)
	inputObjects := make([]*graphql.InputObject, 0, 0)
	interfaces := make([]*graphql.Interface, 0, 0)
	objects := make([]*graphql.Object, 0, 0)
	unions := make([]*graphql.Union, 0, 0)
	scalars := make([]*graphql.Scalar, 0, 0)

	buitlInScalars := map[string]bool{
		"Boolean": true, "Float": true, "ID": true, "Int": true, "String": true,
	}
	for name, gqlType := range schema.TypeMap() {
		// skip built in types
		_, builtIn := buitlInScalars[name]
		if strings.HasPrefix(name, "__") || builtIn {
			continue
		}

		switch t := gqlType.(type) {
		case *graphql.Enum:
			enums = append(enums, t)
		case *graphql.InputObject:
			inputObjects = append(inputObjects, t)
		case *graphql.Interface:
			interfaces = append(interfaces, t)
		case *graphql.Object:
			objects = append(objects, t)
		case *graphql.Union:
			unions = append(unions, t)
		case *graphql.Scalar:
			scalars = append(scalars, t)
		default:
			// TODO unknown -> panic?
		}
	}

	var sdl strings.Builder

	if options.IncludeSchemaDefinition || isSchemaDefinitionNeeded(schema) {
		printSchemaDefinition(schema, &sdl)
	}
	if options.IncludeDirectiveDefinition {
		printDirectiveDefinitions(schema.Directives(), &sdl)
	}
	printEnumDefinitions(enums, &sdl)
	printInputObjectDefinitions(inputObjects, &sdl)
	printInterfaceDefinitions(interfaces, &sdl)
	printObjectDefinitions(objects, &sdl)
	printUnionDefinitions(unions, &sdl)
	printCustomScalars(scalars, &sdl)

	return strings.TrimSpace(sdl.String())
}
