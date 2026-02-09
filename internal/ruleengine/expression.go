/**
 * 规则表达式求值器
 *
 * @author Anner
 * Created on 2026/2/6
 */

package ruleengine

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
)

type expressionScope struct {
	baseNumbers     map[string]float64
	baseBools       map[string]bool
	overrideNumbers map[string]float64
	overrideBools   map[string]bool
}

func (s expressionScope) withNumberOverrides(extra map[string]float64) expressionScope {
	return expressionScope{
		baseNumbers:     s.baseNumbers,
		baseBools:       s.baseBools,
		overrideNumbers: extra,
		overrideBools:   s.overrideBools,
	}
}

func evalRuleExpression(expression string, scope expressionScope) (bool, error) {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return false, fmt.Errorf("parse expression failed: %w", err)
	}
	return evalBoolExpr(expr, scope)
}

func evalBoolExpr(expr ast.Expr, scope expressionScope) (bool, error) {
	switch node := expr.(type) {
	case *ast.BinaryExpr:
		switch node.Op {
		case token.LAND:
			left, err := evalBoolExpr(node.X, scope)
			if err != nil {
				return false, err
			}
			if !left {
				return false, nil
			}
			return evalBoolExpr(node.Y, scope)
		case token.LOR:
			left, err := evalBoolExpr(node.X, scope)
			if err != nil {
				return false, err
			}
			if left {
				return true, nil
			}
			return evalBoolExpr(node.Y, scope)
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			left, err := evalNumericExpr(node.X, scope)
			if err != nil {
				return false, err
			}
			right, err := evalNumericExpr(node.Y, scope)
			if err != nil {
				return false, err
			}
			return compareNumber(left, right, node.Op), nil
		default:
			value, err := evalNumericExpr(expr, scope)
			if err != nil {
				return false, err
			}
			return math.Abs(value) > 1e-9, nil
		}
	case *ast.UnaryExpr:
		if node.Op == token.NOT {
			value, err := evalBoolExpr(node.X, scope)
			if err != nil {
				return false, err
			}
			return !value, nil
		}
		value, err := evalNumericExpr(expr, scope)
		if err != nil {
			return false, err
		}
		return math.Abs(value) > 1e-9, nil
	case *ast.ParenExpr:
		return evalBoolExpr(node.X, scope)
	case *ast.Ident:
		value, ok := scope.lookupBool(node.Name)
		if ok {
			return value, nil
		}
		number, ok := scope.lookupNumber(node.Name)
		if !ok {
			return false, fmt.Errorf("unknown identifier: %s", node.Name)
		}
		return math.Abs(number) > 1e-9, nil
	default:
		value, err := evalNumericExpr(expr, scope)
		if err != nil {
			return false, err
		}
		return math.Abs(value) > 1e-9, nil
	}
}

func evalNumericExpr(expr ast.Expr, scope expressionScope) (float64, error) {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind != token.INT && node.Kind != token.FLOAT {
			return 0, fmt.Errorf("unsupported literal kind: %s", node.Kind.String())
		}
		return strconv.ParseFloat(node.Value, 64)
	case *ast.Ident:
		if value, ok := scope.lookupNumber(node.Name); ok {
			return value, nil
		}
		if value, ok := scope.lookupBool(node.Name); ok {
			if value {
				return 1, nil
			}
			return 0, nil
		}
		return 0, fmt.Errorf("unknown identifier: %s", node.Name)
	case *ast.BinaryExpr:
		return evalNumericBinary(node, scope)
	case *ast.ParenExpr:
		return evalNumericExpr(node.X, scope)
	case *ast.UnaryExpr:
		value, err := evalNumericExpr(node.X, scope)
		if err != nil {
			return 0, err
		}
		if node.Op == token.SUB {
			return -value, nil
		}
		if node.Op == token.ADD {
			return value, nil
		}
		return 0, fmt.Errorf("unsupported unary operator: %s", node.Op.String())
	case *ast.CallExpr:
		name, ok := node.Fun.(*ast.Ident)
		if !ok {
			return 0, fmt.Errorf("unsupported call expression")
		}
		return evalBuiltin(name.Name, node.Args, scope)
	default:
		return 0, fmt.Errorf("unsupported expression node: %T", node)
	}
}

