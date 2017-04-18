package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/bullyork/serviceGen/src/tool"
)

type schema struct {
	syntax   int
	imports  []interface{}
	enums    []int
	messages []message
	options  map[string]interface{}
	extends  []interface{}
	pack     string
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
				fmt.Println("Unexpected token in field options: " + tokens.data[0])
			}
			shift(tokens)
			if tokens.data[0] == "]" {
				fmt.Println("Unexpected ] in field option")
			}
			opts["name"] = shift(tokens)
		case "]":
			shift(tokens)
			return opts
		default:
			fmt.Println("Unexpected token in field options: " + tokens.data[0])
		}
	}
	fmt.Println("No closing tag for field options")
	return opts
}

func onpackagename(tokens *tokensArray) string {
	shift(tokens)
	name := tokens.data[0]
	shift(tokens)
	if tokens.data[0] != ";" {
		fmt.Println("Expected ; but found " + tokens.data[0])
	}
	shift(tokens)
	return name
}

func onsyntaxversion(tokens *tokensArray) int {
	shift(tokens)
	if tokens.data[0] != "=" {
		fmt.Println("Expected = but found " + tokens.data[0])
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
		fmt.Println("Expected protobuf syntax version but found " + versionStr)
	}
	if tokens.data[0] != ";" {
		fmt.Println("Expected ; but found " + tokens.data[0])
	}
	shift(tokens)
	return version
}

type message struct {
	name     string
	enums    []int
	extends  []string
	messages []string
	fields   []string
}

type field struct {
	name     string
	typeArea string
	tag      int
	mapArea  map[string]string
	required bool
	repeated bool
	options  map[string]string
}

type messageBody struct {
	enums      []int
	messages   []string
	fields     []field
	extends    []string
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
				fmt.Println(err)
			}
		case "map":
			field.typeArea = "map"
			shift(tokens)
			if tokens.data[0] != "<" {
				fmt.Println(`Unexpected token in map type: ` + tokens.data[0])
			}
			shift(tokens)
			field.mapArea["from"] = shift(tokens)
			if tokens.data[0] != "," {
				fmt.Println(`Unexpected token in map type: ` + tokens.data[0])
			}
			shift(tokens)
			field.mapArea["to"] = shift(tokens)
			if tokens.data[0] != ">" {
				fmt.Println(`Unexpected token in map type: ` + tokens.data[0])
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
				fmt.Println("Missing field name")
			}
			if field.typeArea == "" {
				fmt.Println("Missing type in message field: " + field.name)
			}
			if field.tag == -1 {
				fmt.Println("Missing tag number in message field: " + field.name)
			}
			shift(tokens)
			return field
		default:
			fmt.Println("Unexpected token in message field: " + tokens.data[0])
		}
	}
	fmt.Println("No ; found for message field")
	return field
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
				fmt.Println("Expected ) but found " + tokens.data[0])
			}
		}
		var value interface{}
		switch tokens.data[0] {
		case ":":
			if result[key] != nil {
				fmt.Println("Duplicate option map key " + key)
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
				fmt.Println("Duplicate option map key " + key)
			}
			l := v.Len()
			sliceValue := make([]interface{}, l)
			for i := 0; i < l; i++ {
				sliceValue[i] = v.Index(i).Interface()
			}
			result[key] = append(sliceValue, value)
		default:
			fmt.Println("Unexpected token in option map: " + tokens.data[0])
		}
	}
	fmt.Println("No closing tag for option map")
	return result
}

func onoption(tokens *tokensArray) map[string]interface{} {
	var name string
	var value interface{}
	var result map[string]interface{}
	for len(tokens.data) > 0 {
		if tokens.data[0] == ";" {
			shift(tokens)
			result["name"] = name
			result["value"] = value
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
					fmt.Println("Expected ) but found " + tokens.data[0])
				}
				shift(tokens)
			}
		case "=":
			shift(tokens)
			if name == "" {
				fmt.Println("Expected key for option with value: " + tokens.data[0])
			}
			value = parse(shift(tokens))
			re, _ := regexp.Compile(`^(SPEED|CODE_SIZE|LITE_RUNTIME)$`)
			flag := re.MatchString(value.(string))
			if name == "optimize_for" && !flag {
				fmt.Println("Unexpected value for option optimize_for: " + value.(string))
			} else if value.(string) == "{" {
				value = onoptionMap(tokens)
			}
		default:
			fmt.Println("Unexpected token in option: " + tokens.data[0])
		}
	}
	return result
}

func onenum(tokens *tokensArray) map[string]interface{} {
	shift(tokens)
	var e map[string]interface{}
	e["name"] = shift(tokens)
	if tokens.data[0] != "{" {
		fmt.Println("Expected { but found " + tokens.data[0])
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
		}
	}
}

func onmessagebody(tokens *tokensArray) messageBody {
	var body messageBody
	for len(tokens.data) > 0 {
		switch tokens.data[0] {
		case "map", "repeated", "optional", "required":
			append(body.fields, onfield(tokens))
		case "enum":
			body.enums.push(onenum(tokens))
		}
	}
}

func onmessage(tokens *tokensArray) message {
	shift(tokens)
	lvl := 1
	var bodyTokens tokensArray
	var msg message
	msg.name = shift(tokens)
	if tokens.data[0] != "{" {
		fmt.Println(`Expected { but found '` + tokens.data[0])
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
			body = onmessagebody(*bodyTokens)
			msg.enums = body.enums
			msg.messages = body.messages
			msg.fields = body.fields
			msg.extends = body.extends
			msg.extensions = body.extensions
			return msg
		}
		bodyTokens.data = append(bodyTokens.data, shift(tokens))
	}
}

func shift(tokens *tokensArray) string {
	str := tokens.data[0]
	tokens.data = tokens.data[1:]
	return str
}

func main() {
	var tokens tokensArray
	tokens.data = tool.Token("../proto/express/common.proto")

	var sch schema
	firstline := true
	for len(tokens.data) > 0 {
		switch tokens.data[0] {
		case "package":
			sch.pack = onpackagename(&tokens)
		case "syntax":
			sch.syntax = onsyntaxversion(&tokens)
		case "message":
			append(sch.messages, onmessage(&tokens))
		}
	}
}
