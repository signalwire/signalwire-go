package builtin

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"

	"github.com/signalwire/signalwire-go/pkg/skills"
	"github.com/signalwire/signalwire-go/pkg/swaig"
)

// MathSkill provides basic mathematical calculation capabilities.
type MathSkill struct {
	skills.BaseSkill
}

// NewMath creates a new MathSkill.
func NewMath(params map[string]any) skills.SkillBase {
	return &MathSkill{
		BaseSkill: skills.BaseSkill{
			SkillName: "math",
			SkillDesc: "Perform basic mathematical calculations",
			SkillVer:  "1.0.0",
			Params:    params,
		},
	}
}

func (s *MathSkill) Setup() bool { return true }

func (s *MathSkill) RegisterTools() []skills.ToolRegistration {
	return []skills.ToolRegistration{
		{
			Name:        "calculate",
			Description: "Perform a mathematical calculation with basic operations (+, -, *, /, %)",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{
						"type":        "string",
						"description": "Mathematical expression to evaluate (e.g., '2 + 3 * 4', '(10 + 5) / 3')",
					},
				},
				"required": []string{"expression"},
			},
			Handler: s.handleCalculate,
		},
	}
}

func (s *MathSkill) handleCalculate(args map[string]any, _ map[string]any) *swaig.FunctionResult {
	expr, _ := args["expression"].(string)
	if expr == "" {
		return swaig.NewFunctionResult("Please provide a mathematical expression to calculate.")
	}

	result, err := safeEval(expr)
	if err != nil {
		return swaig.NewFunctionResult(
			"Error: Invalid expression. Only numbers and basic math operators (+, -, *, /, %, parentheses) are allowed.",
		)
	}

	// Format result nicely
	if result == math.Trunc(result) {
		return swaig.NewFunctionResult(fmt.Sprintf("%s = %d", expr, int64(result)))
	}
	return swaig.NewFunctionResult(fmt.Sprintf("%s = %g", expr, result))
}

// safeEval parses and evaluates a mathematical expression using Go's AST parser.
func safeEval(expression string) (float64, error) {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return 0, fmt.Errorf("parse error: %w", err)
	}
	return evalNode(expr)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		if n.Kind == token.INT || n.Kind == token.FLOAT {
			return strconv.ParseFloat(n.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal: %s", n.Value)

	case *ast.ParenExpr:
		return evalNode(n.X)

	case *ast.UnaryExpr:
		val, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -val, nil
		case token.ADD:
			return val, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %s", n.Op)
		}

	case *ast.BinaryExpr:
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return math.Mod(left, right), nil
		default:
			return 0, fmt.Errorf("unsupported binary operator: %s", n.Op)
		}

	default:
		return 0, fmt.Errorf("unsupported expression type: %T", node)
	}
}

func (s *MathSkill) GetPromptSections() []map[string]any {
	return []map[string]any{
		{
			"title": "Mathematical Calculations",
			"body":  "You can perform mathematical calculations for users.",
			"bullets": []string{
				"Use the calculate tool for any math expressions",
				"Supports basic operations: +, -, *, /, %",
				"Can handle parentheses for complex expressions",
			},
		},
	}
}

func init() {
	skills.RegisterSkill("math", NewMath)
}
