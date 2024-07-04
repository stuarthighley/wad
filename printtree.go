package wad

import "fmt"

// PrintTree prints a binary tree in a clear format
func PrintTree(n *Node) {
	var printRecursive func(BSPMember, string)
	printRecursive = func(member BSPMember, prefix string) {
		switch v := member.(type) {
		case *SubSector:
			fmt.Printf(prefix+"- %v", v)
		case *Node:
			fmt.Println(prefix + "- " + fmt.Sprint(*v))
			printRecursive(v.ChildR, prefix+"   ")
			printRecursive(v.ChildL, prefix+"   ")
		}

		if member.BSPType() == BSPSubSector {
			fmt.Println(prefix + "- null")
			return
		}

	}

	printRecursive(n, "")
}
