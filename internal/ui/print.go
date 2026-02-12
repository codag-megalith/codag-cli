package ui

import (
	"fmt"
	"os"
)

func Success(msg string) {
	fmt.Println(Green.Render("✓") + " " + msg)
}

func Error(msg string) {
	fmt.Fprintln(os.Stderr, Red.Render("✗")+" "+msg)
}

func Warn(msg string) {
	fmt.Println(Yellow.Render("!") + " " + msg)
}

func Info(msg string) {
	fmt.Println(Cyan.Render("›") + " " + msg)
}

func Keyval(key, value string) {
	fmt.Printf("  %s  %s\n", Dim.Render(key+":"), value)
}

func Blank() {
	fmt.Println()
}

func CodeBlock(content string) {
	fmt.Println(CodeBlockStyle.Render(content))
}
