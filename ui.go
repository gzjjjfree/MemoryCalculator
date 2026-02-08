package main

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func CreateUI(state *CalcState) fyne.CanvasObject {
	// --- 顶部 Tab 居中布局 ---
	calcLabel := widget.NewButton("计算", func() {})
	calcLabel.Importance = widget.LowImportance

	convertLabel := widget.NewButton("换算", func() {})
	convertLabel.Importance = widget.LowImportance

	// --- 定义全屏历史查看函数 ---
	showFullHistory := func() {}

	// 定义历史显示, 使用 RichText 获得更好的排版支持
	richHistory := newAllHistoryClickable()
	richHistory.Wrapping = fyne.TextWrapBreak

	// 定义输入框
	richInput := widget.NewRichText()
	// 设置对齐方式（通过段落样式）
	richInput.ExtendBaseWidget(richInput)
	richInput.Wrapping = fyne.TextWrapWord // 改为按单词换行

	// 定义结果显示
	lblResult := widget.NewLabelWithData(state.Result)
	lblResult.Alignment = fyne.TextAlignTrailing

	go func() {
		// 后台补全详细定义
		showFullHistory = func() {
			// 声明滚动容器变量，以便后面引用
			var scroller *container.Scroll

			renderHistory := func() fyne.CanvasObject {
				fullText := state.GetAllHistory()
				if fullText == "" {
					fullText = "暂无历史记录"
				}

				// 使用 RichText 获得更好的排版支持
				content := widget.NewRichText(
					&widget.TextSegment{
						Text: fullText,
						Style: widget.RichTextStyle{
							Alignment: fyne.TextAlignTrailing, // 设置向右对齐
							SizeName:  SmallFont,
						},
					},
				)
				// 必须设置换行模式
				content.Wrapping = fyne.TextWrapBreak

				// 将内容放入滚动容器
				scroller = container.NewVScroll(container.NewPadded(content))
				return scroller
			}

			historyWin := fyne.CurrentApp().NewWindow("全部历史记录")
			// 动态内容容器，方便清除后刷新显示
			contentHolder := container.NewStack(renderHistory())

			// 清除逻辑确认弹窗
			clearAction := func() {
				dialog.ShowConfirm("确认清除", "是否删除所有本地历史记录？此操作不可撤销。", func(ok bool) {
					if ok {
						err := state.ClearAllHistoryLocal()
						if err != nil {
							dialog.ShowError(err, historyWin)
						} else {
							// 刷新当前全屏窗口的显示
							contentHolder.Objects = []fyne.CanvasObject{widget.NewLabel("暂无历史记录")}
							contentHolder.Refresh()
						}
					}
				}, historyWin)
			}

			// 增加清除按钮（放在底部或右上角）
			clearBtn := widget.NewButtonWithIcon("清除全部", theme.DeleteIcon(), clearAction)

			// 布局：顶部标题 + 中间内容 + 底部按钮
			mainLayout := container.NewBorder(nil, clearBtn, nil, nil, contentHolder)

			historyWin.SetContent(mainLayout)
			historyWin.Resize(fyne.NewSize(360, 640))
			historyWin.Show()
			fyne.Do(func() { scroller.ScrollToBottom() })
		}
	}()

	// 即时历史显示框, 冒泡显示
	historyContainer := container.NewBorder(
		nil,                // Top
		richHistory,        // Bottom (强制文字靠底)
		nil,                // Left
		nil,                // Right
		layout.NewSpacer(), // Center (占据剩余所有空间)
	)

	// 将历史记录放在一个固定高度的滚动容器里
	scrollSession := container.NewVScroll(historyContainer)

	var isListener bool = true
	// --- 设置监听器 (现在它们控制的是上面那三个实例) ---
	refreshRichInput := func() {
		isBold, _ := state.IsResultMode.Get()
		lblResult.TextStyle = fyne.TextStyle{Bold: isBold}

		text, _ := state.Display.Get()

		actualW = richInput.Size().Width - theme.Padding()*4
		changeText, isFinal := updateFontSizeBasedOnWidth(text, richInput)

		if isBold {
			labelFontSize = 42 // 当 isBold=true（结果模式）时设为 42
			richInputFontSize = 18
		} else {
			labelFontSize = 18 // 当 isBold=false（输入模式）时设为 18
		}
		lblResult.SizeName = LabelFont

		stateMutex.Lock()
		if isFinal {
			isListener = false
			state.Display.Set(changeText)
			isListener = true
		}
		stateMutex.Unlock()

		// 更新 RichText 内容
		richInput.Segments = []widget.RichTextSegment{
			&widget.TextSegment{
				Text: changeText,
				Style: widget.RichTextStyle{
					TextStyle: fyne.TextStyle{Bold: !isBold}, // 非结果模式时加粗
					// 使用 Fyne 定义的 SizeName
					SizeName: RichInputFont,
					// 设置向右对齐
					Alignment: fyne.TextAlignTrailing,
				},
			},
		}

		richInput.Refresh()
		lblResult.Refresh()
	}

	// --- 创建不同的按键布局 ---
	calcGrid := createCalculatorGrid(state) // 开始时的计算器 4x5 布局
	var calcBigGrid fyne.CanvasObject

	// --- 定义动态按键容器 (Stack) --- Stack 容器会自动填满可用空间，并显示最上层的对象
	keypadContainer := container.NewStack(calcGrid)

	go func() { // 后台启动监听
		// 每次更新历史时，自动滚动到底部
		state.History.AddListener(binding.NewDataListener(func() {
			hText, _ := state.History.Get()

			richHistory.Segments = []widget.RichTextSegment{
				&widget.TextSegment{
					Text: hText,
					Style: widget.RichTextStyle{
						// 设置为禁用色（灰色）
						ColorName: theme.ColorNameDisabled,
						SizeName:  SmallFont,
						Alignment: fyne.TextAlignTrailing,
					},
				},
			}

			richHistory.Refresh()
			time.AfterFunc(time.Millisecond*50, func() {
				fyne.Do(func() { scrollSession.ScrollToBottom() })
			})
		}))

		state.Display.AddListener(binding.NewDataListener(func() {
			if isListener {
				refreshRichInput()
			}
		}))

		// --- 监听模式变化 (OnEqual 动作会触发这里) ---
		state.IsResultMode.AddListener(binding.NewDataListener(func() {
			refreshRichInput() // 按下等号时也要刷新一次颜色和粗细
		}))

		calcBigGrid = createConverterGrid(state) // 新的布局 5x7 布局
		state.IsCalcBig.AddListener(binding.NewDataListener(func() {
			if ok, _ := state.IsCalcBig.Get(); ok {
				keypadContainer.Objects = []fyne.CanvasObject{calcBigGrid}
				keypadContainer.Refresh()
			} else {
				keypadContainer.Objects = []fyne.CanvasObject{calcGrid}
				keypadContainer.Refresh()
			}
		}))
	}()

	// 设置显示全部历史
	historyIcon := widget.NewButtonWithIcon("", theme.HistoryIcon(), showFullHistory)
	historyIcon.Importance = widget.LowImportance

	// 下方输入区容器
	inputArea := container.NewVBox(
		richInput,
		lblResult,
	)

	// 最终的 topBar：两侧是 Spacer，中间是 Tabs，最右边是历史按钮
	topBar := container.NewBorder(nil, nil, nil, historyIcon,
		container.NewHBox(layout.NewSpacer(), calcLabel, convertLabel, layout.NewSpacer()),
	)

	displayArea := container.NewBorder(
		topBar,        // Top
		inputArea,     // Bottom
		nil,           // Left
		nil,           // Right
		scrollSession, // Center (自动填充)
	)

	//content := container.NewGridWithRows(2, displayArea, keypadContainer)
	content := container.New(&ratioLayout{ratio: 0.47}, displayArea, keypadContainer)

	// 创建一个透明的矩形作为底部的“垫片”，高度设置为 20
	bottomSpacer := canvas.NewRectangle(color.Transparent)
	bottomSpacer.SetMinSize(fyne.NewSize(0, 7))

	// 使用 Border 布局，底部（Bottom）放垫片，中间（Center）放你的按钮或计算器面板
	return container.NewBorder(nil, bottomSpacer, nil, nil, content)
}

