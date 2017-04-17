package main

import (
	"fmt"
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