func evalNumericBinary(node *ast.BinaryExpr, scope expressionScope) (float64, error) {
	left, err := evalNumericExpr(node.X, scope)
	if err != nil {
		return 0, err
	}
	right, err := evalNumericExpr(node.Y, scope)
	if err != nil {
		return 0, err
	}

	switch node.Op {
	case token.ADD:
		return left + right, nil
	case token.SUB:
		return left - right, nil
	case token.MUL:
		return left * right, nil
	case token.QUO:
		if right == 0 {
			return 0, nil
		}
		return left / right, nil
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		if compareNumber(left, right, node.Op) {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported binary operator: %s", node.Op.String())
	}
}

func evalBuiltin(name string, args []ast.Expr, scope expressionScope) (float64, error) {
	switch name {
	case "percent_diff", "同比增速":
		if len(args) != 2 {
			return 0, fmt.Errorf("percent_diff args mismatch")
		}
		current, err := evalNumericExpr(args[0], scope)
		if err != nil {
			return 0, err
		}
		lastYear, err := evalNumericExpr(args[1], scope)
		if err != nil {
			return 0, err
		}
		if lastYear == 0 {
			return 0, nil
		}
		return (current - lastYear) / lastYear * 100, nil
	case "abs", "绝对值":
		if len(args) != 1 {
			return 0, fmt.Errorf("abs args mismatch")
		}
		value, err := evalNumericExpr(args[0], scope)
		if err != nil {
			return 0, err
		}
		return math.Abs(value), nil
	case "min", "最小值":
		if len(args) != 2 {
			return 0, fmt.Errorf("min args mismatch")
		}
		left, err := evalNumericExpr(args[0], scope)
		if err != nil {
			return 0, err
		}
		right, err := evalNumericExpr(args[1], scope)
		if err != nil {
			return 0, err
		}
		if left < right {
			return left, nil
		}
		return right, nil
	case "max", "最大值":
		if len(args) != 2 {
			return 0, fmt.Errorf("max args mismatch")
		}
		left, err := evalNumericExpr(args[0], scope)
		if err != nil {
			return 0, err
		}
		right, err := evalNumericExpr(args[1], scope)
		if err != nil {
			return 0, err
		}
		if left > right {
			return left, nil
		}
		return right, nil
	case "round", "四舍五入":
		if len(args) != 1 {
			return 0, fmt.Errorf("round args mismatch")
		}
		value, err := evalNumericExpr(args[0], scope)
		if err != nil {
			return 0, err
		}
		return math.Round(value), nil
	default:
		return 0, fmt.Errorf("unsupported function: %s", name)
	}
}

func compareNumber(left, right float64, op token.Token) bool {
	diff := left - right
	equal := math.Abs(diff) <= 1e-9
	switch op {
	case token.EQL:
		return equal
	case token.NEQ:
		return !equal
	case token.LSS:
		return diff < 0 && !equal
	case token.LEQ:
		return diff < 0 || equal
	case token.GTR:
		return diff > 0 && !equal
	case token.GEQ:
		return diff > 0 || equal
	default:
		return false
	}
}

func (s expressionScope) lookupNumber(name string) (float64, bool) {
	if name == "true" {
		return 1, true
	}
	if name == "false" {
		return 0, true
	}
	if s.overrideNumbers != nil {
		if value, ok := s.overrideNumbers[name]; ok {
			return value, true
		}
	}
	if s.baseNumbers != nil {
		if value, ok := s.baseNumbers[name]; ok {
			return value, true
		}
	}
	if s.overrideBools != nil {
		if value, ok := s.overrideBools[name]; ok {
			if value {
				return 1, true
			}
			return 0, true
		}
	}
	if s.baseBools != nil {
		if value, ok := s.baseBools[name]; ok {
			if value {
				return 1, true
			}
			return 0, true
		}
	}
	return 0, false
}

func (s expressionScope) lookupBool(name string) (bool, bool) {
	if name == "true" {
		return true, true
	}
	if name == "false" {
		return false, true
	}
	if s.overrideBools != nil {
		if value, ok := s.overrideBools[name]; ok {
			return value, true
		}
	}
	if s.baseBools != nil {
		if value, ok := s.baseBools[name]; ok {
			return value, true
		}
	}
	if s.overrideNumbers != nil {
		if value, ok := s.overrideNumbers[name]; ok {
			return math.Abs(value) > 1e-9, true
		}
	}
	if s.baseNumbers != nil {
		if value, ok := s.baseNumbers[name]; ok {
			return math.Abs(value) > 1e-9, true
		}
	}
	return false, false
}
