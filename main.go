package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

var state *CalcState

func main() {
	// 创建应用并设置自定义主题
	myApp := app.NewWithID("com.gzjjj.memorycalculator")
	myApp.Settings().SetTheme(&myTheme{Theme: theme.DefaultTheme(), textSize: 24})

	win := myApp.NewWindow("计算器")

	// 初始化全局状态
	state = NewCalcState(win)

	// 在后台加载历史记录，避免界面卡顿
	go func() {
		state.allHistoryBuilder.Write(state.loadHistoryFromFile())
	}()

	// 创建UI界面
	ui := CreateUI(state)

	// 创建一个Stack容器，底层是原来的计算器界面，顶层是平摊提示框（初始隐藏）
	state.scoreOverlay = container.NewStack()
	state.scoreOverlay.Hide()
	contentStack := container.NewStack(
		ui,                 // 底层：原来的计算器全部界面
		state.scoreOverlay, // 顶层：平时隐藏，触发时显示的平摊提示框
	)
	win.SetContent(contentStack)

	win.Resize(fyne.NewSize(360, 640))

	// 当应用退到后台（例如按了 Home 键），或者被系统停止时触发保存
	myApp.Lifecycle().SetOnExitedForeground(func() {
		state.saveHistoryToFile()
	})
	myApp.Lifecycle().SetOnStopped(func() {
		state.saveHistoryToFile()
	})

	win.ShowAndRun()
}
