package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// --- 自定义主题 ---
type myTheme struct{
	fyne.Theme
    textSize float32
}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForegroundOnPrimary:
		return color.NRGBA{R: 220, G: 235, B: 255, A: 255} // 淡蓝色背景
	// 1. 设置背景色（如你之前所设）
	//case theme.ColorNamePrimary:
	//	return color.NRGBA{R: 220, G: 235, B: 255, A: 255} // 淡蓝色背景
	//case theme.ColorNameError:
	//	return color.NRGBA{R: 255, G: 159, B: 10, A: 255} // 橙红背景
	//case theme.ColorNameButton:
	//	return color.NRGBA{R: 245, G: 245, B: 250, A: 255} // 近白背景

	// 2. 关键：强制设置前景色（字体颜色）
	//case theme.ColorNameForeground:
	// 这里控制普通文本、按钮文本的颜色
	//	return color.Black // 设为纯黑

	// 3. 针对高亮按钮（HighImportance）的字体颜色
	// Fyne 内部使用 ColorNameForegroundOnPrimary 来决定 HighImportance 上的文字颜色
	//case theme.ColorNameForegroundOnPrimary:
	//	return color.Black // 强制让深蓝色/高亮按钮上的字变黑色

	// 4. 针对警告按钮（WarningImportance）的字体颜色
	//case theme.ColorNameForegroundOnError:
	//	return color.Black // 强制让橙红色按钮上的字变黑色
	case theme.ColorNameScrollBar:
		return color.Transparent // 让滚动条透明

	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return m.textSize // 全局基础字体增大 (按键字体)
	case theme.SizeNameScrollBar:
		return 0 // 或者将滚动条宽度设为 0
	case BigFont:
		return 36 // 或者将滚动条宽度设为 0
	case LargeFont:
		return 30 // 或者将滚动条宽度设为 0
	case MediumFont:
		return 24 // 或者将滚动条宽度设为 0
	case SmallFont:
		return 18 // 或者将滚动条宽度设为 0
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

var (
	BigFont    fyne.ThemeSizeName = "BigFontSize"    // "BigFontSize 36"
	LargeFont  fyne.ThemeSizeName = "LargeFontSize"  // "BigFontSize 30"
	MediumFont fyne.ThemeSizeName = "MediumFontSize" //"MediumFontSize" 24
	SmallFont  fyne.ThemeSizeName = "SmallFontSize"  //"SmallFontSize" 18	
)
