package tool

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var maxRange = 0x1FFFFFFF
var packableTypes = []string{"int32", "int64", "uint32", "uint64", "sint32", "sint64", "bool", "fixed64", "sfixed64", "double", "fixed32", "sfixed32", "float"}

// Schema 结构 应该可以被导出
type Schema struct {
	Syntax   int
	Imports  []interface{}
	Enums    []interface{}
	Messages []message
	Options  map[string]interface{}
	Extends  []extand
	Pack     string
	Services []interface{}
}

type tokensArray struct {
	data []string
}

func onfieldoptions(tokens *tokensArray) map[string]string {
	var opts map[string]string
	for len(tokens.data) > 0 {
		switch tokens.data[0] {
		case "[", ",":
			shift(tokens)
			name := shift(tokens)
			if name == "(" {
				name = shift(tokens)
				shift(tokens)
			}
			if tokens.data[0] != "=" {
				panic("Unexpected token in field options: " + tokens.data[0])
			}
			shift(tokens)
			if tokens.data[0] == "]" {
				panic("Unexpected ] in field option")
			}
			opts["name"] = shift(tokens)
		case "]":
			shift(tokens)
			return opts
		default:
			panic("Unexpected token in field options: " + tokens.data[0])
		}
	}
	panic("No closing tag for field options")
}

func onpackagename(tokens *tokensArray) string {
	shift(tokens)
	name := tokens.data[0]
	shift(tokens)
	if tokens.data[0] != ";" {
		panic("Expected ; but found " + tokens.data[0])
	}
	shift(tokens)
	return name
}

func onsyntaxversion(tokens *tokensArray) int {
	shift(tokens)
	if tokens.data[0] != "=" {
		panic("Expected = but found " + tokens.data[0])
	}
	shift(tokens)
	versionStr := tokens.data[0]
	var version int
	shift(tokens)
	switch versionStr {
	case `"proto2"`:
		version = 2
	case `"proto3"`:
		version = 3
	default:
		panic("Expected protobuf syntax version but found " + versionStr)
	}
	if tokens.data[0] != ";" {
		panic("Expected ; but found " + tokens.data[0])
	}
	shift(tokens)
	return version
}

type message struct {
	name       string
	enums      []interface{}
	extends    []interface{}
	messages   []interface{}
	fields     []field
	extensions extensions
}

type field struct {
	name     string
	typeArea string
	tag      int
	mapArea  map[string]string
	required bool
	repeated bool
	options  map[string]string
	oneof    string
}

type messageBody struct {
	enums      []interface{}
	messages   []interface{}
	fields     []field
	extends    []interface{}
	extensions extensions
}

type extensions struct {
	from int
	to   int
}

func onfield(tokens *tokensArray) field {
	var field field
	for len(tokens.data) > 0 {
		switch tokens.data[0] {
		case "=":
			shift(tokens)
			if v, err := strconv.Atoi(shift(tokens)); err == nil {
				field.tag = v
			} else {
				panic(err)
			}
		case "map":
			field.typeArea = "map"
			shift(tokens)
			if tokens.data[0] != "<" {
				panic(`Unexpected token in map type: ` + tokens.data[0])
			}
			shift(tokens)
			field.mapArea["from"] = shift(tokens)
			if tokens.data[0] != "," {
				panic(`Unexpected token in map type: ` + tokens.data[0])
			}
			shift(tokens)
			field.mapArea["to"] = shift(tokens)
			if tokens.data[0] != ">" {
				panic(`Unexpected token in map type: ` + tokens.data[0])
			}
			shift(tokens)
			field.name = shift(tokens)
		case "repeated", "required", "optional":
			var t = shift(tokens)
			field.required = (t == "required")
			field.repeated = (t == "repeated")
			field.typeArea = shift(tokens)
			field.name = shift(tokens)
		case "[":
			field.options = onfieldoptions(tokens)
		case ";":
			if field.name == "" {
				panic("Missing field name")
			}
			if field.typeArea == "" {
				panic("Missing type in message field: " + field.name)
			}
			if field.tag == -1 {
				panic("Missing tag number in message field: " + field.name)
			}
			shift(tokens)
			return field
		default:
			panic("Unexpected token in message field: " + tokens.data[0])
		}
	}
	panic("No ; found for message field")
}

func parse(value string) interface{} {
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}
	return value
}

