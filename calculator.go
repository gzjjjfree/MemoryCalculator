package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/Knetic/govaluate"
)

type CalcState struct {
	win fyne.Window

	Display           binding.String   // 当前输入的算式
	Result            binding.String   // 当前算式的结果预览
	History           binding.String   // 历史记录（每次计算完成后追加）
	AllHistoryBuilder *strings.Builder // 用于保存所有历史记录的字符串，方便写入文件
	lastRecordDate    string           // 记录上一次写入时的日期（如 "2026-03-31"）

	IsNewNumber bool // 是否正在输入一个新的数字（而不是继续在当前数字后面输入）

	IsResultMode binding.Bool // true 代表显示结果（结果粗），false 代表输入中（输入粗)
	IsCalcBig    binding.Bool // 是否使用高级计算布局
	IsRadian     binding.Bool // true 为弧度模式，false 为角度模式
	Is2ndMode    binding.Bool // 是否处于 2nd 模式

	isInterceptingForScore bool            // 是否正在拦截输入
	onScoreInput           func(string)    // 拦截时的回调函数
	scoreOverlay           *fyne.Container // 平摊功能的 UI 容器
}

// 构造函数，初始化状态
func NewCalcState(w fyne.Window) *CalcState {
	s := &CalcState{
		Display:           binding.NewString(),
		Result:            binding.NewString(),
		History:           binding.NewString(),
		AllHistoryBuilder: &strings.Builder{},
		IsNewNumber:       true,
		IsResultMode:      binding.NewBool(),
		IsCalcBig:         binding.NewBool(),
		IsRadian:          binding.NewBool(),
		Is2ndMode:         binding.NewBool(),
		win:               w,
	}
	s.Display.Set("")
	s.Result.Set("0")
	s.IsResultMode.Set(true) // 初始结果行粗
	s.IsCalcBig.Set(false)
	s.IsRadian.Set(false) // 默认角度模式
	s.Is2ndMode.Set(false)
	return s
}

