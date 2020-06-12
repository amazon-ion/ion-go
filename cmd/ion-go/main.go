package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/amzn/ion-go/internal"
	"github.com/amzn/ion-go/ion"
)

// main is the main entry point for ion-go.
func main() {
	if len(os.Args) <= 1 {
		printHelp()
		return
	}

	var err error

	switch os.Args[1] {
	case "help", "--help", "-h":
		printHelp()

	case "version", "--version", "-v":
		printVersion()

	case "process":
		err = process(os.Args[2:])

	default:
		err = errors.New("unrecognized command \"" + os.Args[1] + "\"")
	}

	if err != nil {
		fmt.Println(err.Error())
		printHelp()
	}
}

// printHelp prints the help message for the program.
func printHelp() {
	fmt.Println("Usage:")
	fmt.Println("  ion-go help")
	fmt.Println("  ion-go version")
	fmt.Println("  ion-go process [args]")
	fmt.Println("  ion-go compare [args]")
	fmt.Println("  ion-go extract [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  help       Prints this help message.")
	fmt.Println("  version    Prints version information about this tool.")
	fmt.Println("  extract    Extracts symbols from the given inputs into a shared symbol table.")
	fmt.Println("  compare    Compares all inputs against all other inputs and writes out a ComparisonReport.")
	fmt.Println("  process    Reads the input file(s) and re-writes the contents in the specified format.")
}

// printVersion prints (in ion) the version info for this tool.
func printVersion() {
	w := ion.NewTextWriterOpts(os.Stdout, ion.TextWriterPretty)

	w.BeginStruct()
	{
		w.FieldName("version")
		w.WriteString(internal.GitCommit)

		buildtime, err := time.Parse(time.RFC3339, internal.BuildTime)
		if err == nil {
			w.FieldName("build_time")
			w.WriteTimestamp(buildtime)
		} else {
			w.FieldName("build_time")
			w.WriteString("unknown-buildtime")
		}
	}
	w.EndStruct()

	if err := w.Finish(); err != nil {
		panic(err)
	}
}
