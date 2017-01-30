package utils

import (
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// QueryUnescape извлекает URL адрес из строки, переданной в поле http запроса
func QueryUnescape(s string) (string, bool) {

	if strings.HasPrefix(s, "/") {
		s = s[1:]
	}
	// заменить вхождения "_1" на "%"
	s = regexp.MustCompile("_2").ReplaceAllString(s, "%")
	// заменить вхождения "_2" на "_"
	s = regexp.MustCompile("_1").ReplaceAllString(s, "_")
	// разэкранировать url
	var s1, err = url.QueryUnescape(s)
	if err != nil {
		return "", false
	}
	return s1, true
}

// QueryEscape экранирует строку адреса http запроса
func QueryEscape(s string) string {
	// экранировать url
	s = url.QueryEscape(s)
	// заменить вхождения "_2" на "_"
	s = regexp.MustCompile("_").ReplaceAllString(s, "_1")
	// заменить вхождения "%" на "_2"
	s = regexp.MustCompile("%").ReplaceAllString(s, "_2")
	return s
}

// FuncFileLine возвращает имя и номер строки исходного файла
func FuncFileLine() string {
	pc, fileName, fileLine, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	return fmt.Sprintf("%s[%s:%d]", funcName, fileName, fileLine)
}

// ErrorWithInfo - включить в описание ошибки имя функции,
// путь к исходному файлу  и номер строки в исходном файле
func ErrorWithInfo(err error) error {
	if err == nil {
		panic("ExtError nil")
	}
	pc, fileName, fileLine, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	return fmt.Errorf("%s[%s:%d], %v", funcName, fileName, fileLine, err)
}

// NewErrorWithInfo - создать объект Error, включающий имя функции,
// путь к исходному файлу  и номер строки в исходном файле
func NewErrorWithInfo(str string) error {
	pc, fileName, fileLine, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	return fmt.Errorf("%s[%s:%d], %s", funcName, fileName, fileLine, str)
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

// RandStringRunes generates a random string of a fixed length n
func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