func makeBtn(text string, icon fyne.Resource, style int, action func()) fyne.CanvasObject {
	var b *widget.Button
	if icon != nil {
		// 如果有图标，创建图标按钮（可以带文字，也可以 text 传 ""）
		b = widget.NewButtonWithIcon(text, icon, action)
	} else {
		// 否则创建普通文字按钮
		b = widget.NewButton(text, action)
	}

	colorFont := color.Gray16{Y: 0}
	colorBackground := color.NRGBA{R: 242, G: 242, B: 242, A: 255}
	if text >= "0" && text <= "9" && text != "1/x" && text != "2nd" {
		colorFont = color.Gray16{Y: 32768}
	}
	if text == "+" || text == "-" || text == "×" || text == "÷" || text == "xʸ" || text == "x!" ||
		text == "(" || text == "1/x" || text == ")" || text == "π" || text == "2nd" || text == "e" {
		colorBackground = color.NRGBA{R: 220, G: 235, B: 255, A: 255} // 淡蓝色背景
	}
	customTheme := &myTheme{Theme: theme.DefaultTheme(), textSize: 30, colorFont: colorFont, colorBackground: colorBackground}
	//customTheme := &myTheme{Theme: theme.DefaultTheme(), textSize: 30}
	container.NewThemeOverride(b, customTheme)

	switch style {
	case 0:
		b.Importance = widget.HighImportance
	case 1:
		b.Importance = widget.HighImportance
	case 2:
		b.Importance = widget.WarningImportance
	case 3:
		b.Importance = widget.HighImportance
	default:
		b.Importance = widget.MediumImportance
	}
	return container.NewStack(b)
}

