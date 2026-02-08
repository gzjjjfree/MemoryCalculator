package main

import (
	"math"
	"strconv"
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestCalculate(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	// 此时调用 NewCalcState 就不会报错了
	state = NewCalcState()

	tests := []struct {
		name     string
		input    string
		isRadian bool   // 是否切换为弧度模式
		expected string // 预期结果字符串
	}{
		// --- 基础四则运算 ---
		{"Addition", "1+1", false, "2"},
		{"Subraction", "10-5.5", false, "4.5"},
		{"Multiplication", "2×3", false, "6"},
		{"Division", "4÷2", false, "2"},

		// --- 百分号逻辑 (根据当前实现: x% = x*0.01) ---
		{"Percentage Simple", "100%", false, "1"},
		{"Percentage Addition", "50%+50%", false, "1"},
		// 逻辑说明：5÷2% 解析为 5/2*0.01 = 2.5*0.01 = 0.025 (符合左结合律优先级)
		{"Percentage Priority", "5÷2%", false, "0.025"}, 

		// --- 幂运算与根号 ---
		{"Power", "2^3", false, "8"},
		{"Square", "sqr(4)", false, "16"},
		{"Sqrt", "sqrt(9)", false, "3"},

		// --- 三角函数: 角度模式 (DEG) ---
		{"Sin DEG", "sin(30)", false, "0.5"},
		{"Cos DEG", "cos(60)", false, "0.5"},
		{"Tan DEG", "tan(45)", false, "1"},

		// --- 三角函数: 弧度模式 (RAD) ---
		// 逻辑说明：sin(π/6) 在弧度模式下等于 0.5
		{"Sin RAD", "sin(π÷6)", true, "0.5"}, 
		{"Asin RAD", "asin(1)", true, "1.570796"}, // π/2

		// --- 常数测试 ---
		{"Constant Pi", "π", false, "3.141593"},
		{"Constant E", "e", false, "2.718282"},

		// --- 异常处理 ---
		{"Division by Zero", "1÷0", false, "Error"},
		{"Invalid Sqrt", "sqrt(-1)", false, "Error"},
		{"Domain Error", "asin(2)", false, "Error"},
		{"Empty Input", "", false, "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置模式
			state.IsRadian.Set(tt.isRadian)
			
			// 执行计算
			got := state.Calculate(tt.input)

			// 验证结果
			if !compareResults(got, tt.expected) {
				t.Errorf("Input: %s (Rad:%v), Expected: %s, Got: %s", 
					tt.input, tt.isRadian, tt.expected, got)
			}
		})
	}
}

// compareResults 辅助函数：处理浮点数精度和 Error 字符串匹配
func compareResults(got, expected string) bool {
	if got == expected {
		return true
	}

	// 如果涉及 Error，必须完全匹配
	if expected == "Error" || got == "Error" {
		return got == expected
	}

	// 尝试将字符串转为浮点数进行近似比较
	fGot, err1 := strconv.ParseFloat(got, 64)
	fExp, err2 := strconv.ParseFloat(expected, 64)
	if err1 == nil && err2 == nil {
		// 定义容差，例如 1e-6
		return math.Abs(fGot-fExp) < 1e-6
	}

	return false
}
