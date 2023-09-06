package main

import (
	"fmt"
	"path/filepath"
)

//func matchRegx() {
//	filePath := "money_message.log"
//	target := "money_message.log"
//
//	target = strings.Replace(strings.Replace(target, "[", "\\[", -1), "]", "\\]", -1)
//
//	pattern := fmt.Sprintf(`^%s$`, strings.Replace(strings.Replace(target, ".", "\\.", -1), "*", ".*", -1))
//	fmt.Println(pattern)
//
//	match, err := regexp.MatchString(pattern, filePath)
//	if err != nil {
//		fmt.Println("Error:", err)
//		return
//	}
//
//	if match {
//		fmt.Printf("The file path matches the pattern '%s'\n", target)
//	} else {
//		fmt.Printf("The file path does not match the pattern '%s'\n", target)
//	}
//}
//
//func countAndPositions(s string) (int, bool, bool, bool) {
//	var count int
//	isAtStart := strings.HasPrefix(s, "*")
//	isAtEnd := strings.HasSuffix(s, "*")
//
//	for _, char := range s {
//		if char == '*' {
//			count++
//		}
//	}
//
//	isAtMiddle := !isAtStart && !isAtEnd && count > 0
//
//	return count, isAtStart, isAtEnd, isAtMiddle
//}
//
//func matchSuffix(fileName string, target string) bool {
//
//	_, isAtStart, isAtEnd, isAtMiddle := countAndPositions(target)
//
//	if isAtStart && !isAtEnd && isAtMiddle { // 只在 首位
//		return false
//	} else if !isAtStart && !isAtEnd && isAtMiddle { // 只在 末位
//		return false
//	} else if !isAtStart && !isAtEnd && !isAtMiddle { // 不存在
//		return fileName == target
//	} else {
//		parts := strings.Split(target, "*")
//
//		matched := true
//		lastIndex := len(parts) - 1
//		for i, part := range parts {
//			if part != "" {
//				matched = strings.Contains(fileName, part)
//
//				if !matched {
//					break
//				}
//
//				if i == 0 {
//					index := strings.Index(fileName, part)
//					matched = index == 0
//				}
//
//				if i == lastIndex {
//					index := strings.Index(fileName, part)
//					matchedIndex := len(strings.Replace(fileName, part, "", -1))
//					matched = index == matchedIndex
//				}
//			}
//		}
//
//		return matched
//	}
//}

func main() {

	//fileName := "ddd.id-ddd.[admin@sectex.net].bot"
	//target := "*.id-*.[admin@sectex.net].bot"
	//
	//fmt.Println(strings.Index(target, "*"))

	//matched := matchSuffix(fileName, target)
	//
	//if matched {
	//	fmt.Printf("The file path matches the pattern '%s'\n", target)
	//} else {
	//	fmt.Printf("The file path does not match the pattern '%s'\n", target)
	//}

	str := "abcsd，some_random_string_.tar.gz"

	fmt.Println(filepath.Ext(str))
	//endsWithSuffix := strings.HasPrefix(str, "abcd")
	//if endsWithSuffix {
	//	fmt.Println("The string ends with 'abcd'")
	//} else {
	//	fmt.Println("The string does not end with 'abcd'")
	//}
}
