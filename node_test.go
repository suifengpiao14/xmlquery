package xmlquery

import (
	"html"
	"reflect"
	"strings"
	"testing"
)

func findNode(root *Node, name string) *Node {
	node := root.FirstChild
	for {
		if node == nil || node.Data == name {
			break
		}
		node = node.NextSibling
	}
	return node
}

func childNodes(root *Node, name string) []*Node {
	var list []*Node
	node := root.FirstChild
	for {
		if node == nil {
			break
		}
		if node.Data == name {
			list = append(list, node)
		}
		node = node.NextSibling
	}
	return list
}

func testNode(t *testing.T, n *Node, expected string) {
	if n.Data != expected {
		t.Fatalf("expected node name is %s,but got %s", expected, n.Data)
	}
}

func testAttr(t *testing.T, n *Node, name, expected string) {
	for _, attr := range n.Attr {
		if attr.Name.Local == name && attr.Value == expected {
			return
		}
	}
	t.Fatalf("not found attribute %s in the node %s", name, n.Data)
}

func testValue(t *testing.T, val, expected interface{}) {
	if val == expected {
		return
	}
	if reflect.DeepEqual(val, expected) {
		return
	}
	t.Fatalf("expected value is %+v, but got %+v", expected, val)
}

func testTrue(t *testing.T, v bool) {
	if v {
		return
	}
	t.Fatal("expected value is true, but got false")
}

// Given a *Node, verify that all the pointers (parent, first child, next sibling, etc.) of
// - the node itself,
// - all its child nodes, and
// - pointers along the silbling chain
// are valid.
func verifyNodePointers(t *testing.T, n *Node) {
	if n == nil {
		return
	}
	if n.FirstChild != nil {
		testValue(t, n, n.FirstChild.Parent)
	}
	if n.LastChild != nil {
		testValue(t, n, n.LastChild.Parent)
	}

	verifyNodePointers(t, n.FirstChild)
	// There is no need to call verifyNodePointers(t, n.LastChild)
	// because verifyNodePointers(t, n.FirstChild) will traverse all its
	// siblings to the end, and if the last one isn't n.LastChild then it will fail.

	parent := n.Parent // parent could be nil if n is the root of a tree.

	// Verify the PrevSibling chain
	cur, prev := n, n.PrevSibling
	for ; prev != nil; cur, prev = prev, prev.PrevSibling {
		testValue(t, prev.Parent, parent)
		testValue(t, prev.NextSibling, cur)
	}
	testTrue(t, cur.PrevSibling == nil)
	testTrue(t, parent == nil || parent.FirstChild == cur)

	// Verify the NextSibling chain
	cur, next := n, n.NextSibling
	for ; next != nil; cur, next = next, next.NextSibling {
		testValue(t, next.Parent, parent)
		testValue(t, next.PrevSibling, cur)
	}
	testTrue(t, cur.NextSibling == nil)
	testTrue(t, parent == nil || parent.LastChild == cur)
}

func TestRemoveFromTree(t *testing.T) {
	xml := `<?procinst?>
		<!--comment-->
		<aaa><bbb/>
			<ddd><eee><fff/></eee></ddd>
		<ggg/></aaa>`
	parseXML := func() *Node {
		doc, err := Parse(strings.NewReader(xml))
		testTrue(t, err == nil)
		return doc
	}

	t.Run("remove an elem node that is the only child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/ddd/eee")
		testTrue(t, n != nil)
		removeFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ddd></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove an elem node that is the first but not the last child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/bbb")
		testTrue(t, n != nil)
		removeFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove an elem node that is neither the first nor  the last child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/ddd")
		testTrue(t, n != nil)
		removeFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ggg></ggg></aaa>`)
	})

	t.Run("remove an elem node that is the last but not the first child of its parent", func(t *testing.T) {
		doc := parseXML()
		n := FindOne(doc, "//aaa/ggg")
		testTrue(t, n != nil)
		removeFromTree(n)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd></aaa>`)
	})

	t.Run("remove decl node works", func(t *testing.T) {
		doc := parseXML()
		procInst := doc.FirstChild
		testValue(t, procInst.Type, DeclarationNode)
		removeFromTree(procInst)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<!--comment--><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove comment node works", func(t *testing.T) {
		doc := parseXML()
		commentNode := doc.FirstChild.NextSibling.NextSibling // First .NextSibling is an empty text node.
		testValue(t, commentNode.Type, CommentNode)
		removeFromTree(commentNode)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})

	t.Run("remove call on root does nothing", func(t *testing.T) {
		doc := parseXML()
		removeFromTree(doc)
		verifyNodePointers(t, doc)
		testValue(t, doc.OutputXML(false),
			`<?procinst?><!--comment--><aaa><bbb></bbb><ddd><eee><fff></fff></eee></ddd><ggg></ggg></aaa>`)
	})
}