func createConverterGrid(state *CalcState) fyne.CanvasObject {
	var degBtn *widget.Button

	colorBackground := color.NRGBA{R: 220, G: 235, B: 255, A: 255} // 淡蓝色背景
	customTheme := &myTheme{Theme: theme.DefaultTheme(), textSize: 30, colorBackground: colorBackground}

	// 使用一个特殊的构造逻辑或直接创建，以便拿到指针
	// 我们直接写一个闭包来生成这个特定按钮
	degBtn = widget.NewButton("DEG", state.OnDegToRad)
	degBtn.Importance = widget.HighImportance
	container.NewThemeOverride(degBtn, customTheme)
	// 将其包装进你想要的容器格式
	degBtnObj := container.NewStack(degBtn)

	// 为 IsRadian 增加监听器，实现 UI 自动同步
	state.IsRadian.AddListener(binding.NewDataListener(func() {
		isRad, _ := state.IsRadian.Get()
		if isRad {
			degBtn.SetText("RAD")
		} else {
			degBtn.SetText("DEG")
		}
	}))

	// 定义一个用于存储需要切换文本的按钮的 Map
	toggleButtons := make(map[string]*widget.Button)

	// 修改后的快捷创建函数，会将按钮存入 map
	makeToggleBtn := func(id string, style int) fyne.CanvasObject {
		btn := widget.NewButton(id, func() { state.OnAdvancedTap(id) })
		toggleButtons[id] = btn // 存入引用

		colorBackground := color.NRGBA{R: 220, G: 235, B: 255, A: 255} // 淡蓝色背景
		customTheme := &myTheme{Theme: theme.DefaultTheme(), textSize: 30, colorBackground: colorBackground}
		container.NewThemeOverride(btn, customTheme)
		// 设置样式（同makeBtn 逻辑）
		if style == 1 {
			btn.Importance = widget.HighImportance
		}
		return container.NewStack(btn)
	}

	//定义 2nd 模式切换逻辑
	state.Is2ndMode.AddListener(binding.NewDataListener(func() {
		is2nd, _ := state.Is2ndMode.Get()

		// 定义对应的文字映射
		labels := map[string][2]string{
			"sin": {"sin", "asin"},
			"cos": {"cos", "acos"},
			"tan": {"tan", "atan"},
			"lg":  {"lg", "10ˣ"},
			"ln":  {"ln", "eˣ"},
			"√x":  {"√x", "x²"},
		}

		for id, btn := range toggleButtons {
			if pair, ok := labels[id]; ok {
				if is2nd {
					btn.SetText(pair[1])
				} else {
					btn.SetText(pair[0])
				}
			}
		}
	}))

	grid := container.NewGridWithColumns(5,
		makeBtn("2nd", nil, 1, state.OnToggle2nd),
		degBtnObj,
		makeToggleBtn("sin", 1), // 改为调用高级功能
		makeToggleBtn("cos", 1),
		makeToggleBtn("tan", 1),

		makeBtn("xʸ", nil, 1, func() { state.OnTap("^") }), // 幂运算通常需要输入两个数
		makeToggleBtn("lg", 1),
		makeToggleBtn("ln", 1),
		makeBtn("(", nil, 1, func() { state.OnTap("(") }),
		makeBtn(")", nil, 1, func() { state.OnTap(")") }),

		makeToggleBtn("√x", 1),
		makeBtn("C", nil, 2, state.OnClear),
		makeBtn("⌫", nil, 0, state.OnBackspace),
		makeBtn("%", nil, 0, func() { state.OnTap("%") }),
		makeBtn("÷", nil, 1, func() { state.OnTap("÷") }),

		makeBtn("x!", nil, 1, func() { state.OnAdvancedTap("x!") }),
		makeBtn("7", nil, 0, func() { state.OnTap("7") }),
		makeBtn("8", nil, 0, func() { state.OnTap("8") }),
		makeBtn("9", nil, 0, func() { state.OnTap("9") }),
		makeBtn("×", nil, 1, func() { state.OnTap("×") }),

		makeBtn("1/x", nil, 1, func() { state.OnAdvancedTap("1/x") }),
		makeBtn("4", nil, 0, func() { state.OnTap("4") }),
		makeBtn("5", nil, 0, func() { state.OnTap("5") }),
		makeBtn("6", nil, 0, func() { state.OnTap("6") }),
		makeBtn("-", nil, 1, func() { state.OnTap("-") }),

		makeBtn("π", nil, 1, func() { state.OnAdvancedTap("π") }),
		makeBtn("1", nil, 0, func() { state.OnTap("1") }),
		makeBtn("2", nil, 0, func() { state.OnTap("2") }),
		makeBtn("3", nil, 0, func() { state.OnTap("3") }),
		makeBtn("+", nil, 1, func() { state.OnTap("+") }),

		makeBtn("", theme.GridIcon(), 0, state.OnGoBigGrid),
		makeBtn("e", nil, 1, func() { state.OnAdvancedTap("e") }),
		makeBtn("0", nil, 0, func() { state.OnTap("0") }),
		makeBtn(".", nil, 0, func() { state.OnTap(".") }),
		makeBtn("=", nil, 2, state.OnEqual),
	)

	return grid
}