// 处理按键输入的核心函数
func (s *CalcState) OnTap(char string) {
	// 如果处于拦截模式，将按键传给临时函数，不执行计算逻辑
	if s.isInterceptingForScore && s.onScoreInput != nil {
		s.onScoreInput(char)
		return
	}

	// 获取当前是否处于结果模式（结果模式下输入算式会重置当前输入）
	isResultMode, _ := s.IsResultMode.Get()

	// 如果当前是结果模式，点击 % 则运行特定函数, 不执行计算逻辑
	if char == "%" && isResultMode {
		s.displayScore()
		return
	}

	current, _ := s.Display.Get()

	// 防止第一个字符就是运算符 (除了减号表示负数)
	if current == "" && s.IsNewNumber {
		if strings.ContainsAny(char, "+×÷)") {
			return
		}
	}

	// 如果是新数字开始，重置加粗状态
	if s.IsNewNumber {
		// 如果新输入的字符是运算符，且当前结果不是 0，则保留结果作为新输入的开头（例如继续在结果后面输入运算符）
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

// 处理清除键
func (s *CalcState) OnClear() {
	// 如果处于拦截模式，将按键传给临时函数，不执行计算逻辑
	if s.isInterceptingForScore && s.onScoreInput != nil {
		s.onScoreInput("C")
		return
	}

	current, _ := s.Display.Get()
	history, _ := s.History.Get()
	finalRes := s.Calculate(current)
	// 只有当当前有输入内容时才存入历史，避免存入多余空行
	if s.IsNewNumber == false && current != "0" && current != "" {
		current = checkLastOperator(current)

		newHistory, _ := updateFontSizeBasedOnWidth(current+" = "+finalRes, nil) // 更新字体大小和换行状态
		s.History.Set(history + "\n" + newHistory)

		s.recordToHistory(current, finalRes) // 追加到历史记录中
	}

	s.Display.Set("")
	s.Result.Set("0")

	s.IsNewNumber = true
	isChangeRow = false
	s.IsResultMode.Set(true)
}

// 处理等号键
func (s *CalcState) OnEqual() {
	// 如果处于拦截模式，将按键传给临时函数，不执行计算逻辑
	if s.isInterceptingForScore && s.onScoreInput != nil {
		s.onScoreInput("=")
		return
	}

	history, _ := s.History.Get()
	current, _ := s.Display.Get()

	// 自动补全未闭合的括号 (防止 govaluate 报错)
	leftCount := strings.Count(current, "(")
	rightCount := strings.Count(current, ")")
	if leftCount > rightCount {
		current += strings.Repeat(")", leftCount-rightCount)
	}

	finalRes := s.Calculate(current)

	// 只有当当前有输入内容时才存入历史，避免存入多余空行
	if finalRes != "" && finalRes != "Error" {
		current = checkLastOperator(current)

		newHistory, _ := updateFontSizeBasedOnWidth(current+" = "+finalRes, nil) // 更新字体大小和换行状态
		s.History.Set(history + "\n" + newHistory)

		s.Result.Set("= " + finalRes)
		s.IsNewNumber = true
		isChangeRow = false
		s.IsResultMode.Set(true)

		s.recordToHistory(current, finalRes) // 追加到历史记录中
	}
}

// 计算函数，使用 govaluate 解析和计算表达式
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
	exprStr = strings.ReplaceAll(exprStr, "%", "*0.01")   // 修复百分号
	exprStr = strings.ReplaceAll(exprStr, "1/x(", "inv(") // 修复倒数函数
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
		"asin": func(args ...any) (any, error) {
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
		"cos": func(args ...any) (any, error) {
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
		"acos": func(args ...any) (any, error) {
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
		"tan": func(args ...any) (any, error) {
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
		"atan": func(args ...any) (any, error) {
			val := args[0].(float64)
			res := math.Atan(val)
			if !isRad {
				res = res * 180 / math.Pi
			}
			return res, nil
		},
		"sqrt": func(args ...any) (any, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			return math.Sqrt(val), nil
		},
		"lg": func(args ...any) (any, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			return math.Log10(val), nil
		},
		"ln": func(args ...any) (any, error) {
			if len(args) < 1 {
				return 0.0, nil
			}
			val, ok := args[0].(float64)
			if !ok {
				return 0.0, nil
			}
			return math.Log(val), nil
		},
		"pow": func(args ...any) (any, error) {
			if len(args) < 2 {
				return nil, errors.New("pow requires 2 arguments")
			}
			return math.Pow(args[0].(float64), args[1].(float64)), nil
		},
		"pow10": func(args ...any) (any, error) {
			return math.Pow(10, args[0].(float64)), nil
		},
		"exp": func(args ...any) (any, error) {
			return math.Exp(args[0].(float64)), nil
		},
		"sqr": func(args ...any) (any, error) {
			val := args[0].(float64)
			return val * val, nil
		},
		"fact": func(args ...any) (any, error) {
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

	res, err := expression.Evaluate(map[string]any{})	
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

// 处理退格键
func (s *CalcState) OnBackspace() {
	// 如果处于拦截模式，将按键传给临时函数，不执行计算逻辑
	if s.isInterceptingForScore && s.onScoreInput != nil {
		s.onScoreInput("⌫")
		return
	}

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

	// 使用 rune 处理多字节字符
	runes := []rune(current)

	if len(runes) <= 1 { // 如果只有一个字符，退格后变成空
		s.Display.Set("")
		s.Result.Set("= 0")
		isChangeRow = false // 重置换行状态
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

		// 清理掉最后一个换行符
		lastNewlineIdx := strings.LastIndexAny(newEq, "\n")
		if lastNewlineIdx != -1 {
			newEq = newEq[:lastNewlineIdx] + newEq[lastNewlineIdx+1:]
		}

		stateMutex.Lock()
		isChangeRow = false
		stateMutex.Unlock()
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

// 切换大布局的动作
func (s *CalcState) OnGoBigGrid() {
	s.IsNewNumber = true
	if ok, _ := s.IsCalcBig.Get(); ok {
		s.IsCalcBig.Set(false)
	} else {
		s.IsCalcBig.Set(true)
	}
}

// 处理高级函数按钮的输入
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
		return
	}

	// 实时预览
	res := s.Calculate(newEq)
	if res != "" {
		s.Result.Set("= " + res)
	}
}

// 切换 DEG/RAD 的动作
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

// 显示平摊分数的界面
func (s *CalcState) displayScore() {
	result, _ := s.Result.Get()
	scoreStr := strings.TrimPrefix(result, "= ")
	scoreFloat, _ := strconv.ParseFloat(scoreStr, 64)
	totalScore := int(scoreFloat)

	if totalScore <= 0 || totalScore >= 600 {
		return
	}

	peopleInput := binding.NewString()
	peopleInput.Set("")

	// 定义核心计算逻辑
	updatePreview := func() {
		val, _ := peopleInput.Get()
		if val == "" {
			s.Result.Set(fmt.Sprintf("总分:%d | 请输入人数", totalScore))
			s.IsResultMode.Set(false)
			return
		}

		num, err := strconv.Atoi(val)
		if err != nil || num <= 0 {
			s.Result.Set("人数无效")
			s.IsResultMode.Set(false)
			return
		}

		// 计算平均分和余数
		base := totalScore / num
		rem := totalScore % num
		var finalStr string
		if rem == 0 {
			finalStr = fmt.Sprintf("总分:%d | %d人%d分  ", totalScore, num, base)
		} else {
			finalStr = fmt.Sprintf("总分:%d | %d人%d分, %d人%d分  ",
				totalScore, rem, base + 1, num - rem, base)
		}
		s.Result.Set(finalStr)
		s.IsResultMode.Set(false)
	}

	// 监听输入变化实现实时预览
	peopleInput.AddListener(binding.NewDataListener(func() {
		updatePreview()
	}))

	displayLabel := widget.NewLabelWithData(peopleInput)
	displayLabel.Alignment = fyne.TextAlignCenter
	displayLabel.TextStyle = fyne.TextStyle{Bold: true}

	title := widget.NewLabel("请输入平摊人数")
	title.Alignment = fyne.TextAlignCenter

	// 3. 按钮逻辑改造
	btnCancel := widget.NewButton("返回", func() {
		s.isInterceptingForScore = false
		s.scoreOverlay.Hide()
		// 返回时恢复原始结果显示
		s.Result.Set(result)
	})

	// 变更为重置按钮
	btnReset := widget.NewButton("重置", func() {
		peopleInput.Set("") // 清空输入，触发监听器更新预览
	})

	cardContent := container.NewVBox(
		title,
		container.NewPadded(displayLabel),
		container.NewHBox(layout.NewSpacer(), layout.NewSpacer(), btnCancel, layout.NewSpacer(), btnReset, layout.NewSpacer(), layout.NewSpacer()),
	)

	cardBackground := canvas.NewRectangle(theme.BackgroundColor())
	cardBackground.SetMinSize(fyne.NewSize(360, 150))
	card := container.NewStack(cardBackground, cardContent)

	s.scoreOverlay.Objects = []fyne.CanvasObject{
		container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(card),
			layout.NewSpacer(),
			layout.NewSpacer(),
			layout.NewSpacer(),
			layout.NewSpacer(),
			layout.NewSpacer(),
			layout.NewSpacer(),
			layout.NewSpacer(),
		),
	}
	s.scoreOverlay.Refresh()
	s.scoreOverlay.Show()

	// 接管输入
	s.isInterceptingForScore = true
	s.onScoreInput = func(char string) {
		current, _ := peopleInput.Get()
		if char >= "0" && char <= "9" {
			// 限制人数长度防止溢出
			if len(current) < 3 {
				peopleInput.Set(current + char)
			}
		} else if char == "C" {
			s.isInterceptingForScore = false
			s.scoreOverlay.Hide()
			s.Result.Set(result)
			s.IsResultMode.Set(false)
		} else if char == "⌫" {
			if current != "" {
				runes := []rune(current)
				// 截取掉最后一个字符
				newVal := string(runes[:len(runes)-1])
				peopleInput.Set(newVal)
			}
		}
	}
}
