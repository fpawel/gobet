package utils

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"math/rand"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
	"math"
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
		panic("ErrorWithInfo nil")
	}
	_, fileName, fileLine, _ := runtime.Caller(1)

	text := fmt.Sprintf("[%s:%d]:%v",
		filepath.Base(fileName), fileLine, err)

	return errors.New(text)
}

// NewErrorWithInfo - создать объект Error, включающий имя функции,
// путь к исходному файлу  и номер строки в исходном файле
func NewErrorWithInfo(str string) error {
	_, fileName, fileLine, _ := runtime.Caller(1)

	text := fmt.Sprintf("[%s:%d], %s",
		filepath.Base(fileName), fileLine, str)
	return errors.New(text)
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

func GetBytesOfObject(data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GetHashCodeOfObject(data interface{}) uint64 {
	fnv32a := fnv.New64a()
	bytes, err := GetBytesOfObject(data)
	if err != nil {
		log.Fatal(err.Error())
	}
	fnv32a.Write(bytes)
	return fnv32a.Sum64()
}

func RoundFloat64(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func Float64ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(RoundFloat64(num * output)) / output
}


// HumanateBytes produces a human readable representation of an SI size.
// bytes(82854982) -> 83 MB
func HumanateBytes(s uint64) string {
	sizes := []string{"B", "kB", "MB", "GB", "TB", "PB", "EB"}
	return humanateBytes(s, 1000, sizes)
}

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanateBytes(s uint64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%d B", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
