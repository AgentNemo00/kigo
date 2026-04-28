package handler

import (
	"golang.org/x/term"
	"os"
)

func GetScreenDimensions() (int, int) {
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0, 0
	}
	return width, height
}
