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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"crypto/x509"
	"sync"
	"encoding/hex"
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

type ApiJson struct {
	CanvasXMax uint32
	CanvasYMax uint32
	SvgStrings []string
}

var bTLeaves []*bTNode
var leavesLock = &sync.Mutex{}
var genesisBlockHash string

var bT = make(map[string]*bTNode)
var btLock = &sync.Mutex{}

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

	btLock.Lock()
	bT[pc.cur] = node
	btLock.Unlock()
	if len(blockHashes) == 0 {
		leavesLock.Lock()
		if !inList(node, bTLeaves) {
			// this is a new leaf node
			bTLeaves = append(bTLeaves, node)
		}
		leavesLock.Unlock()
	} else {
		// add all the children nodes to the todo
		for _, blockHash := range blockHashes {
			todo = append(todo, PC{parent: pc.cur, cur: blockHash})
		}
	}
	// recursive call
	buildTreeHelper(todo)
}

func inList(node *bTNode, list []*bTNode) bool {
	for _, n := range list {
		if node.hash == n.hash {
			return true
		}
	}
	return false
}

func updateTree() {
	var todo []PC
	for _, node := range bTLeaves {
		todo = append(todo, PC{parent: node.parent, cur: node.hash})
	}
	buildTreeHelper(todo)
}

func getShapes() []string {
	// find longest chain
	maxHeight := -1
	var head *bTNode
	for _, leaf := range bTLeaves {
		if leaf.height > maxHeight {
			maxHeight = leaf.height
			head = leaf
		}
	}

	// build up chain from genesis -> head (so can order svgs)
	var chain []*bTNode
	cur := head
	btLock.Lock()
	for cur != nil {
		chain = append(chain, cur)
		cur, _ = bT[cur.parent]
	}
	btLock.Unlock()

	// reverse order of chain
	// from https://stackoverflow.com/questions/19239449/how-do-i-reverse-an-array-in-go
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	// build up list of shapes
	var svgStrings []string
	for _, node := range chain {
		shapeHashes, err := canvas.GetShapes(node.hash)
		if checkError(err) != nil {
			continue
		}

		for _, shapeHash := range shapeHashes {
			svgString, err := canvas.GetSvgString(shapeHash)
			if checkError(err) != nil {
				continue
			}

			svgStrings = append(svgStrings, svgString)
		}
	}

	return svgStrings
}

func pollTree() {
	// build up blockchain
	var err error
	genesisBlockHash, err = canvas.GetGenesisBlock()
	if checkError(err) != nil {
		return
	}

	buildTree()

	for {
		// constantly try to update tree
		updateTree()
	}
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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		svgStrings := getShapes()
		fmt.Println(svgStrings)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		p := ApiJson{
			CanvasXMax: settings.CanvasXMax,
			CanvasYMax: settings.CanvasYMax,
			SvgStrings: svgStrings}
		json.NewEncoder(w).Encode(p)
	})

	go pollTree()

	log.Fatal(http.ListenAndServe(":8080", nil))
}
