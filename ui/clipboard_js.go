//go:build js

package ui

import "syscall/js"

func copyToClipboard(text string) {
	js.Global().Get("navigator").Get("clipboard").Call("writeText", text)
}
