package main

import (
	"github.com/biodebox/go-coge-cli/internal"
	"github.com/biodebox/go-coge-cli/internal/founder"
	"log"
	"os"
)

func main() {
	f, err := founder.NewFounder(`C:\Users\recyg\go\src\github.com\biodebox\go-coge-sql\cmd\test\main.go`)
	if err != nil {
		log.Fatal(err)
	}
	file, err := os.OpenFile(`C:\Users\recyg\go\src\github.com\biodebox\go-coge-sql\cmd\test\main_generated.go`, os.O_RDWR | os.O_CREATE | os.O_TRUNC, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	defer func() {
		if err := file.Close(); err != nil {

		}
	}()
	types, err := f.GetTypes(`Command`)
	if err != nil {
		log.Fatal(err)
	}
	commands, err := internal.ParseCommands(f.GetPackage(), types)
	if err != nil {
		log.Fatal(err)
	}
	_= commands
	for _, command := range commands {
		command.Writer = file
		command.FileSet = f.GetFileSet()
		if err := internal.Generate(command); err != nil {
			log.Fatalln(err)
		}
	}
}
