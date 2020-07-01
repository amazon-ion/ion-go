package main

import (
	"fmt"
	"math"
)

func main() {

	fmt.Println(math.Round(123456788.4))

	val := math.Round(123456788.5)
	fmt.Println(val)

	val2 := math.Round(9.9)
	fmt.Println(val2)
	fmt.Println(math.Round(123456788.6))

	fmt.Println("Hello, world.")
}
