package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

func main() {
	myApp := app.NewWithID("com.gzjjj.memorycalculator")
	myApp.Settings().SetTheme(&myTheme{Theme: theme.DefaultTheme(), textSize: 24})

	win := myApp.NewWindow("计算器")

	state := NewCalcState()
	ui := CreateUI(state)

	win.SetContent(container.NewPadded(ui))
	//win.SetContent(ui)
	win.Resize(fyne.NewSize(360, 640))

	win.ShowAndRun()
}
