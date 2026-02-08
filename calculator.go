package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"fyne.io/fyne/v2/data/binding"
	"github.com/Knetic/govaluate"
)

type CalcState struct {
	Display     binding.String
	Result      binding.String
	History     binding.String
	IsNewNumber bool
	// 用于控制 UI 字体是否加粗的绑定
	IsResultMode binding.Bool // true 代表显示结果（结果粗），false 代表输入中（输入粗

	IsCalcBig binding.Bool
	IsRadian  binding.Bool // true 为弧度模式，false 为角度模式
	Is2ndMode binding.Bool
}

func NewCalcState() *CalcState {
	s := &CalcState{
		Display:      binding.NewString(),
		Result:       binding.NewString(),
		History:      binding.NewString(),
		IsNewNumber:  true,
		IsResultMode: binding.NewBool(),
		IsCalcBig:    binding.NewBool(),
		IsRadian:     binding.NewBool(),
		Is2ndMode:    binding.NewBool(),
	}
	s.Display.Set("")
	s.Result.Set("0")
	s.IsResultMode.Set(true) // 初始结果行粗
	s.IsCalcBig.Set(false)
	s.IsRadian.Set(false) // 默认角度模式
	s.Is2ndMode.Set(false)
	return s
}

func (s *CalcState) OnTap(char string) {
	current, _ := s.Display.Get()

	// 防止第一个字符就是运算符 (除了减号表示负数)
	if current == "" && s.IsNewNumber {
		if strings.ContainsAny(char, "+×÷)") {
			return
		}
	}

	// 如果是新数字开始，重置加粗状态
	if s.IsNewNumber {
		result, _ := s.Result.Get()
		current = ""
		if result != "0" && strings.ContainsAny(char, "+-×÷)") {
			current = result[2:]
		}
		s.IsResultMode.Set(false)
		s.Display.Set(current + char)
		s.IsNewNumber = false
		return
	} else {
		// 处理重复点击运算符：如果最后一个字符是运算符，再次点击则替换它
		operators := "+-×÷"
		if len(current) > 0 && strings.ContainsAny(char, operators) {
			lastChar := current[len(current)-1:]
			if strings.ContainsAny(lastChar, operators) {
				s.Display.Set(current[:len(current)-1] + char)
				return
			}
		}
		s.Display.Set(current + char)
	}

	// 实时更新结果
	newEq, _ := s.Display.Get()
	res := s.Calculate(newEq)
	if res != "" {
		s.Result.Set("= " + res)
	}
}

func (s *CalcState) OnClear() {
	current, _ := s.Display.Get()
	history, _ := s.History.Get()
	finalRes := s.Calculate(current)
	// 只有当当前有输入内容时才存入历史，避免存入多余空行
	if s.IsNewNumber == false && current != "0" && current != "" {
		current = checkLastOperator(current)
		s.History.Set(history + "\n" + current + " = " + finalRes)
		s.appendToLocalFile(current) // 自动保存
	}

	s.Display.Set("")
	s.Result.Set("0")

	s.IsNewNumber = true
	isChangeRow = false
	s.IsResultMode.Set(true)
}

func (s *CalcState) OnEqual() {
	history, _ := s.History.Get()
	current, _ := s.Display.Get()

	leftCount := strings.Count(current, "(")
	rightCount := strings.Count(current, ")")
	if leftCount > rightCount {
		current += strings.Repeat(")", leftCount-rightCount)
	}

	finalRes := s.Calculate(current)

	if finalRes != "" && finalRes != "Error" {
		current = checkLastOperator(current)
		newHistory := history + "\n" + current + " = " + finalRes
		s.History.Set(newHistory)
		s.Result.Set("= " + finalRes)
		s.IsNewNumber = true
		isChangeRow = false
		s.IsResultMode.Set(true)

		s.appendToLocalFile(current + " = " + finalRes) // 自动保存
	}
}

