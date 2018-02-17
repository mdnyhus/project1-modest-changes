package main

import (
	"./blockartlib"
	"bufio"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run cli.go [miner ip:port] [privKey]")
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

	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, privKey)
	if err != nil {
		fmt.Println("Failure opening canvas")
		panic(err)
	}

	fmt.Println("Welcome to the interactive client for blockartlib")
	fmt.Printf("Successfully communicating with miner with addr %s\n", minerAddr)
	fmt.Println("Your settings are:")
	fmt.Printf("\tCanvas Width: %d\n", settings.CanvasXMax)
	fmt.Printf("\tCanvas Height: %d\n\n", settings.CanvasYMax)
	fmt.Println("Commands:")
	// No cirlce for now, default to path.
	fmt.Println("\tAddShape [validateNum] [svgString] [fill] [stroke] [PATH | CIRCLE]")
	fmt.Println("\tGetSvgString [shapeHash]")
	fmt.Println("\tGetInk")
	fmt.Println("\tDeleteShape [validateNum] [shapeHash]")
	fmt.Println("\tGetShapes [blockHash]")
	fmt.Println("\tGetGensisBlock")
	fmt.Println("\tGetChildren [blockHash]")
	fmt.Println("\tCloseCanvas")
	fmt.Println("\tExit")

	for {
		err = nil
		reader := bufio.NewReader(os.Stdin)
		fmt.Print(" > ")
		text, _ := reader.ReadString('\n')
		words := strings.Fields(text)

		if len(words) == 0 {
			continue
		}

		switch words[0] {
		case "AddShape":
			if len(words) < 7 {
				fmt.Println("Bad args")
				fmt.Println("AddShapeUsage:")
				fmt.Println("\tAddShape [validateNum] [svgString] [fill] [stroke] [PATH | CIRCLE]")
				continue
			}

			validateNum, err := strconv.Atoi(words[1])
			svgString := strings.Join(words[2:len(words)-3], " ")
			fill := words[len(words) - 3]
			stroke := words[len(words) - 2]
			shapeTypeArg := words[len(words) - 1]

			if err != nil || (shapeTypeArg != "PATH" && shapeTypeArg != "CIRCLE") {
				fmt.Println("Bad args")
				fmt.Println("AddShapeUsage:")
				fmt.Println("\tAddShape [validateNum] [svgString] [fill] [stroke]")
				continue
			}

			var shapeType blockartlib.ShapeType
			if shapeTypeArg == "PATH" {
				shapeType = blockartlib.ShapeType(blockartlib.PATH)
			} else {
				shapeType = blockartlib.ShapeType(blockartlib.CIRCLE)
			}

			shapeHash, blockHash, inkRemaining, err := canvas.AddShape(uint8(validateNum), shapeType, svgString, fill, stroke)
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("shapeHash: %s\n", shapeHash)
			fmt.Printf("blockHash: %s\n", blockHash)
			fmt.Printf("inkRemaining: %d\n", inkRemaining)
		case "GetSvgString":
			if len(words) != 2 {
				fmt.Println("Bad args")
				fmt.Println("GetSvgStringUsage:")
				fmt.Println("\tGetSvgString [shapeHash]")
				continue
			}

			shapeHash := words[1]

			svgString, err := canvas.GetSvgString(shapeHash)
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("svgString: %s\n", svgString)
		case "GetInk":
			if len(words) != 1 {
				fmt.Println("Bad args")
				fmt.Println("GetInk Usage:")
				fmt.Println("\tGetInk")
				continue
			}

			inkRemaining, err := canvas.GetInk()
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("inkRemaining: %d\n", inkRemaining)
		case "DeleteShape":
			if len(words) != 3 {
				fmt.Println("Bad args")
				fmt.Println("DeleteShape Usage")
				fmt.Println("\tDeleteShape [validateNum] [shapeHash]")
				continue
			}

			validateNum, err := strconv.Atoi(words[1])
			shapeHash := words[2]

			if err != nil {
				fmt.Println("Bad args")
				fmt.Println("DeleteShape Usage")
				fmt.Println("\tDeleteShape [validateNum] [shapeHash]")
				continue
			}

			inkRemaining, err := canvas.DeleteShape(uint8(validateNum), shapeHash)
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("inkRemaining: %d\n", inkRemaining)
		case "GetShapes":
			if len(words) != 2 {
				fmt.Println("Bad args")
				fmt.Println("GetShapes Usage:")
				fmt.Println("\tGetShapes [blockHash]")
				continue
			}

			blockHash := words[1]

			shapeHashes, err := canvas.GetShapes(blockHash)
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("shapeHashes: %v\n", shapeHashes)
		case "GetGenesisBlock":
			if len(words) != 1 {
				fmt.Println("Bad args")
				fmt.Println("GetGenesisBlock Usage:")
				fmt.Println("\tGetGenesisBlock")
			}

			blockHash, err := canvas.GetGenesisBlock()
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("blockHash: %s\n", blockHash)
		case "GetChildren":
			if len(words) != 2 {
				fmt.Println("Bad args")
				fmt.Println("GetChildren Usage:")
				fmt.Println("\tGetChildren [blockHash]")
			}

			blockHash := words[1]

			blockHashes, err := canvas.GetChildren(blockHash)
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("blockHashes: %v\n", blockHashes)
		case "CloseCanvas":
			if len(words) != 1 {
				fmt.Println("Bad args")
				fmt.Println("CloseCanvas Usage:")
				fmt.Println("\tCloseCanvas")
			}

			inkRemaining, err := canvas.CloseCanvas()
			if err != nil {
				fmt.Println("========== ERROR ==========")
				fmt.Println(err)
				fmt.Println("==========  END  ==========")
				continue
			}

			fmt.Printf("inkRemaining: %d\n", inkRemaining)
		case "Exit":
			os.Exit(0)
		default:
			fmt.Println("Unrecognized command")
		}
	}
}
