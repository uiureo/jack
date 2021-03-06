package compiler

import (
	"fmt"
	"strconv"

	"github.com/uiureo/jack/parser"
)

func Compile(node *parser.Node) string {
	result := ""
	table := buildSymbolTable(node, nil)

	className := node.Children[1].Value

	for _, node := range node.Children {
		if node.Name == "subroutineDec" {
			result += compileSubroutineDec(node, table, className)
		}
	}

	return result
}

func symbolToSegment(symbol *Symbol) string {
	if symbol.Kind == "field" {
		return "this"
	}

	return symbol.Kind
}

func pushSymbol(symbol *Symbol) string {
	return fmt.Sprintf("push %s %d\n", symbolToSegment(symbol), symbol.Number)
}

func popSymbol(symbol *Symbol) string {
	return fmt.Sprintf("pop %s %d\n", symbolToSegment(symbol), symbol.Number)
}

var labelCount = map[string]int{}

func compileSubroutineDec(node *parser.Node, classTable *SymbolTable, className string) string {
	labelCount = map[string]int{}
	result := ""

	table := buildSymbolTable(node, classTable)
	name := node.Children[2].Value

	localVarCount := 0
	for _, symbol := range table.Scopes[0] {
		if symbol.Kind == "local" {
			localVarCount++
		}
	}

	result += fmt.Sprintf("function %s.%s %d\n", className, name, localVarCount)

	subroutineType := node.Children[0].Value
	switch subroutineType {
	case "constructor":
		fieldCount := 0
		for _, symbol := range classTable.Scopes[0] {
			if symbol.Kind == "field" {
				fieldCount++
			}
		}

		result += fmt.Sprintf("push constant %d\n", fieldCount)
		result += "call Memory.alloc 1\n"
		result += "pop pointer 0\n"
	case "method":
		result += "push argument 0\n"
		result += "pop pointer 0\n"
	}

	subroutineBody, _ := node.Find(&parser.Node{Name: "subroutineBody"})
	statements, _ := subroutineBody.Find(&parser.Node{Name: "statements"})

	result += pushStatements(statements, table)

	return result
}

func uniqueLabel(base string) string {
	count := labelCount[base]
	labelCount[base]++

	return base + strconv.Itoa(count)
}

func pushStatements(statements *parser.Node, table *SymbolTable) string {
	result := ""

	for _, statement := range statements.Children {
		switch statement.Name {
		case "letStatement":
			identifier, _ := statement.Find(&parser.Node{Name: "identifier"})
			symbol := table.Get(identifier.Value)
			if symbol == nil {
				panic(fmt.Sprintf("variable `%v` is not defined: %v\n%v", identifier.Value, table.String(), statements.ToXML()))
			}

			bracket, _ := statement.Find(&parser.Node{Name: "symbol", Value: "["})
			if bracket != nil {
				expressions := statement.FindAll(&parser.Node{Name: "expression"})
				result += pushExpression(expressions[0], table)
				result += pushSymbol(symbol)
				result += "add\n"
				result += pushExpression(expressions[1], table)
				result += "pop temp 0\n"
				result += "pop pointer 1\n"
				result += "push temp 0\n"
				result += "pop that 0\n"
			} else {
				expression, _ := statement.Find(&parser.Node{Name: "expression"})
				result += pushExpression(expression, table)

				result += popSymbol(symbol)
			}

		case "doStatement":
			subroutineCall := &parser.Node{Name: "subroutineCall", Children: statement.Children[1 : len(statement.Children)-1]}
			result += compileSubroutineCall(subroutineCall, table)
			result += "pop temp 0\n"

		case "returnStatement":
			expression, _ := statement.Find(&parser.Node{Name: "expression"})

			if expression != nil {
				result += pushExpression(expression, table)
			} else {
				result += "push constant 0\n"
			}

			result += "return\n"
		case "ifStatement":
			ifExpression, _ := statement.Find(&parser.Node{Name: "expression"})
			ifStatementsList := statement.FindAll(&parser.Node{Name: "statements"})

			trueLabel := uniqueLabel("IF_TRUE")
			falseLabel := uniqueLabel("IF_FALSE")
			endLabel := uniqueLabel("IF_END")

			if len(ifStatementsList) > 1 {
				ifStatements, elseStatements := ifStatementsList[0], ifStatementsList[1]

				result += pushExpression(ifExpression, table)
				result += "if-goto " + trueLabel + "\n"
				result += "goto " + falseLabel + "\n"
				result += "label " + trueLabel + "\n"
				result += pushStatements(ifStatements, table)
				result += "goto " + endLabel + "\n"
				result += "label " + falseLabel + "\n"
				result += pushStatements(elseStatements, table)
				result += "label " + endLabel + "\n"
			} else {
				ifStatements := ifStatementsList[0]

				result += pushExpression(ifExpression, table)
				result += "if-goto " + trueLabel + "\n"
				result += "goto " + falseLabel + "\n"
				result += "label " + trueLabel + "\n"
				result += pushStatements(ifStatements, table)
				result += "label " + falseLabel + "\n"
			}

		case "whileStatement":
			expLabel := uniqueLabel("WHILE_EXP")
			endLabel := uniqueLabel("WHILE_END")

			result += "label " + expLabel + "\n"
			whileExpression, _ := statement.Find(&parser.Node{Name: "expression"})
			result += pushExpression(whileExpression, table)
			result += "not\n"
			result += "if-goto " + endLabel + "\n"

			whileBody, _ := statement.Find(&parser.Node{Name: "statements"})
			result += pushStatements(whileBody, table)
			result += "goto " + expLabel + "\n"
			result += "label " + endLabel + "\n"
		}
	}

	return result
}

