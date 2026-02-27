package service

import (
	"crypto/sha256"
	"encoding/hex"
)

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Hash  string
}

type MerkleTree struct {
	Root *MerkleNode
}

func calculateHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func NewMerkleNode(left, right *MerkleNode, data string) *MerkleNode {
	node := &MerkleNode{}

	if left == nil && right == nil {
		node.Hash = calculateHash(data)
	} else {
		leftHash := left.Hash
		rightHash := ""

		if right != nil {
			rightHash = right.Hash
		} else {
			rightHash = leftHash
		}
		node.Hash = calculateHash(leftHash + rightHash)
	}

	node.Left = left
	node.Right = right

	return node
}

func NewMerkleTree(keys []string) *MerkleTree {
	var nodes []*MerkleNode

	for _, key := range keys {
		nodes = append(nodes, NewMerkleNode(nil, nil, key))
	}

	if len(nodes) == 0 {
		return &MerkleTree{Root: nil}
	}

	for len(nodes) > 1 {
		var nextLevel []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			left := nodes[i]
			var right *MerkleNode

			if i+1 < len(nodes) {
				right = nodes[i+1]
			}

			parentNode := NewMerkleNode(left, right, "")
			nextLevel = append(nextLevel, parentNode)
		}
		nodes = nextLevel
	}

	return &MerkleTree{Root: nodes[0]}
}