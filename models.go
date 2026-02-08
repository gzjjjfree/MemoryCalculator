package main

import (
	"io"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

var isChangeRow bool = false
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
			lastOpIdxInLine := strings.LastIndexAny(lastLine, operators)
			if lastOpIdxInLine != -1 {
				absoluteOpIdx := searchStart + lastOpIdxInLine
				changeText = changeText[:absoluteOpIdx] + "\n" + changeText[absoluteOpIdx:]
			}
			return changeText, true
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
	if richInputFontSize == 18 {
		return text, false
	}
	stateMutex.Lock()
	isChangeRow = true
	stateMutex.Unlock()
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