func pushExpression(expression *parser.Node, table *SymbolTable) string {
	if expression == nil {
		panic("argument must not be nil")
	}

	if expression.Name != "expression" {
		panic(fmt.Sprintf("argument must be `expression`, but actual: %v", expression.ToXML()))
	}

	leftTerm, _ := expression.Find(&parser.Node{Name: "term"})

	result := compileTerm(leftTerm, table)

	if len(expression.Children) > 1 {
		operator := expression.Children[1]
		expression.Children = expression.Children[2:]
		result += pushExpression(expression, table)
		result += compileOperator(operator.Value)
	}

	return result
}

func compileOperator(operator string) string {
	switch operator {
	case "+":
		return "add\n"
	case "-":
		return "sub\n"
	case "*":
		return "call Math.multiply 2\n"
	case "/":
		return "call Math.divide 2\n"
	case "<":
		return "lt\n"
	case ">":
		return "gt\n"
	case "&":
		return "and\n"
	case "|":
		return "or\n"
	case "=":
		return "eq\n"
	default:
		return ""
	}
}

func compileUnaryOperator(operator string) string {
	switch operator {
	case "-":
		return "neg\n"
	case "~":
		return "not\n"
	default:
		return ""
	}
}

func compileTerm(term *parser.Node, table *SymbolTable) string {
	firstChild := term.Children[0]

	lastChild := term.Children[len(term.Children)-1]

	isSubroutineCall := !(firstChild.Name == "symbol" && firstChild.Value == "(") && (lastChild.Name == "symbol" && lastChild.Value == ")")

	if isSubroutineCall {
		return compileSubroutineCall(term, table)
	}

	switch firstChild.Name {
	case "integerConstant":
		return fmt.Sprintf("push constant %s\n", firstChild.Value)
	case "stringConstant":
		return pushString(firstChild.Value)
	case "keyword":
		switch firstChild.Value {
		case "true":
			return "push constant 0\nnot\n"
		case "false", "null":
			return "push constant 0\n"
		case "this":
			return "push pointer 0\n"
		}
	case "identifier":
		symbol := table.Get(firstChild.Value)

		bracket, _ := term.Find(&parser.Node{Name: "symbol", Value: "["})
		if bracket != nil {
			result := ""

			expression, _ := term.Find(&parser.Node{Name: "expression"})
			result += pushExpression(expression, table)
			result += pushSymbol(symbol)
			result += "add\n"
			result += "pop pointer 1\n"
			result += "push that 0\n"

			return result
		}

		return pushSymbol(symbol)

	case "symbol":
		switch firstChild.Value {
		case "(":
			expression, _ := term.Find(&parser.Node{Name: "expression"})
			return pushExpression(expression, table)
		case "-", "~":
			childTerm, _ := term.Find(&parser.Node{Name: "term"})
			return compileTerm(childTerm, table) + compileUnaryOperator(firstChild.Value)
		}
	}

	return ""
}