func toSlice(arr interface{}) []interface{} {
	v := reflect.ValueOf(arr)
	if v.Kind() != reflect.Slice {
		panic("toslice arr not slice")
	}
	l := v.Len()
	ret := make([]interface{}, l)
	for i := 0; i < l; i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

func onoptionMap(tokens *tokensArray) interface{} {
	var result map[string]interface{}
	for len(tokens.data) > 0 {
		if tokens.data[0] == "}" {
			shift(tokens)
			return result
		}
		hasBracket := tokens.data[0] == "("
		if hasBracket {
			shift(tokens)
		}
		key := shift(tokens)
		if hasBracket {
			if tokens.data[0] != ")" {
				panic("Expected ) but found " + tokens.data[0])
			}
		}
		var value interface{}
		switch tokens.data[0] {
		case ":":
			if result[key] != nil {
				panic("Duplicate option map key " + key)
			}
			shift(tokens)
			value = parse(shift(tokens))
			if value.(string) == "{" {
				value = onoptionMap(tokens)
			}
			result[key] = value
		case "{":
			shift(tokens)
			value = onoptionMap(tokens)
			if result[key] == nil {
				var s = make([]interface{}, 0)
				result[key] = s
			}
			v := reflect.ValueOf(result[key])
			if v.Kind() != reflect.Slice {
				panic("Duplicate option map key " + key)
			}
			l := v.Len()
			sliceValue := make([]interface{}, l)
			for i := 0; i < l; i++ {
				sliceValue[i] = v.Index(i).Interface()
			}
			result[key] = append(sliceValue, value)
		default:
			panic("Unexpected token in option map: " + tokens.data[0])
		}
	}
	panic("No closing tag for option map")
}

type optionsStruct struct {
	name  string
	value interface{}
}

func onoption(tokens *tokensArray) optionsStruct {
	var name string
	var value interface{}
	var result optionsStruct
	for len(tokens.data) > 0 {
		if tokens.data[0] == ";" {
			shift(tokens)
			result.name = name
			result.value = value
			return result
		}
		switch tokens.data[0] {
		case "option":
			shift(tokens)
			hasBracket := tokens.data[0] == "("
			if hasBracket {
				shift(tokens)
			}
			name = shift(tokens)
			if hasBracket {
				if tokens.data[0] != ")" {
					panic("Expected ) but found " + tokens.data[0])
				}
				shift(tokens)
			}
		case "=":
			shift(tokens)
			if name == "" {
				panic("Expected key for option with value: " + tokens.data[0])
			}
			value = parse(shift(tokens))
			re, _ := regexp.Compile(`^(SPEED|CODE_SIZE|LITE_RUNTIME)$`)
			flag := re.MatchString(value.(string))
			if name == "optimize_for" && !flag {
				panic("Unexpected value for option optimize_for: " + value.(string))
			} else if value.(string) == "{" {
				value = onoptionMap(tokens)
			}
		default:
			panic("Unexpected token in option: " + tokens.data[0])
		}
	}
	return result
}

type enumValue struct {
	name string
	val  interface{}
}

func onenumvalue(tokens *tokensArray) enumValue {
	var result enumValue
	if len(tokens.data) < 4 {
		info := strings.Join(tokens.data[0:3], " ")
		panic("Invalid enum value: " + info)
	}
	if tokens.data[1] != "=" {
		panic("Expected = but found " + tokens.data[1])
	}
	if tokens.data[3] != ";" {
		panic("Expected ; or [ but found " + tokens.data[1])
	}
	name := shift(tokens)
	shift(tokens)
	var val map[string]interface{}
	val["value"], _ = strconv.Atoi(shift(tokens))
	if tokens.data[0] == "[" {
		val["options"] = onfieldoptions(tokens)
	}
	shift(tokens)
	result.name = name
	result.val = val
	return result
}

func onenum(tokens *tokensArray) map[string]interface{} {
	shift(tokens)
	var e map[string]interface{}
	e["name"] = shift(tokens)
	if tokens.data[0] != "{" {
		panic("Expected { but found " + tokens.data[0])
	}
	shift(tokens)
	for len(tokens.data) > 0 {
		if tokens.data[0] == "}" {
			shift(tokens)
			if tokens.data[0] == ";" {
				shift(tokens)
			}
			return e
		}
		if tokens.data[0] == "option" {
			options := onoption(tokens)
			e["options"] = options
		}
		var val = onenumvalue(tokens)
		e["values"] = val
	}
	panic("No closing tag for enum")
}

func onextensions(tokens *tokensArray) extensions {
	shift(tokens)
	from, err1 := strconv.Atoi(shift(tokens))
	if err1 != nil {
		panic("Invalid from in extensions definition")
	}
	if shift(tokens) != "to" {
		panic("Expected keyword 'to' in extensions definition")
	}
	var to = shift(tokens)
	var toNumber int
	var err2 error
	if to == "max" {
		toNumber = maxRange
	}
	toNumber, err2 = strconv.Atoi(to)
	if err2 != nil {
		panic("Invalid to in extensions definition")
	}
	if shift(tokens) != ";" {
		panic("Missing ; in extensions definition")
	}
	var result extensions
	result.from = from
	result.to = toNumber
	return result
}

type extand struct {
	name    string
	message message
}

func onextend(tokens *tokensArray) extand {
	var out extand
	out.name = tokens.data[1]
	out.message = onmessage(tokens)
	return out
}

func onmessagebody(tokens *tokensArray) messageBody {
	var body messageBody
	for len(tokens.data) > 0 {
		switch tokens.data[0] {
		case "map", "repeated", "optional", "required":
			body.fields = append(body.fields, onfield(tokens))
		case "enum":
			body.enums = append(body.enums, onenum(tokens))
		case "message":
			body.messages = append(body.messages, onmessage(tokens))
		case "extensions":
			body.extensions = onextensions(tokens)
		case "oneof":
			shift(tokens)
			name := shift(tokens)
			if tokens.data[0] != "{" {
				panic("Unexpected token in oneof: " + tokens.data[0])
			}
			shift(tokens)
			for tokens.data[0] != "}" {
				unshift(tokens, "optional")
				field := onfield(tokens)
				field.oneof = name
				body.fields = append(body.fields, field)
			}
			shift(tokens)
		case "extend":
			body.extends = append(body.extends, onextend(tokens))
		case ";":
			shift(tokens)
		case "reserved", "option":
			shift(tokens)
			for tokens.data[0] != ";" {
				shift(tokens)
			}
		default:
			unshift(tokens, "optional")
			body.fields = append(body.fields, onfield(tokens))
		}
	}
	return body
}

func onmessage(tokens *tokensArray) message {
	shift(tokens)
	lvl := 1
	var bodyTokens tokensArray
	var msg message
	msg.name = shift(tokens)
	if tokens.data[0] != "{" {
		panic(`Expected { but found '` + tokens.data[0])
	}
	shift(tokens)
	for len(tokens.data) > 0 {
		if tokens.data[0] == "{" {
			lvl++
		} else if tokens.data[0] == "}" {
			lvl--
		}
		if lvl == 0 {
			shift(tokens)
			body := onmessagebody(&bodyTokens)
			msg.enums = body.enums
			msg.messages = body.messages
			msg.fields = body.fields
			msg.extends = body.extends
			msg.extensions = body.extensions
			return msg
		}
		bodyTokens.data = append(bodyTokens.data, shift(tokens))
	}
	if lvl == 0 {
		panic("No closing tag for message")
	}
	return msg
}

func shift(tokens *tokensArray) string {
	str := tokens.data[0]
	tokens.data = tokens.data[1:]
	return str
}

func unshift(tokens *tokensArray, str string) string {
	tokens.data = append([]string{str}, tokens.data...)
	return str
}

func onimport(tokens *tokensArray) string {
	shift(tokens)
	str := shift(tokens)
	re, _ := regexp.Compile(`^"+|"+$`)
	file := re.ReplaceAllString(str, "")
	if tokens.data[0] != ";" {
		panic("Unexpected token: " + tokens.data[0] + `. Expected ";"`)
	}
	shift(tokens)
	return file
}

type rpcType struct {
	name            string
	inputType       string
	outputType      string
	clientStreaming bool
	serverStreaming bool
	options         map[string]interface{}
}

func onrpc(tokens *tokensArray) rpcType {
	shift(tokens)
	var rpc rpcType
	rpc.name = shift(tokens)
	if tokens.data[0] != "(" {
		panic("Expected ( but found " + tokens.data[0])
	}
	shift(tokens)
	if tokens.data[0] == "stream" {
		shift(tokens)
		rpc.clientStreaming = true
	}
	rpc.inputType = shift(tokens)

	if tokens.data[0] != ")" {
		panic("Expected ) but found " + tokens.data[0])
	}
	shift(tokens)
	if tokens.data[0] != "returns" {
		panic("Expected returns but found " + tokens.data[0])
	}
	shift(tokens)
	if tokens.data[0] != "(" {
		panic("Expected ( but found " + tokens.data[0])
	}
	if tokens.data[0] != "stream" {
		shift(tokens)
		rpc.serverStreaming = true
	}
	rpc.outputType = shift(tokens)

	if tokens.data[0] != ")" {
		panic("Expected ) but found " + tokens.data[0])
	}
	shift(tokens)

	if tokens.data[0] == ";" {
		shift(tokens)
		return rpc
	}

	if tokens.data[0] != "{" {
		panic("Expected { but found " + tokens.data[0])
	}
	shift(tokens)

	for len(tokens.data) > 0 {
		if tokens.data[0] == "}" {
			shift(tokens)
			if tokens.data[0] == ";" {
				shift(tokens)
			}
			return rpc
		}
		if tokens.data[0] == "option" {
			opt := onoption(tokens)
			if rpc.options[opt.name] != nil {
				panic("Duplicate option " + opt.name)
			}
			rpc.options[opt.name] = opt.value
		} else {
			panic("Unexpected token in rpc options: " + tokens.data[0])
		}
	}
	panic("No closing tag for rpc")
}

type serviceType struct {
	name    string
	methods []interface{}
	options map[string]interface{}
}

func enumNameIsFieldType(enums []interface{}, field field) bool {
	for _, v := range enums {
		if v.(string) == field.typeArea {
			return true
		}
	}
	return false
}

func enumNameIsNestedEnumName(msg message, nestedEnumName string) bool {
	for _, v := range msg.enums {
		if v.(string) == nestedEnumName {
			return true
		}
	}
	return false
}

func onservice(tokens *tokensArray) serviceType {
	shift(tokens)
	var service serviceType
	service.name = shift(tokens)
	if tokens.data[0] != "{" {
		panic("Expected { but found " + tokens.data[0])
	}
	shift(tokens)
	for len(tokens.data) > 0 {
		if tokens.data[0] == "}" {
			shift(tokens)
			if len(tokens.data) > 0 && tokens.data[0] == ";" {
				shift(tokens)
			}
			return service
		}
		switch tokens.data[0] {
		case "option":
			opt := onoption(tokens)
			if service.options[opt.name] != nil {
				panic("Duplicate option " + opt.name)
			}
			service.options[opt.name] = opt.value
		case "rpc":
			service.methods = append(service.methods, onrpc(tokens))
		default:
			panic("Unexpected token in service: " + tokens.data[0])
		}
	}
	panic("No closing tag for service")
}

// ParseProto 方法 用于转换protobuf
func ParseProto(path string) Schema {
	var tokens tokensArray
	tokens.data = Token(path)

	var sch Schema
	firstline := true
	for len(tokens.data) > 0 {
		switch tokens.data[0] {
		case "package":
			sch.Pack = onpackagename(&tokens)
		case "syntax":
			if !firstline {
				panic("Protobuf syntax version should be first thing in file")
			}
			sch.Syntax = onsyntaxversion(&tokens)
		case "message":
			sch.Messages = append(sch.Messages, onmessage(&tokens))
		case "enum":
			sch.Enums = append(sch.Enums, onenum(&tokens))
		case "option":
			opt := onoption(&tokens)
			if sch.Options[opt.name] != nil {
				panic("Duplicate option " + opt.name)
			}
			sch.Options[opt.name] = opt.value
		case "import":
			sch.Imports = append(sch.Imports, onimport(&tokens))
		case "extend":
			sch.Extends = append(sch.Extends, onextend(&tokens))
		case "service":
			sch.Services = append(sch.Services, onservice(&tokens))
		default:
			panic("Unexpected token: " + tokens.data[0])
		}
		firstline = false
	}
	for _, ext := range sch.Extends {
		for _, msg := range sch.Messages {
			if msg.name == ext.name {
				for _, field := range ext.message.fields {
					if (msg.extensions.from == 0 && msg.extensions.to == 0) || field.tag < msg.extensions.from || field.tag > msg.extensions.to {
						panic(msg.name + " does not declare " + strconv.Itoa(field.tag) + "as an extension number")
					}
					msg.fields = append(msg.fields, field)
				}
			}
		}
	}
	for _, msg := range sch.Messages {
		for _, field := range msg.fields {
			if field.options == nil && field.options["packed"] == "true" {
				if IndexOf(packableTypes, field.typeArea) == -1 {
					if strings.Index(field.typeArea, ".") == -1 {
						isFieldType := enumNameIsFieldType(msg.enums, field)
						if len(msg.enums) != 0 && isFieldType {
							return sch
						}
					} else {
						fieldSplit := strings.Split(field.typeArea, ".")
						if len(fieldSplit) > 2 {
							panic("what is this?")
						}
						messageName := fieldSplit[0]
						nestedEnumName := fieldSplit[1]
						var message message
						for _, mssg := range sch.Messages {
							if mssg.name == messageName {
								message = msg
								break
							}
						}
						isNestedEnumName := enumNameIsNestedEnumName(message, nestedEnumName)
						if len(message.enums) > 0 && isNestedEnumName {
							return sch
						}
					}
					panic("Fields of type " + field.typeArea + `cannot be declared [packed=true]. ' +
            'Only repeated fields of primitive numeric types (types which use ` +
						`the varint, 32-bit, or 64-bit wire types) can be declared "packed". ` +
						`See https://developers.google.com/protocol-buffers/docs/encoding#optional`)
				}
			}
		}
	}
	return sch
}
