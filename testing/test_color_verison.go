package main

import (
	"github.com/pterm/pterm"
	"github.com/pterm/pterm/putils"
)

func main() {
	pterm.DefaultCenter.Println("This text is centered!\nIt centeres the whole block by default.\nIn that way you can do stuff like this:")

	// Generate BigLetters
	s, _ := pterm.DefaultBigText.WithLetters(putils.LettersFromString("activeio rdmc")).Srender()
	pterm.DefaultCenter.Println(s) // Print BigLetters with the default CenterPrinter

	pterm.DefaultCenter.WithCenterEachLineSeparately().Println("This text is centered!\nBut each line is\ncentered\nseparately")
}