func (s *CalcState) Calculate(equation string) string {
	if equation == "" || equation == "0" {
		return "0"
	}

	equation = checkLastOperator(equation)

	// 安全符号替换与自动补全
	exprStr := equation
	// 替换 × ÷
	exprStr = strings.ReplaceAll(exprStr, "×", "*")
	exprStr = strings.ReplaceAll(exprStr, "÷", "/")
	exprStr = strings.ReplaceAll(exprStr, "%", "*0.01")     // 修复百分号
	exprStr = strings.ReplaceAll(exprStr, "1/x(", "inv(")  // 修复倒数函数
	// 替换 π（仅独立常量，不在函数名中）
	exprStr = replaceConstant(exprStr, "π", fmt.Sprintf("%f", math.Pi))
	// 替换 e（仅独立常量，不在函数名、exp等中）
	exprStr = replaceConstant(exprStr, "e", fmt.Sprintf("%f", math.E))
	// 替换 ^ 为 pow 函数（如 2^3 -> pow(2,3)）
	exprStr = replacePower(exprStr)

	// 自动补全未闭合的括号 (防止 govaluate 报错)
	leftCount := strings.Count(exprStr, "(")
	rightCount := strings.Count(exprStr, ")")
	if leftCount > rightCount {
		exprStr += strings.Repeat(")", leftCount-rightCount)
	}

	isRad, _ := s.IsRadian.Get()
	// 定义高级函数映射 (增加安全检查)
	functions := map[string]govaluate.ExpressionFunction{
		"sin": func(args ...any) (any, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			if !isRad { // 如果不是弧度模式，进行转换
				val = val * math.Pi / 180
			}
			return math.Sin(val), nil
		},
		"asin": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return nil, errors.New("Domain Error")
			}
			val, ok := args[0].(float64)
			if !ok || val < -1 || val > 1 {
				return nil, errors.New("Domain Error")
			}
			res := math.Asin(val)
			if !isRad {
				res = res * 180 / math.Pi
			}
			return res, nil
		},
		"cos": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			if !isRad { // 如果不是弧度模式，进行转换
				val = val * math.Pi / 180
			}
			return math.Cos(val), nil
			//return math.Cos(val * math.Pi / 180), nil
		},
		"acos": func(args ...interface{}) (interface{}, error) {
			val := args[0].(float64)
			if val < -1 || val > 1 {
				return nil, errors.New("Domain Error")
			}
			res := math.Acos(val)
			if !isRad {
				res = res * 180 / math.Pi
			}
			return res, nil
		},
		"tan": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			if !isRad { // 如果不是弧度模式，进行转换
				val = val * math.Pi / 180
			}
			return math.Tan(val), nil
		},
		"atan": func(args ...interface{}) (interface{}, error) {
			val := args[0].(float64)
			res := math.Atan(val)
			if !isRad {
				res = res * 180 / math.Pi
			}
			return res, nil
		},
		"sqrt": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			return math.Sqrt(val), nil
		},
		"lg": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			return math.Log10(val), nil
		},
		"ln": func(args ...interface{}) (interface{}, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			return math.Log(val), nil
		},
		"pow": func(args ...interface{}) (interface{}, error) {
			if len(args) < 2 {
				return nil, errors.New("pow requires 2 arguments")
			}
			return math.Pow(args[0].(float64), args[1].(float64)), nil
		},
		"pow10": func(args ...interface{}) (interface{}, error) {
			return math.Pow(10, args[0].(float64)), nil
		},
		"exp": func(args ...interface{}) (interface{}, error) {
			return math.Exp(args[0].(float64)), nil
		},
		"sqr": func(args ...interface{}) (interface{}, error) {
			val := args[0].(float64)
			return val * val, nil
		},
		"fact": func(args ...interface{}) (interface{}, error) {
			n := args[0].(float64)
			if n < 0 {
				return 0.0, nil
			}
			res := 1.0
			for i := 2.0; i <= n; i++ {
				res *= i
			}
			return res, nil
		},
		"inv": func(args ...any) (any, error) {
			val := args[0].(float64)
			if val == 0 {
				return nil, errors.New("Division by zero")
			}
			return 1.0 / val, nil
		},
	}

	// 执行解析计算
	expression, err := govaluate.NewEvaluableExpressionWithFunctions(exprStr, functions)
	if err != nil {
		return ""
	}

	res, err := expression.Evaluate(map[string]interface{}{})
	//res, err := expression.Evaluate(nil)
	if err != nil {
		return "Error"
	}
	// 检查结果是否有效
	f, ok := res.(float64)
	if !ok {
		return "Error"
	}
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return "Error" // 这样 1/0 就会返回 Error 了
	}
	// 格式化输出，如果是整数则不带小数点
	return fmt.Sprintf("%g", res)
}

// 安全替换常量（仅替换独立的 π、e，不在函数名、变量名中）
func replaceConstant(expr, symbol, value string) string {
	if symbol == "π" {
		return strings.ReplaceAll(expr, "π", value)
	}
	// \b 匹配单词边界
	pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol))
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllString(expr, value)
}

