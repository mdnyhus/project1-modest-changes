package main

import "fmt"
import "../../blockartlib"

func main() {
	fmt.Println("starting tests...")

	//blockartlib.SvgToShape("<path d=\"M10 10 H 90 V 90 H 10 Z\" fill=\"transparent\" stroke=\"black\"/>")
	shape , err := blockartlib.SvgToShape("M 250 50 l 150 300")
	fmt.Println("Created Shape")
	fmt.Println(shape)
	panic(err)
}