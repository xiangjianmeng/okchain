// Copyright The go-okchain Authors 2018,  All rights reserved.

package merkel

import (
	"crypto/sha256"
	"strings"
)

type MerkleTree struct {
	RootNode *MerkleNode
}

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

//创建一个新的节点
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil {
		//叶子节点
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}

//make a merkel tree
func NewMerkleTree(data [][]byte) *MerkleTree {
	if len(data) == 0 {
		return &MerkleTree{RootNode: &MerkleNode{
			Left:  nil,
			Right: nil,
			Data:  []byte(strings.Repeat("0", 32)),
		}}
	}
	var nodes []MerkleNode

	//even node numbers
	if len(data)%2 != 0 {
		data = append(data, data[len(data)-1])
	}

	//leaf node, left=right=nil, data stores the hashes, such as txs etc.
	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, *node)
	}
	// fmt.Printf("nodes.len=%d, nodes=%+v\n\n", len(nodes), nodes)

	//make node from bottom to top until root node
	for i := 0; i < len(data)/2; i++ {
		// fmt.Println("data index=", i)
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 {
			// fmt.Println("nodes index=", j, j+1, " nodes/newLevel len=", len(nodes))
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, *node)
		}
		nodes = newLevel
		if len(nodes)%2 != 0 {
			nodes = append(nodes, nodes[len(nodes)-1])
		}
	}

	mTree := MerkleTree{&nodes[0]}

	return &mTree

}
