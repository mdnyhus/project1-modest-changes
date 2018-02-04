package main

import "fmt"
import "../../blockartlib"

func main() {
	fmt.Println("starting tests...")

	//blockartlib.SvgToShape("<path d=\"M10 10 H 90 V 90 H 10 Z\" fill=\"transparent\" stroke=\"black\"/>")
	blockartlib.SvgToShape("  <path id=\"lineBC\" d=\"M 250 50 l 150 300\" stroke=\"red\" stroke-width=\"3\" fill=\"none\"/>")
}