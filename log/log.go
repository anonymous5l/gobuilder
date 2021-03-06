package log

import "fmt"

var DebugEnabled bool

func Log(params ...any) {
	fmt.Println(append([]any{"\u001B[34m"}, append(params, "\u001B[0m")...)...)
}

func Error(params ...any) {
	fmt.Println(append([]any{"\u001B[31mERR -"}, append(params, "\u001B[0m")...)...)
}

func Debug(params ...any) {
	if DebugEnabled {
		fmt.Println(append([]any{"\u001B[33mVER -"}, append(params, "\u001B[0m")...)...)
	}
}

func Warn(params ...any) {
	fmt.Println(append([]any{"\u001B[33mWRN -"}, append(params, "\u001B[0m")...)...)
}

func Ok(params ...any) {
	fmt.Println(append([]any{"\u001B[32m"}, append(params, "\u001B[0m")...)...)
}
