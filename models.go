package main

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

var isChangeRow bool = false       // 是否需要换行
var stateMutex sync.Mutex          // 保护 isChangeRow 的并发访问
var richInputFontSize float32 = 42 // 初始字体大小，后续会根据输入动态调整
var labelFontSize float32 = 18     // 历史记录标签的字体大小，保持固定

// 全局变量：当前输入框宽度和字体大小
var actualW float32 = 300 // 可在UI初始化时动态赋值

// 字体大小自适应，step=2
func updateFontSizeBasedOnWidth(text string, richInput *widget.RichText) (string, bool) {
	var changeText string = text
	if actualW <= 0 {
		return changeText, false
	}
	if isChangeRow {
		lastNewlineIdx := strings.LastIndex(changeText, "\n")
		searchStart := 0
		if lastNewlineIdx != -1 {
			searchStart = lastNewlineIdx + 1
		}
		lastLine := changeText[searchStart:]
		if measureWidth(lastLine, 18) > actualW {
			operators := "+-×÷*/="

			// 初始搜索位置：当前行的末尾
			searchPos := len(lastLine)
			foundIdx := -1

			for {
				// 在当前搜索范围 [0:searchPos] 内找最后一个运算符
				idx := strings.LastIndexAny(lastLine[:searchPos], operators)

				if idx == -1 {
					break // 没找到任何运算符，跳出循环
				}

				// 检查如果在此处换行，该行（从开头到该符号前）是否能放下
				// 注意：这里检查的是从这一行起始到该符号位置的宽度
				if measureWidth(lastLine[:idx], 18) <= actualW {
					foundIdx = idx
					break // 找到了最靠右且不超宽的符号，退出循环
				}

				// 如果该符号处依然超宽，继续向左搜索
				searchPos = idx
			}

			if foundIdx != -1 {
				// 找到合适的换行点
				absoluteOpIdx := searchStart + foundIdx
				changeText = changeText[:absoluteOpIdx] + "\n" + changeText[absoluteOpIdx:]
				return changeText, true
			} else {
				// 【保底逻辑】如果整行都没有符合条件的符号（例如全是长数字）
				// 为了防止显示溢出，只能在当前行最后一个字符强行换行
				absoluteOpIdx := searchStart + len(lastLine) - 1
				changeText = changeText[:absoluteOpIdx] + "\n" + changeText[absoluteOpIdx:]
				return changeText, true
			}
		}
		return text, false
	}

	fontSizes := []float32{42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18}
	for _, size := range fontSizes {
		if measureWidth(text, size) <= actualW {
			richInputFontSize = size
			return text, false
		}
	}
	richInputFontSize = 18 // 降到最低
	stateMutex.Lock()
	isChangeRow = true
	stateMutex.Unlock()
	// 此时递归进入 isChangeRow 分支执行符号换行
	return updateFontSizeBasedOnWidth(text, richInput)
}

// 获取当前字体大小（step=2递减），text取自全局state
func getRichInputFontSize() float32 {
	return richInputFontSize
}

func getLabelFontSize() float32 {
	return labelFontSize
}

// 测量文本在特定字号下的物理宽度
func measureWidth(text string, fontSize float32) float32 {
	style := fyne.TextStyle{Bold: true}
	if fyne.CurrentApp() == nil || fyne.CurrentApp().Driver() == nil {
		return float32(len(text)) * fontSize // fallback
	}
	size, _ := fyne.CurrentApp().Driver().RenderedTextSize(text, fontSize, style, nil)
	return size.Width
}

type CalcState struct {
	win fyne.Window

	display           binding.String   // 当前输入的算式
	result            binding.String   // 当前算式的结果预览
	history           binding.String   // 历史记录（每次计算完成后追加）
	allHistoryBuilder *strings.Builder // 用于保存所有历史记录的字符串，方便写入文件
	saveFileName      string           // 本地文件名（如 "history.txt"）
	lastRecordDate    string           // 记录上一次写入时的日期（如 "2026-03-31"）

	isNewNumber bool // 是否正在输入一个新的数字（而不是继续在当前数字后面输入）

	isResultMode binding.Bool // true 代表显示结果（结果粗），false 代表输入中（输入粗)
	isCalcBig    binding.Bool // 是否使用高级计算布局
	isRadian     binding.Bool // true 为弧度模式，false 为角度模式
	is2ndMode    binding.Bool // 是否处于 2nd 模式

	isInterceptingForScore bool            // 是否正在拦截输入
	onScoreInput           func(string)    // 拦截时的回调函数
	scoreOverlay           *fyne.Container // 平摊功能的 UI 容器
}

