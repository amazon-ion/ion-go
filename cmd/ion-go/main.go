/*
 * Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package main

import (
	"errors"
	"fmt"
	"os"

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
		err = printVersion()

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
func printVersion() error {
	w := ion.NewTextWriterOpts(os.Stdout, ion.TextWriterPretty)

	if err := w.BeginStruct(); err != nil {
		return err
	}
	{
		if err := w.FieldName("version"); err != nil {
			return err
		}
		if err := w.WriteString(internal.GitCommit); err != nil {
			return err
		}

		buildtime, err := ion.NewTimestampFromStr(internal.BuildTime, ion.TimestampPrecisionSecond, ion.TimezoneUTC)
		if err == nil {
			if err := w.FieldName("build_time"); err != nil {
				return err
			}
			if err := w.WriteTimestamp(buildtime); err != nil {
				return err
			}
		} else {
			if err := w.FieldName("build_time"); err != nil {
				return err
			}
			if err := w.WriteString("unknown-buildtime"); err != nil {
				return err
			}
		}
	}
	if err := w.EndStruct(); err != nil {
		return err
	}

	if err := w.Finish(); err != nil {
		panic(err)
	}

	return nil
}
