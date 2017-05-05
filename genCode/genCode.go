package genCode

import (
	"fmt"

	"github.com/bullyork/serviceGen/tool"
	"github.com/spf13/viper"
)

// GenCode 生成ts 前端代码
func GenCode() {
	var sch tool.Schema
	sch = tool.ParseProto(viper.Get("protobufPath").(string))
	for _, v := range sch.Messages {
		fmt.Println(v)
	}
	for _, v := range sch.Services {
		fmt.Println(v)
	}
}
