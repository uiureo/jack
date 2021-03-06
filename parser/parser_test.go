package parser

import (
	"testing"

	"github.com/uiureo/jack/tokenizer"
)

func parse(source string) *Node {
	node, _ := ParseStatements(tokenizer.Tokenize(source))
	return node
}

func TestParseLetStatement(t *testing.T) {
	root := parse(`
    let city = "Paris";
    let bar = Foo.new();
  `)

	if !(root.Name == "statements" && root.Children[0].Name == "letStatement") {
		t.Errorf("expect node to have: letStatement, but got: \n%v", root.ToXML())
	}
}

func TestParseLetStatementWithArrayIndex(t *testing.T) {
	node, tokens := parseLetStatement(tokenizer.Tokenize(`let a[2]="foo";`))

	if node.Name != "letStatement" {
		t.Errorf("expect: letStatement, actual: %v", node.ToXML())
	}

	if len(tokens) > 0 {
		t.Error("parse failed")
	}
}

func TestParseIfStatement(t *testing.T) {
	root := parse(`
if (x > 153) {
  let city="Paris";
}
`)

	if !(root.Name == "statements" && root.Children[0].Name == "ifStatement") {
		t.Errorf("expect node to have: ifStatement, but got: \n%v", root.ToXML())
	}
}

func TestParseIfElseStatement(t *testing.T) {
	root := parse(`
if (x > 153) {
  let city="Paris";
} else {
  let city="Osaka";
}
`)
	statement := root.Children[0]

	if !(root.Name == "statements" && statement.Name == "ifStatement") {
		t.Errorf("expect node to have: ifStatement, but got: \n%v", root.ToXML())
	}

	found := false
	for _, node := range statement.Children {
		if node.Name == "keyword" && node.Value == "else" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("expect node to have \"else\" keyword\n%v", root.ToXML())
	}
}

func TestParseStatements(t *testing.T) {
	root := parse(`
let foo="foo";
let bar="bar";
`)

	if !(root.Name == "statements" && len(root.Children) == 2) {
		t.Errorf("expect statements, got: \n%v", root.ToXML())
	}
}

func TestParseWhileStatement(t *testing.T) {
	root := parse(`
    while (i > 100) {
      let foo=0;
    }
  `)

	if len(root.Children) == 0 {
		t.Errorf("expect node to have children, but got:\n%v", root.ToXML())
		return
	}

	statement := root.Children[0]

	if statement.Name != "whileStatement" {
		t.Errorf("expect node to have whileStatement, but got:\n%v", root.ToXML())
	}
}

func TestParseDoStatement(t *testing.T) {
	root := parse(`
    do foo(1, 2, 3);
  `)

	if len(root.Children) == 0 {
		t.Errorf("expect node to have children, but got:\n%v", root.ToXML())
		return
	}

	statement := root.Children[0]

	if statement.Name != "doStatement" {
		t.Errorf("expect node to have whileStatement, but got:\n%v", root.ToXML())
	}
}

func TestParseReturnStatement(t *testing.T) {
	root := parse(`return 1 + 2;`)

	if len(root.Children) == 0 {
		t.Errorf("expect node to have children, but got:\n%v", root.ToXML())
		return
	}

	statement := root.Children[0]

	if statement.Name != "returnStatement" {
		t.Errorf("expect node to have whileStatement, but got:\n%v", root.ToXML())
	}
}

func TestParseClass(t *testing.T) {
	root, tokens := parseClass(tokenizer.Tokenize(`
		class Main {
			function void main() {
				return;
			}
		}
	`))

	if root.Name != "class" {
		t.Errorf("expect node `<class>`, but got:\n%v", root.ToXML())
	}

	node, i := root.Find(&Node{Name: "keyword", Value: "class"})
	if !(node != nil && i == 0) {
		t.Errorf("expect node to have class keyword, but got:\n%v", root.ToXML())
	}

	if node, _ := root.Find(&Node{Name: "subroutineDec"}); node == nil {
		t.Errorf("expect node to have subroutineDec, but got:\n%v", root.ToXML())
	}

	if len(tokens) > 0 {
		t.Errorf("expect len(tokens) == 0, but actual: %v", len(tokens))
	}
}

func TestParseTerm(t *testing.T) {
	testParseTermSuccess(t, `42`)
	testParseTermSuccess(t, `"foo"`)
	testParseTermSuccess(t, `null`)
	testParseTermSuccess(t, `bar`)
	testParseTermSuccess(t, `foo[1+2]`)
	testParseTermSuccess(t, `(1 + 2)`)
	testParseTermSuccess(t, `-123`)
}

func testParseTermSuccess(t *testing.T, source string) (*Node, []*tokenizer.Token) {
	root, tokens := parseTerm(tokenizer.Tokenize(source))

	if len(tokens) > 0 {
		t.Errorf("`%s`: expect len(tokens) == 0, but actual: %v", source, len(tokens))
	}

	if root.Name != "term" {
		t.Errorf("`%s`: expect node to be term, but actual: %s", source, root.ToXML())
	}

	return root, tokens
}

func TestParseClassWithField(t *testing.T) {
	root, tokens := parseClass(tokenizer.Tokenize(`
		class Main {
			field int x, y;
			static int size;

			function void main() {
				return;
			}
		}
	`))

	if node, _ := root.Find(&Node{Name: "classVarDec"}); node == nil {
		t.Errorf("expect node to have classVarDec, but got:\n%v", root.ToXML())
	}

	if len(tokens) > 0 {
		t.Errorf("expect len(tokens) == 0, but actual: %v", len(tokens))
	}
}

func TestParseClassWithMethod(t *testing.T) {
	root, tokens := parseClass(tokenizer.Tokenize(`
		class Foo {
			constructor Foo new() {
				return;
			}

			method boolean bar() {
				return true;
			}
		}
	`))

	if node, _ := root.Find(&Node{Name: "subroutineDec"}); node == nil {
		t.Errorf("expect node to have classVarDec, but got:\n%v", root.ToXML())
	}

	if len(tokens) > 0 {
		t.Errorf("expect len(tokens) == 0, but actual: %v", len(tokens))
	}
}

func TestParseVarDec(t *testing.T) {
	node, tokens := parseVarDec(tokenizer.Tokenize(`var int i, sum;`))

	if node.Name != "varDec" {
		t.Errorf("expect Name:`varDec` but actual: %v", node.Name)
	}

	if len(tokens) > 0 {
		t.Error("parse fails")
	}
}
