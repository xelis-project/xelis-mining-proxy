package log

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/go-color"
)

var LogLevel uint8 = 0

func prefix() string {
	x := time.Now().Local().Format("2006-01-02 15:04:05 ")

	if LogLevel != 0 {
		_, file, line, _ := runtime.Caller(2)
		fileSpl := strings.Split(file, "/")
		debugInfos := strings.Split(fileSpl[len(fileSpl)-1], ".")[0] + ":" + strconv.FormatInt(int64(line), 10)
		for len(debugInfos) < 20 {
			debugInfos = debugInfos + " "
		}
		x += debugInfos
	}

	return x
}

func Title(a ...any) {
	fmt.Print("  " + fmt.Sprintln(a...))
}

func Info(a ...any) {
	fmt.Print(prefix() + "I " + fmt.Sprintln(a...))
}
func Infof(format string, a ...any) {
	fmt.Print(prefix() + fmt.Sprintf("I "+format+"\n", a...))
}

func Warn(a ...any) {
	fmt.Print(prefix() + color.Ize(color.Yellow, "W "+fmt.Sprintln(a...)))
}
func Warnf(format string, a ...any) {
	fmt.Print(prefix() + color.Ize(color.Yellow, fmt.Sprintf("W "+format+"\n", a...)))
}

func Err(a ...any) {
	fmt.Print(prefix() + color.Ize(color.Red, "E "+fmt.Sprintln(a...)))
}

func Errf(format string, a ...any) {
	fmt.Print(prefix() + color.Ize(color.Red, fmt.Sprintf("E "+format+"\n", a...)))
}

func Fatal(a ...any) {
	fmt.Print(prefix() + color.Ize(color.Red, "E "+fmt.Sprintln(a...)))
	panic(a)
}

func Debug(a ...any) {
	if LogLevel < 1 {
		return
	}

	fmt.Print(prefix() + color.Ize(color.Cyan, "D "+fmt.Sprintln(a...)))
}
func Debugf(format string, a ...any) {
	if LogLevel < 1 {
		return
	}

	fmt.Print(prefix() + color.Ize(color.Cyan, fmt.Sprintf("D "+format+"\n", a...)))
}