func pushString(str string) string {
	result := ""

	size := len(str)
	result += fmt.Sprintf("push constant %d\n", size)
	result += "call String.new 1\n"

	for _, ch := range str {
		result += fmt.Sprintf("push constant %d\n", rune(ch))
		result += "call String.appendChar 2\n"
	}

	return result
}

func compileSubroutineCall(node *parser.Node, table *SymbolTable) string {
	result := ""
	argSize := 0
	_, i := node.Find(&parser.Node{Name: "symbol", Value: "("})

	var functionName string
	if i == 1 {
		subroutineName := node.Children[0].Value
		thisClassName := table.Find(&Symbol{Kind: "class"}).SymbolType

		functionName = fmt.Sprintf("%s.%s", thisClassName, subroutineName)

		result += "push pointer 0\n"
		argSize++
	} else if i == 3 {
		classOrVarName := node.Children[0].Value

		var className string
		if symbol := table.Get(classOrVarName); symbol != nil && symbol.Kind != "class" {
			className = symbol.SymbolType
			argSize++

			result += pushSymbol(symbol)
		} else {
			className = classOrVarName
		}

		subroutineName := node.Children[2].Value

		functionName = fmt.Sprintf("%s.%s", className, subroutineName)
	}

	expressionList, _ := node.Find(&parser.Node{Name: "expressionList"})

	expressions := expressionList.FindAll(&parser.Node{Name: "expression"})

	for _, expression := range expressions {
		result += pushExpression(expression, table)
	}

	argSize += len(expressions)
	result += fmt.Sprintf("call %s %d\n", functionName, argSize)

	return result
}

func buildSymbolTable(node *parser.Node, base *SymbolTable) *SymbolTable {
	if base == nil {
		base = &SymbolTable{}
	}

	table := &SymbolTable{}

	currentScope := map[string]*Symbol{}

	baseScopes := make([]map[string]*Symbol, len(base.Scopes))
	copy(baseScopes, base.Scopes)

	table.Scopes = append([]map[string]*Symbol{currentScope}, baseScopes...)

	switch node.Name {
	case "class":
		identifier, _ := node.Find(&parser.Node{Name: "identifier"})
		className := identifier.Value

		table.Set(className, &Symbol{Kind: "class", SymbolType: className})

		for _, node := range node.Children {
			if node.Name == "classVarDec" {
				kind := node.Children[0].Value
				symbolType := node.Children[1].Value

				names := []string{}
				for _, node := range node.Children[2:] {
					if node.Name == "identifier" {
						names = append(names, node.Value)
					}
				}

				for _, name := range names {
					table.Set(name, &Symbol{SymbolType: symbolType, Kind: kind})
				}
			}
		}
	case "subroutineDec":
		if node.Children[0].Value == "method" {
			table.Set("this", &Symbol{SymbolType: "this", Kind: "argument"})
		}

		var parameterList *parser.Node

		for _, node := range node.Children {
			if node.Name == "parameterList" {
				parameterList = node
				break
			}
		}

		for i := 0; i < len(parameterList.Children); i += 3 {
			typeNode := parameterList.Children[i]
			identifier := parameterList.Children[i+1]
			name := identifier.Value

			table.Set(name, &Symbol{SymbolType: typeNode.Value, Kind: "argument"})
		}

		subroutineBody, _ := node.Find(&parser.Node{Name: "subroutineBody"})
		for _, node := range subroutineBody.Children {
			if node.Name == "varDec" {
				symbolType := node.Children[1].Value

				names := []string{}
				for _, node := range node.Children[2:] {
					if node.Name == "identifier" {
						names = append(names, node.Value)
					}
				}

				for _, name := range names {
					table.Set(name, &Symbol{
						SymbolType: symbolType,
						Kind:       "local",
					})
				}
			}
		}
	}

	return table
}
