package main

import (
	"github.com/gen2brain/beeep"
)

func main() {
	err := beeep.Notify("Title", "Message body", "assets/information.png")
	if err != nil {
		panic(err)
	}
}
