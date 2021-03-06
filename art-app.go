/*

A trivial application to illustrate how the blockartlib library can be
used from an application in project 1 for UBC CS 416 2017W2.

Usage:
go run art-app.go
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import "./blockartlib"

import "fmt"
import "os"
import "crypto/ecdsa"

import "crypto/elliptic"
import "crypto/rand"

func main() {
	minerAddr := "127.0.0.1:8080"
	//privKey := // TODO: use crypto/ecdsa to read pub/priv keys from a file argument.
	keyPointer, _ := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	privKey := *keyPointer

	// Open a canvas.
	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, privKey)
	if checkError(err) != nil {
		return
	}

	// TODO - only here to get rid of warnings of unused variables
	fmt.Println(settings)

	validateNum := uint8(2)

	// Add a line.
	shapeHash, blockHash, ink, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 0 5", "transparent", "red")
	if checkError(err) != nil {
		return
	}

	// TODO - only here to get rid of warnings of unused variables
	fmt.Println(blockHash)
	fmt.Println(ink)

	// Add another line.
	shapeHash2, blockHash2, ink2, err := canvas.AddShape(validateNum, blockartlib.PATH, "M 0 0 L 5 0", "transparent", "blue")
	if checkError(err) != nil {
		return
	}

	// TODO - only here to get rid of warnings of unused variables
	fmt.Println(shapeHash2)
	fmt.Println(blockHash2)
	fmt.Println(ink2)

	// Delete the first line.
	ink3, err := canvas.DeleteShape(validateNum, shapeHash)
	if checkError(err) != nil {
		return
	}

	// TODO - only here to get rid of warnings of unused variables
	fmt.Println(ink3)

	// assert ink3 > ink2

	// Close the canvas.
	ink4, err := canvas.CloseCanvas()
	if checkError(err) != nil {
		return
	}

	// TODO - only here to get rid of warnings of unused variables
	fmt.Println(ink4)
}

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}
