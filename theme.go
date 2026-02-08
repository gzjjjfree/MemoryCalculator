package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// --- 自定义主题 ---
type myTheme struct {
	fyne.Theme
	textSize  float32
	colorFont color.Gray16
	colorBackground color.NRGBA
}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameForegroundOnPrimary:		
		return m.colorFont
	case theme.ColorNamePrimary:
		return m.colorBackground // color.NRGBA{R: 220, G: 235, B: 255, A: 255} // 淡蓝色背景
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
		return 36
	case LargeFont:
		return 30
	case MediumFont:
		return 24
	case SmallFont:
		return 18
	case RichInputFont:
		return getFontSize()
	case LabelFont:
		return getLabelFontSize()
	default:
		return theme.DefaultTheme().Size(name)
	}
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

var (
	BigFont       fyne.ThemeSizeName = "BigFontSize"       // "BigFontSize 36"
	LargeFont     fyne.ThemeSizeName = "LargeFontSize"     // "BigFontSize 30"
	MediumFont    fyne.ThemeSizeName = "MediumFontSize"    //"MediumFontSize" 24
	SmallFont     fyne.ThemeSizeName = "SmallFontSize"     //"SmallFontSize" 18
	RichInputFont fyne.ThemeSizeName = "RichInputFontSize" //func getFontSize()
	LabelFont	  fyne.ThemeSizeName = "LabelFontSize"      // "LabelFontSize" 14
)
