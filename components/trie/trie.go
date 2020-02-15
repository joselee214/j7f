package trie

import "sync"

// Trie Tree
type Trie struct {
	Root           *Node
	CheckWhiteList bool // 是否检查白名单
	WhitePrefix    *Trie
	WhiteSuffix    *Trie
}

// Node Trie tree node
type Node struct {
	Node *sync.Map
	End  bool
}

// NewTrie returns a Trie tree
func NewTrie() *Trie {
	t := new(Trie)
	t.Root = NewTrieNode()
	return t
}

// NewTrieNode return a *TrieNode
func NewTrieNode() *Node {
	n := new(Node)
	n.Node = new(sync.Map)
	n.End = false
	return n
}

// Add 添加一个敏感词(UTF-8的)到Trie树中
func (t *Trie) Add(keyword string) {
	//通过rune类型处理字符，得到字符串长度，而不是字符串底层占字节长度
	//unicode/utf8包提供了用utf8获取长度的方法 RuneCountInString(str)
	chars := []rune(keyword)

	if len(chars) == 0 {
		return
	}

	node := t.Root
	for _, char := range chars {
		v, _ := node.Node.LoadOrStore(char, NewTrieNode())
		if nodeChar, ok := v.(*Node); ok {
			node = nodeChar
		}
	}
	node.End = true

}

// Del 从Trie树中删除一个敏感词
func (t *Trie) Del(keyword string) {
	chars := []rune(keyword)
	if len(chars) == 0 {
		return
	}

	t.cycleDel(t.Root, chars, 0)
}

func (t *Trie) cycleDel(node *Node, chars []rune, index int) (shouldDel bool) {
	char := chars[index]
	l := len(chars)

	if n, ok := node.Node.Load(char); ok {
		if nodeChar, ok := n.(*Node); ok {
			if index+1 < l && nodeChar.End == false {
				shouldDel = t.cycleDel(nodeChar, chars, index+1)
				if shouldDel {
					node.Node.Delete(char)
				}
			} else if nodeChar.End {
				shouldDel = true
				node.Node.Delete(char)
			}
		}
	}

	return
}

// Query 查询敏感词
// 将text中在trie里的敏感字，替换为*号
// 返回结果: 是否有敏感字, 敏感字数组, 替换后的文本
func (t *Trie) Query(text string) (bool, []string, string) {
	found := []string{}
	chars := []rune(text)
	l := len(chars)
	if l == 0 {
		return false, found, text
	}

	var (
		i, j, jj int
	)

	node := t.Root
	for i = 0; i < l; i++ {
		if v, ok := node.Node.Load(chars[i]); !ok {
			continue
		} else {
			if nodeChar, ok := v.(*Node); ok {
				node = nodeChar
			}
		}

		jj = 0

		for j = i + 1; j < l; j++ {
			if vv, ok := node.Node.Load(chars[j]); !ok {
				if jj > 0 {
					if t.CheckWhiteList && t.isInWhiteList(found, chars, i, jj, l) {
						i = jj
					} else {
						found = t.replaceToAsterisk(found, chars, i, jj)
						i = jj
					}
				}
				break
			} else {
				if nodeChar, ok := vv.(*Node); ok {
					node = nodeChar
				}
			}

			if node.End {
				jj = j //还有子节点的情况, 记住上次找到的位置, 以匹配最大串 (eg: AV, AV女优)

				if j+1 == l { // 是最后节点或者最后一个字符, break
					if t.CheckWhiteList && t.isInWhiteList(found, chars, i, j, l) {
						i = j
						break

					} else {
						found = t.replaceToAsterisk(found, chars, i, j)
						i = j
						break
					}
				}
			}
		}
		node = t.Root
	}

	exist := false
	if len(found) > 0 {
		exist = true
	}

	return exist, found, string(chars)
}

func (t *Trie) isInWhiteList(found []string, chars []rune, i, j, length int) (inWhiteList bool) {
	inWhiteList = t.isInWhitePreffixList(found, chars, i, j, length)
	if !inWhiteList {
		inWhiteList = t.isInWhiteSuffixList(found, chars, i, j, length)
	}
	return
}

// 取前5个字去 前缀白名单中检查
func (t *Trie) isInWhitePreffixList(found []string, chars []rune, i, j, length int) (inWhiteList bool) {
	if i == 0 { // 第一个
		return
	}
	prefixPos := i - 4
	if prefixPos < 0 {
		prefixPos = 0
	}
	prefixWords := string(chars[prefixPos : i+1])
	exist, _, respChars := t.WhitePrefix.Query(prefixWords)
	if exist {
		tmp := []rune(respChars)
		if tmp[len(tmp)-1] == 42 { // 42代表'*',string(byte(42))输出'*'
			inWhiteList = true
		}
	}
	return
}

// 取后5个字去 后缀白名单中检查
func (t *Trie) isInWhiteSuffixList(found []string, chars []rune, i, j, length int) (inWhiteList bool) {
	if j+1 == length { // 最后一个字了
		return
	}

	suffixPos := j + 5
	if suffixPos > length {
		suffixPos = length
	}
	suffixWords := string(chars[j:suffixPos])
	exist, _, respChars := t.WhiteSuffix.Query(suffixWords)
	if exist {
		tmp := []rune(respChars)
		if tmp[0] == 42 {
			inWhiteList = true
		}
	}
	return
}

// 替换为*号
func (t *Trie) replaceToAsterisk(found []string, chars []rune, i, j int) []string {
	tmpFound := chars[i : j+1]
	found = append(found, string(tmpFound))
	for k := i; k <= j; k++ {
		chars[k] = 42 // *的rune为42
	}
	return found
}

// ReadAll 返回所有敏感词
func (t *Trie) ReadAll() (words []string) {
	words = []string{}
	words = t.cycleRead(t.Root, words, "")
	return
}

func (t *Trie) cycleRead(node *Node, words []string, parentWord string) []string {
	node.Node.Range(func(char, n interface{}) bool {
		if nodeChar, ok := n.(*Node); ok {
			if nodeChar.End {
				words = append(words, parentWord+string(char.(int32)))
			}
			words = t.cycleRead(nodeChar, words, parentWord+string(char.(int32)))
		}
		return true
	})
	return words
}
