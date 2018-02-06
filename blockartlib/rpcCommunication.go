/*

This file is part of the blockartlib package, and contains all the structs required for RPC calls between blockartlib and an
ink-miner. It is placed in this file for cleaner organization of code.

*/

package blockartlib

type AddShapeArgs struct {
	// ink-miner only sees internal representation of shapes, conversion is all done by blockartlib before RPC call
	Shape Shape
	ValidateNum uint8
}

type AddShapeReply struct {
	ShapeHash string
	BlockHash string
	InkRemaining uint32
	
	// RPC errors are all cast to a ServerError
	// So, store actual error here; nil indicates no error
	Error error
}