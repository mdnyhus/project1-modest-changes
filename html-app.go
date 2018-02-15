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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
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

func buildTree() {
	buildTreeHelper([]PC{PC{parent: "", cur: genesisBlockHash}})
}

func buildTreeHelper(todo []PC) {
	if len(todo) <= 0 {
		// done!
		return
	}

	// pop off todo
	var pc PC
	pc, todo = todo[0], todo[1:]

	var height int
	if pc.parent == "" {
		// genesis block
		height = 0
	} else {
		var ok bool
		parent, ok := bT[pc.parent]
		if !ok {
			// this element is invalid
			return
		}
		height = parent.height + 1
	}

	blockHashes, err := canvas.GetChildren(pc.cur)
	if checkError(err) != nil {
		return
	}

	node := &bTNode{
		hash:     pc.cur,
		height:   height,
		parent:   pc.parent,
		children: blockHashes}

	bT[pc.cur] = node
	if len(blockHashes) == 0 {
		// this is a leaf node
		bTLeaves = append(bTLeaves, node)
	} else {
		// add all the children nodes to the todo
		for _, blockHash := range blockHashes {
			todo = append(todo, PC{parent: pc.cur, cur: blockHash})
		}
	}
	// recursive call
	buildTreeHelper(todo)
}

func main() {
	// TODO - make it take arguments
	minerAddr := "127.0.0.1:8080"
	//privKey := // TODO: use crypto/ecdsa to read pub/priv keys from a file argument.
	keyPointer, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	privKey := *keyPointer

	// Open a canvas.
	var err error
	canvas, settings, err = blockartlib.OpenCanvas(minerAddr, privKey)
	if checkError(err) != nil {
		return
	}

	// build up blockchain
	genesisBlockHash, err = canvas.GetGenesisBlock()
	if checkError(err) != nil {
		return
	}

	buildTree()
	// find longest chain
	maxHeight := -1
	var head *bTNode
	for _, leaf := range bTLeaves {
		if leaf.height > maxHeight {
			maxHeight = leaf.height
			head = leaf
		}
	}

	// build up list of shapes
	cur := head
	var svgStrings []string
	for cur != nil {
		shapeHashes, err := canvas.GetShapes(cur.hash)
		if checkError(err) != nil {
			return
		}

		for _, shapeHash := range shapeHashes {
			svgString, err := canvas.GetSvgString(shapeHash)
			if checkError(err) != nil {
				return
			}

			svgStrings = append(svgStrings, svgString)
		}

		cur, _ = bT[cur.parent]
	}

	// want shapes in order they were created, which is reverse of svgStrings
	// from https://stackoverflow.com/questions/19239449/how-do-i-reverse-an-array-in-go
	for i, j := 0, len(svgStrings)-1; i < j; i, j = i+1, j-1 {
		svgStrings[i], svgStrings[j] = svgStrings[j], svgStrings[i]
	}

	// write html file
	file, _ := os.OpenFile("./html-app.html", os.O_RDWR|os.O_CREATE, 0666)
	file.Write([]byte("<svg>\n"))

	for i := 0; i < len(svgStrings); i++ {
		fmt.Println(svgStrings[i])
		file.Write([]byte(svgStrings[i] + "\n"))
	}

	file.Write([]byte("<svg/>"))
	file.Sync()
	file.Close()
}
