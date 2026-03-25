package main

import (
	"io"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var isChangeRow bool = false // 是否需要换行
var stateMutex sync.Mutex
var richInputFontSize float32 = 42
var labelFontSize float32 = 18

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
			operators := "+-×÷*/"

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
func getFontSize() float32 {
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

// 清除所有历史记录（包括内存和本地文件）
func (s *CalcState) ClearAllHistoryLocal() error {
	// 清除当前显示的当次历史
	s.History.Set("")

	// 删除本地文件
	storage := fyne.CurrentApp().Storage()
	err := storage.Remove("history.txt")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

// 读取全部历史（给全屏界面用）
func (s *CalcState) GetAllHistory() string {
	storage := fyne.CurrentApp().Storage()

	// 检查文件是否存在
	reader, err := storage.Open("history.txt")
	if err != nil {
		// 文件不存在是正常的（第一次运行），直接返回空
		return ""
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return ""
	}
	return string(data)
}

// 核心保存逻辑：带日期检测
func (s *CalcState) appendToLocalFile(newRecord string) {
	existingHistory := s.GetAllHistory()

	now := time.Now().Format("2006-01-02")

	// 读取文件看最后一行是不是今天的日期

	var finalWrite strings.Builder
	// 如果文件为空，或者最后一次记录里不包含今天的日期标题
	if existingHistory == "" || !strings.Contains(existingHistory, "--- "+now+" ---") {
		finalWrite.WriteString("\n--- " + now + " ---\n")
	}
	finalWrite.WriteString(newRecord + "\n")

	// 合并内容
	allContent := existingHistory + finalWrite.String()

	if len(allContent) > 500*1024 { // 如果文件大于 500KB
		lines := strings.Split(allContent, "\n")
		if len(lines) > 5000 { // 至少清理掉一半，减少操作频率
			allContent = strings.Join(lines[len(lines)-5000:], "\n")
		}
	}

	// 写入
	storage := fyne.CurrentApp().Storage()
	writer, err := storage.Create("history.txt")

	if err != nil {
		writer, err = storage.Save("history.txt")
		if err != nil {
			s.History.Set("创建失败:" + err.Error())
			return
		}
	}
	defer writer.Close()

	_, err = writer.Write([]byte(allContent))
	if err != nil {
		s.History.Set("写入失败:" + err.Error())
	}
}