func TestSelectElement(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?>
    <AAA>
        <BBB id="1"/>
        <CCC id="2">
            <DDD/>
        </CCC>
		<CCC id="3">
            <DDD/>
        </CCC>
     </AAA>`
	root, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Error(err)
	}
	version := root.FirstChild.SelectAttr("version")
	if version != "1.0" {
		t.Fatal("version!=1.0")
	}
	aaa := findNode(root, "AAA")
	var n *Node
	n = aaa.SelectElement("BBB")
	if n == nil {
		t.Fatalf("n is nil")
	}
	n = aaa.SelectElement("CCC")
	if n == nil {
		t.Fatalf("n is nil")
	}

	var ns []*Node
	ns = aaa.SelectElements("CCC")
	if len(ns) != 2 {
		t.Fatalf("len(ns)!=2")
	}
}

func TestEscapeOutputValue(t *testing.T) {
	data := `<AAA>&lt;*&gt;</AAA>`

	root, err := Parse(strings.NewReader(data))
	if err != nil {
		t.Error(err)
	}

	escapedInnerText := root.OutputXML(true)
	if !strings.Contains(escapedInnerText, "&lt;*&gt;") {
		t.Fatal("Inner Text has not been escaped")
	}

}

func TestOutputXMLWithNamespacePrefix(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8"?><S:Envelope xmlns:S="http://schemas.xmlsoap.org/soap/envelope/"><S:Body></S:Body></S:Envelope>`
	doc, _ := Parse(strings.NewReader(s))
	if s != doc.OutputXML(false) {
		t.Fatal("xml document missing some characters")
	}
}

func TestOutputXMLWithCommentNode(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<!-- Students grades are updated bi-monthly -->
	<class_list>
		<student>
			<name>Robert</name>
			<grade>A+</grade>
		</student>
	<!--
		<student>
			<name>Lenard</name>
			<grade>A-</grade>
		</student>
	-->
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))
	if e, g := "<!-- Students grades are updated bi-monthly -->", doc.OutputXML(true); strings.Index(g, e) == -1 {
		t.Fatal("missing some comment-node.")
	}
	n := FindOne(doc, "//class_list")
	t.Log(n.OutputXML(false))
	if e, g := "<name>Lenard</name>", n.OutputXML(false); strings.Index(g, e) == -1 {
		t.Fatal("missing some comment-node")
	}
}

func TestOutputXMLWithSpaceParent(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student xml:space="preserve">
			<name> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	n := FindOne(doc, "/class_list/student/name")
	expected := "<name> Robert </name>"
	if g := doc.OutputXML(true); strings.Index(g, expected) == -1 {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	output := html.UnescapeString(doc.OutputXML(true))
	if strings.Contains(output, "\n") {
		t.Errorf("the outputted xml contains newlines")
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithSpaceDirect(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student>
			<name xml:space="preserve"> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	n := FindOne(doc, "/class_list/student/name")
	expected := `<name xml:space="preserve"> Robert </name>`
	if g := doc.OutputXML(false); strings.Index(g, expected) == -1 {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	output := html.UnescapeString(doc.OutputXML(true))
	if strings.Contains(output, "\n") {
		t.Errorf("the outputted xml contains newlines")
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithSpaceOverwrittenToPreserve(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student xml:space="default">
			<name xml:space="preserve"> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	n := FindOne(doc, "/class_list/student")
	expected := `<name xml:space="preserve"> Robert </name>`
	if g := n.OutputXML(false); strings.Index(g, expected) == -1 {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	output := html.UnescapeString(doc.OutputXML(true))
	if strings.Contains(output, "\n") {
		t.Errorf("the outputted xml contains newlines")
	}
	t.Log(n.OutputXML(false))
}

func TestOutputXMLWithSpaceOverwrittenToDefault(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8"?>
	<class_list>
		<student xml:space="preserve">
			<name xml:space="default"> Robert </name>
			<grade>A+</grade>
		</student>
	</class_list>`
	doc, _ := Parse(strings.NewReader(s))
	t.Log(doc.OutputXML(true))

	n := FindOne(doc, "/class_list/student")
	expected := `<name xml:space="default">Robert</name>`
	if g := doc.OutputXML(false); strings.Index(g, expected) == -1 {
		t.Errorf(`expected "%s", obtained "%s"`, expected, g)
	}

	output := html.UnescapeString(doc.OutputXML(true))
	if strings.Contains(output, "\n") {
		t.Errorf("the outputted xml contains newlines")
	}
	t.Log(n.OutputXML(false))
}