// 构造函数，初始化状态
func NewCalcState(w fyne.Window) *CalcState {
	s := &CalcState{
		display:           binding.NewString(),
		result:            binding.NewString(),
		history:           binding.NewString(),
		allHistoryBuilder: &strings.Builder{},
		saveFileName:      "history.txt",
		isNewNumber:       true,
		isResultMode:      binding.NewBool(),
		isCalcBig:         binding.NewBool(),
		isRadian:          binding.NewBool(),
		is2ndMode:         binding.NewBool(),
		win:               w,
	}
	s.display.Set("")
	s.result.Set("0")
	s.isResultMode.Set(true) // 初始结果行粗
	s.isCalcBig.Set(false)
	s.isRadian.Set(false) // 默认角度模式
	s.is2ndMode.Set(false)
	return s
}

// 清除所有历史记录（包括内存和本地文件）
func (s *CalcState) ClearAllHistoryLocal() error {
	// 清除当前显示的当次历史
	s.history.Set("")
	s.allHistoryBuilder.Reset()

	rootURI := fyne.CurrentApp().Storage().RootURI()
	if rootURI == nil {
		return nil
	}

	fileURI, err := storage.Child(rootURI, "history.txt")
	if err != nil {
		return nil
	}

	// 删除本地文件
	err = storage.Delete(fileURI)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

// 记录历史：每次计算完成后调用，参数是算式和结果
func (s *CalcState) recordToHistory(expression string, result string) {
	now := time.Now().Format("2006-01-02")

	// 核心逻辑：对比缓存的日期
	// 只有当日期变了（或者是 APP 启动后的第一次记录），才插入标题
	if s.lastRecordDate != now {
		// 即使日期变了，我们也双重检查一下 Builder 内容（防止初始化时的边界问题）
		dateHeader := "--- " + now + " ---"

		// 只有当确实没包含这个标题时才写入
		// 注意：这里只在日期切换的那一刻调用一次 String()，平时不调用
		if !strings.Contains(s.allHistoryBuilder.String(), dateHeader) {
			s.allHistoryBuilder.WriteString("\n" + dateHeader + "\n")
		}

		// 更新缓存的日期
		s.lastRecordDate = now
	}

	// 假设每行格式为：算式 = 结果
	entry := fmt.Sprintf("%s = %s\n", expression, result)

	// 直接操作 string 字段
	s.allHistoryBuilder.WriteString(entry)
}

// 保存历史到文件：在应用退到后台或者被系统停止时调用
func (s *CalcState) saveHistoryToFile() {
	if s.allHistoryBuilder == nil || s.allHistoryBuilder.Len() == 0 {
		return // 如果没有内容，直接返回，避免覆写空文件
	}

	// 自动清理逻辑：防止内存中的 Builder 过大
	// 500KB 约等于 512,000 字节
	if s.allHistoryBuilder.Len() > 500*1024 {
		content := s.allHistoryBuilder.String()
		lines := strings.Split(content, "\n")

		if len(lines) > 5000 {
			// 保留最后 5000 行
			newContent := strings.Join(lines[len(lines)-5000:], "\n")

			// strings.Builder 不支持直接删除，必须重置后重新写入
			s.allHistoryBuilder.Reset()
			s.allHistoryBuilder.WriteString(newContent)
		}
	}

	// 【关键修改：使用 Fyne 的沙盒路径 API 读写文件】
	// 获取 App 专属的安全沙盒根目录
	rootURI := fyne.CurrentApp().Storage().RootURI()
	if rootURI == nil {
		return // 如果获取不到，直接返回
	}

	// 在根目录下构建/获取 history.txt 的完整路径对象
	fileURI, err := storage.Child(rootURI, s.saveFileName)
	if err != nil {
		return
	}

	// 打开 Writer 写入文件 (会自动覆盖并保存)
	writer, err := storage.Writer(fileURI)
	if err != nil {
		return
	}
	defer writer.Close()

	_, _ = writer.Write([]byte(s.allHistoryBuilder.String()))
}

// 从文件加载历史：在应用启动时调用，返回文件内容的字节切片
func (s *CalcState) loadHistoryFromFile() []byte {
	rootURI := fyne.CurrentApp().Storage().RootURI()
	if rootURI == nil {
		return nil
	}

	fileURI, err := storage.Child(rootURI, s.saveFileName)
	if err != nil {
		return nil
	}

	reader, err := storage.Reader(fileURI)
	if err != nil {
		// 文件不存在是正常的（第一次运行），直接返回空
		return nil
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil
	}
	return data
}
