package main

import "fmt"
import "../../blockartlib"

func main() {
	fmt.Println("starting tests...")

	//shape , err := blockartlib.SvgToShape("M 250.121 50.3122 h -20.123 v -30.323 l 150.12 300.545")

	fmt.Println("Created Shape")
	fmt.Println(shape)
	if err != nil {
		panic(err)
	}


	blockartlib.PaintCanvas()

}