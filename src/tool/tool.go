package tool

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)

var inside = false

// IndexOf 方法
func IndexOf(vs []string, ele string) int {
	for p, v := range vs {
		if v == ele {
			return p
		}
	}
	return -1
}

// Map 方法
func Map(vs []string, f func(string) string) []string {
	vsm := make([]string, len(vs))
	for i, v := range vs {
		vsm[i] = f(v)
	}
	return vsm
}

// Filter 方法
func Filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

// Substr 方法
func Substr(str string, start int, end int) string {
	rs := []rune(str)
	length := len(rs)

	if start < 0 || start > length {
		fmt.Println("start is wrong")
	}

	if end < 0 || end > length {
		fmt.Println("end is wrong")
	}

	return string(rs[start:end])
}

func trimCallback(s string) string {
	return strings.TrimSpace(s)
}

func filterBoolean(s string) bool {
	if s == "" {
		return false
	}
	return true
}

func noComments(line string) string {
	i := strings.Index(line, "//")
	if i > -1 {
		return Substr(line, 0, i)
	}
	return line
}

func noMultilineComments(line string) bool {
	i := strings.Index(line, "/*")
	j := strings.Index(line, "*/")
	if i > -1 {
		inside = true
		return false
	}
	if j > -1 {
		inside = false
		return false
	}
	return !inside
}

func concatCopyPreAllocate(slices [][]string) []string {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}
	tmp := make([]string, totalLen)
	var i int
	for _, s := range slices {
		i += copy(tmp[i:], s)
	}
	return tmp
}

// Token 方法
func Token(path string) []string {
	fileInfo, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("请检查读取路径是否错误！")
	}
	contents := string(fileInfo)
	re, _ := regexp.Compile(`([;,{}\(\)=\:\[\]<>]|\/\*|\*\/)`)
	contents = re.ReplaceAllString(contents, " $1 ")
	contentsArray := strings.Split(contents, "\n")
	contentSlice := contentsArray[:]
	contentSlice = Map(contentSlice, trimCallback)
	contentSlice = Filter(contentSlice, filterBoolean)
	contentSlice = Map(contentSlice, noComments)
	contentSlice = Filter(contentSlice, noMultilineComments)
	contentSlice = Map(contentSlice, trimCallback)
	contentSlice = Filter(contentSlice, filterBoolean)
	contents = strings.Join(contentSlice, "\n")
	contentSlice = regexp.MustCompile(`\s+|\n+`).Split(contents, -1)
	for i, v := range contentSlice {
		re1, _ := regexp.Compile(`^(\"|\')([^\'\"]*)$`)
		flag1 := re1.MatchString(v)
		if flag1 {
			var j int
			if count := strings.Count(v, ""); count == 1 {
				j = i + 1
			} else {
				j = i
			}
			for ; j < len(contentSlice); j++ {
				re2, _ := regexp.Compile(`^([^\'\"]*)(\"|\')$`)
				flag2 := re2.MatchString(contentSlice[j])
				if flag2 {
					s1 := contentSlice[0:i]
					s3 := contentSlice[(j + 1):]
					temp := contentSlice[i:(j + 1)]
					s2 := []string{strings.Join(temp, "")}
					slices := [][]string{s1, s2, s3}
					contentSlice = concatCopyPreAllocate(slices)
				}
			}
		}
	}
	return contentSlice
}
