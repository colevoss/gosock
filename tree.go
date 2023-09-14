package gosock

import (
	"encoding/json"
	"log"
)

type NodeType uint8

const (
	StaticNode NodeType = iota
	ParamNode
)

type Node struct {
	Path     string
	NodeType NodeType
	Channel  *Router
	Children []*Node
}

func NewTree() *Node {
	return &Node{
		Path:     "",
		NodeType: StaticNode,
	}
}

func NewNode(key string, nodeType NodeType) *Node {
	return &Node{
		Path:     key,
		NodeType: nodeType,
		// dynamically create chidlren
		// children
	}
}

// TODO: Add case for duplicate keys
func (n *Node) Add(key string, mc *Router) {
	root := n

walk:
	for {
		i := findLongestCommonPrefix(root.Path, key)

		/**
			 * If new key has match less than this path's length, split this
			 * node at the new key's match index and create a child
		   *
			 * n.Path = "test"
			 * key    = "tes"
		   * sets this to "tes"
		*/
		if i < len(root.Path) {
			originalPath := root.Path

			newRootPath := originalPath[:i]
			root.Path = newRootPath

			newChildPath := originalPath[i:]

			node := NewNode(newChildPath, StaticNode)
			node.Children = root.Children
			node.Channel = root.Channel
			root.Channel = nil
			root.Children = []*Node{node}
		}

		// If new key matches this path but is longer add as child
		if len(key) > i {
			key = key[i:]

			if root.IsLeaf() {
				root.add(key, mc)
				return
			}

			if root.IsParam() {
				root = root.Children[0]
				continue walk
			}

			for _, ch := range root.Children {
				if ch.Path[0] == key[0] {
					root = ch
					continue walk
				}
			}

			root.add(key, mc)

			return
		}
	}
}

func (n *Node) Lookup(key string) (root *Node, params *Params) {
	root = n

walk:
	for {
		if root.IsParam() {
			end := 0

			for end < len(key) && key[end] != '.' {
				end++
			}

			val := key[:end]
			key = key[end:]

			if params == nil {
				// TODO: pool params
				params = &Params{}
			}

			params.Add(root.GetParamName(), val)
		} else {
			i := findLongestCommonPrefix(root.Path, key)

			if i < len(root.Path) {
				return nil, nil
			}

			if i == len(key) {
				return root, params
			}

			key = key[i:]
		}

		// I think we need this.
		// If key is used up and we aren't on a leaf
		// we shouldn't continue
		// if len(key) == 0 {
		// 	return root, params
		// }

		for _, child := range root.Children {
			if child.IsParam() || child.Path[0] == key[0] {
				root = child
				continue walk
			}
			//
			// if child.Path[0] == key[0] {
			// 	root = child
			// 	continue walk
			// }
		}

		// If we have gotten this far then there is not match in any of the children
		break walk
	}

	return nil, nil
}

func (n *Node) EmptyPath() bool {
	return n.Path == ""
}

func (n *Node) IsLeaf() bool {
	if n.Children == nil {
		return true
	}

	if len(n.Children) == 0 {
		return true
	}

	return false
}

func BetterGetParam(path string) (start int, end int, paramName string) {
	paramName = ""

	for start, c := range []byte(path) {
		if c != '{' {
			continue
		}

		paramName = paramName + "{"

		for end, c := range []byte(path[start+1:]) {
			if c == '}' {
				paramName = paramName + "}"
				return start, start + end + 2, paramName
			}

			paramName = paramName + string(c)
		}
	}

	return -1, -1, paramName
}

func (n *Node) GetParamName() string {
	if !n.IsParam() {
		return ""
	}

	return n.Path[1 : len(n.Path)-1]
}

func GetParam(path string) (start int, end int, paramName string) {
	for start, c := range []byte(path) {
		// Should not start looking for a wildcard
		if c != '.' {
			continue
		}

		// else check if {wildcard} starts

		paramStarted := false
		paramName := ""
		paramEnded := false

		for end, wildCardStarter := range []byte(path[start+1:]) {
			if !paramStarted && wildCardStarter == '{' {
				paramStarted = true
				continue
			}

			if paramStarted && wildCardStarter == '}' {
				if len(paramName) == 0 {
					panic("Param must have a name")
				}

				paramEnded = true
				continue
			}

			if paramEnded {
				// Param has ended so make sure it ends with a period
				if wildCardStarter == '.' {
					return start, start + end + 1, paramName
				}

				// Param is not valid if not followed by a .
				// Probably return a `valid` return argument instead of panicing
				panic("Param must be entire segment of path")
			}

			paramName = paramName + string(wildCardStarter)
		}

		return start, len(path), paramName
	}

	return -1, -1, ""
}

func (n *Node) add(path string, mc *Router) {
	root := n
	key := path

	for {
		start, end, param := BetterGetParam(key)

		if start < 0 {
			break
		}

		// param does not start at beginning so create a parent node
		if start > 0 {
			parent := &Node{
				Path:     key[:start],
				NodeType: StaticNode,
			}

			root.addNode(parent)
			root = parent
			key = key[start:]
			continue
		}

		if start == 0 {
			paramNode := &Node{
				Path:     param,
				NodeType: ParamNode,
			}

			root.addNode(paramNode)

			root = paramNode
			key = key[end:]

			continue
		}
	}

	node := &Node{
		Path:     key,
		NodeType: StaticNode,
		Channel:  mc,
	}

	root.addNode(node)
}

func (n *Node) addNode(node *Node) {
	n.ensureChildren()

	n.Children = append(n.Children, node)
}

func (n *Node) ensureChildren() {
	if n.Children != nil {
		return
	}

	n.Children = make([]*Node, 0)
}

func (n *Node) IsStatic() bool {
	return n.NodeType == StaticNode
}

func (n *Node) IsParam() bool {
	return n.NodeType == ParamNode
}

func (n *Node) Print() {
	str, _ := json.MarshalIndent(n, "", "  ")

	log.Printf(string(str))
}

func findLongestCommonPrefix(a string, b string) int {
	i := 0

	max := min(len(a), len(b))

	for i < max && a[i] == b[i] {
		i++
	}

	return i
}
