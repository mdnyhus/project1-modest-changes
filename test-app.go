/*

An app that creates an html file based on the longest chain

Usage:
go run art-app.go
*/

package main

// Expects blockartlib.go to be in the ./blockartlib/ dir, relative to
// this art-app.go file
import (
	"./blockartlib"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"time"
)

var canvas blockartlib.Canvas
var settings blockartlib.CanvasSettings

type bTNode struct {
	hash        string
	shapeHashes []string
	height      int
	parent      string
	children    []string
}

type PC struct {
	parent string
	cur    string
}

var bTLeaves []*bTNode
var genesisBlockHash string

var bT = make(map[string]*bTNode)

// If error is non-nil, print it out and return it.
func checkError(err error) error {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error ", err.Error())
		return err
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run html-app.go [minerAddr ip:port] [privKey]")
		os.Exit(1)
	}

	minerAddr := os.Args[1]
	privKeyArg := os.Args[2]

	privKeyStr, err := hex.DecodeString(privKeyArg)
	if err != nil {
		panic(err)
	}
	privKeyParsed, err := x509.ParseECPrivateKey(privKeyStr)
	if err != nil {
		panic(err)
	}
	privKey := *privKeyParsed

	// Open a canvas.
	canvas, settings, err = blockartlib.OpenCanvas(minerAddr, privKey)
	if checkError(err) != nil {
		return
	}
	// _, _, inkRemaining, err := canvas.AddShape(3, blockartlib.PATH, "M 0 0 H 10 V 10 h -10 Z", "red", "blue")
	// fmt.Println(err)
	// fmt.Println(inkRemaining)

	// wait to get 100 ink
	var ink uint32
	iterations := 0
	for ink < uint32(49) {
		ink, err = canvas.GetInk()
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(ink)
		fmt.Println(iterations)
		iterations++
		time.Sleep(1000 * time.Millisecond)
	}

	// shapeHash, _, inkRemaining, err := canvas.AddShape(2, blockartlib.PATH, "M 0 0 H 5 V 5 h -5 Z", "red", "blue")
	// fmt.Println("ADD SHAPE DONE:")
	// fmt.Println(err)
	// fmt.Println(inkRemaining)
	// inkRemaining, err = canvas.DeleteShape(2, shapeHash)
	// fmt.Println("DELETE SHAPE DONE:")
	// fmt.Println(err)
	// fmt.Println(inkRemaining)
	fmt.Println("adding the shape")
	shapeHash, _, inkRemaining, err := canvas.AddShape(1, blockartlib.PATH, "M 0 0 H 5 V 5 h -5 Z", "red", "blue")
	fmt.Println(shapeHash)
	fmt.Println(inkRemaining)
	fmt.Println(err)
	shapeHash, _, inkRemaining, err = canvas.AddShape(1, blockartlib.PATH, "M 5 5 h 10 v 10 h -5 Z", "green", "blue")
	fmt.Println(shapeHash)
	fmt.Println(inkRemaining)
	fmt.Println(err)
	shapeHash, _, inkRemaining, err = canvas.AddShape(1, blockartlib.PATH, "M 10 10 h 5 v 5 h -5 Z", "yellow", "blue")
	fmt.Println(shapeHash)
	fmt.Println(inkRemaining)
	fmt.Println(err)
	fmt.Println("Congrats you are done... shapes are added")
}