// 替换幂运算符 ^ 为 pow(x,y)
func replacePower(expr string) string {
	// 用正则匹配形如 a^b 的表达式，替换为 pow(a,b)
	// 只处理简单数字和括号表达式
	pattern := `([0-9.]+|\([^)]+\))\^([0-9.]+|\([^)]+\))`
	re := regexp.MustCompile(pattern)
	return re.ReplaceAllStringFunc(expr, func(m string) string {
		parts := strings.Split(m, "^")
		if len(parts) == 2 {
			return fmt.Sprintf("pow(%s,%s)", parts[0], parts[1])
		}
		return m
	})
}

func (s *CalcState) OnBackspace() {
	// 如果当前处于结果显示模式，退格键通常应该直接清空结果回到输入模式
	isResult, _ := s.IsResultMode.Get()
	if isResult {
		s.IsResultMode.Set(false)
		s.IsNewNumber = false
		return
	}

	current, _ := s.Display.Get()
	if current == "0" || current == "" {
		return
	}

	// --- 关键修复：使用 rune 处理多字节字符 ---
	runes := []rune(current)
	if len(runes) <= 1 {
		s.Display.Set("")
		s.Result.Set("= 0")
	} else {
		// 删掉最后一个字符
		newEq := string(runes[:len(runes)-1])
		// 自动清理掉残余的函数名，如输入了 sin( 删掉 ( 后，把 sin 也删掉，保持算式整洁
		for _, fn := range []string{"sin", "cos", "tan", "lg", "ln", "sqrt", "fact"} {
			if strings.HasSuffix(newEq, fn) {
				newEq = strings.TrimSuffix(newEq, fn)
				break
			}
		}

		lastNewlineIdx := strings.LastIndexAny(newEq, "\n")
		if lastNewlineIdx != -1 {
			newEq = newEq[:lastNewlineIdx] + newEq[lastNewlineIdx+1:]
		}
		isChangeRow = false
		s.Display.Set(newEq)

		// 更新实时预览
		res := s.Calculate(newEq)
		if res != "" {
			s.Result.Set("= " + res)
		} else {
			s.Result.Set("0")
		}
	}
}

func (s *CalcState) OnGoBigGrid() {
	s.IsNewNumber = true
	if ok, _ := s.IsCalcBig.Get(); ok {
		s.IsCalcBig.Set(false)
	} else {
		s.IsCalcBig.Set(true)
	}
}

func (s *CalcState) OnAdvancedTap(op string) {
	s.IsNewNumber = false
	s.IsResultMode.Set(false)
	is2nd, _ := s.Is2ndMode.Get()
	current, _ := s.Display.Get()

	// 注意：这里的 key 必须和你按钮初始化的 text 一致
	opMapping := map[string]string{
		"sin": "sin(", "cos": "cos(", "tan": "tan(",
		"lg": "lg(", "ln": "ln(", "√x": "sqrt(",
	}

	// 如果是 2nd 模式，映射到对应的反函数或二次幂
	secondMapping := map[string]string{
		"sin": "asin(", "cos": "acos(", "tan": "atan(",
		"lg": "pow10(", "ln": "exp(", "√x": "sqr(",
	}

	var toAdd string
	if is2nd {
		if val, ok := secondMapping[op]; ok {
			toAdd = val
		} else {
			toAdd = op // 如果没有映射，按原样处理
		}
	} else {
		if val, ok := opMapping[op]; ok {
			toAdd = val
		} else {
			toAdd = op
		}
	}

	s.Display.Set(current + toAdd)

	newEq, _ := s.Display.Get()
	// 如果是以 "(" 结尾（刚输入完函数名），通常不需要显示即时预览结果
	if strings.HasSuffix(newEq, "(") {
		//s.Result.Set("0")
		return
	}

	// 实时预览
	res := s.Calculate(newEq)
	if res != "" {
		s.Result.Set("= " + res)
	}
}

func (s *CalcState) OnDegToRad() {
	isRad, _ := s.IsRadian.Get()

	s.IsRadian.Set(!isRad)

	// 切换后重新触发一次计算，更新预览结果
	current, _ := s.Display.Get()
	if current == "" {
		s.Result.Set("0")
	}
	s.Result.Set("= " + s.Calculate(current))
}

// 切换 2nd 状态的动作
func (s *CalcState) OnToggle2nd() {
	val, _ := s.Is2ndMode.Get()
	s.Is2ndMode.Set(!val)
}

// 最后一个字符为运算符时，删除它
func checkLastOperator(equation string) string {
	if len(equation) == 0 {
		return equation
	}
	operators := "+-×÷"
	lastChar := equation[len(equation)-1:]
	if strings.ContainsAny(lastChar, operators) {
		equation = equation[:len(equation)-1]
	}
	return equation
}
