package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

var state *CalcState

func main() {

	myApp := app.NewWithID("com.gzjjj.memorycalculator")
	myApp.Settings().SetTheme(&myTheme{Theme: theme.DefaultTheme(), textSize: 24})

	win := myApp.NewWindow("计算器")

	state = NewCalcState(win)
	ui := CreateUI(state)

	state.scoreOverlay = container.NewStack()
	state.scoreOverlay.Hide()
	contentStack := container.NewStack(
		ui,                 // 底层：原来的计算器全部界面
		state.scoreOverlay, // 顶层：平时隐藏，触发时显示的平摊提示框
	)
	win.SetContent(contentStack)

	//win.SetContent(container.NewPadded(ui))
	//win.SetContent(ui)
	win.Resize(fyne.NewSize(360, 640))

	win.ShowAndRun()
}