func createCalculatorGrid(state *CalcState) fyne.CanvasObject {
	grid := container.NewGridWithColumns(4,
		makeBtn("C", nil, 2, state.OnClear),
		makeBtn("⌫", nil, 0, state.OnBackspace),
		makeBtn("%", nil, 0, func() {}),
		makeBtn("÷", nil, 3, func() { state.OnTap("÷") }),

		makeBtn("7", nil, 0, func() { state.OnTap("7") }),
		makeBtn("8", nil, 0, func() { state.OnTap("8") }),
		makeBtn("9", nil, 0, func() { state.OnTap("9") }),
		makeBtn("×", nil, 3, func() { state.OnTap("×") }),

		makeBtn("4", nil, 0, func() { state.OnTap("4") }),
		makeBtn("5", nil, 0, func() { state.OnTap("5") }),
		makeBtn("6", nil, 0, func() { state.OnTap("6") }),
		makeBtn("-", nil, 3, func() { state.OnTap("-") }),

		makeBtn("1", nil, 0, func() { state.OnTap("1") }),
		makeBtn("2", nil, 0, func() { state.OnTap("2") }),
		makeBtn("3", nil, 0, func() { state.OnTap("3") }),
		makeBtn("+", nil, 3, func() { state.OnTap("+") }),

		makeBtn("", theme.GridIcon(), 0, state.OnGoBigGrid),
		makeBtn("0", nil, 0, func() { state.OnTap("0") }),
		makeBtn(".", nil, 0, func() { state.OnTap(".") }),
		makeBtn("=", nil, 2, state.OnEqual),
	)

	return grid
}

type allHistoryClickable struct {
	widget.RichText
}

// 处理点击事件
func (t *allHistoryClickable) Tapped(_ *fyne.PointEvent) {
	if len(t.Segments) == 0 {
		return
	}

	// 先根据第一个片段判断当前状态（是灰还是亮）
	firstSeg := t.Segments[0].(*widget.TextSegment)
	targetColor := theme.ColorNameForeground
	if firstSeg.Style.ColorName == theme.ColorNameForeground {
		targetColor = theme.ColorNameDisabled
	}

	// 遍历所有片段，统一修改颜色
	for _, seg := range t.Segments {
		if txtSeg, ok := seg.(*widget.TextSegment); ok {
			txtSeg.Style.ColorName = targetColor
		}
	}

	// 刷新 UI
	t.Refresh()
}

func newAllHistoryClickable() *allHistoryClickable {
	t := &allHistoryClickable{}

	t.ExtendBaseWidget(t) // 必须在这里扩展
	// 关键：触发内部初始化，确保 MinSize 计算正常
	t.Segments = []widget.RichTextSegment{}
	t.Scroll = container.ScrollNone
	return t
}

type ratioLayout struct {
	ratio float32 // 上部占比，如 0.4
}

func (r *ratioLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	topHeight := size.Height * r.ratio
	objects[0].Resize(fyne.NewSize(size.Width, topHeight))
	objects[0].Move(fyne.NewPos(0, 0))

	objects[1].Resize(fyne.NewSize(size.Width, size.Height-topHeight))
	objects[1].Move(fyne.NewPos(0, topHeight))
}

func (r *ratioLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(100, 100) // 设置一个基础最小尺寸
}
