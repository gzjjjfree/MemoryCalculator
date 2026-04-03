package main

import (
	"image/color"
	"strings"
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

// --- UI 构建 ---
func CreateUI(state *CalcState) fyne.CanvasObject {
	// --- 顶部 Tab 居中布局 ---
	calcLabel := widget.NewButton("计算", func() {})
	calcLabel.Importance = widget.LowImportance

	convertLabel := widget.NewButton("换算", func() {})
	convertLabel.Importance = widget.LowImportance

	// --- 定义全屏历史查看函数 ---
	showFullHistory := func() {
		historyWin := fyne.CurrentApp().NewWindow("全部历史记录")
		historyWin.Resize(fyne.NewSize(360, 640))

		// 将 Builder 内容转为切片，过滤掉可能的空行
		rawLines := strings.Split(state.AllHistoryBuilder.String(), "\n")
		var data []string
		for _, line := range rawLines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			data = append(data, trimmed)
		}

		// 创建 List 组件
		list := widget.NewList(
			// 返回数据总量
			func() int {
				return len(data)
			},
			// 创建单元格外观（模板）
			func() fyne.CanvasObject {
				label := widget.NewLabel("")
				label.Alignment = fyne.TextAlignTrailing
				label.Wrapping = fyne.TextWrapBreak               // 允许在长等式处自动换行
				label.TextStyle = fyne.TextStyle{Monospace: true} // 等宽字体更整齐
				labelFontSize = 18
				label.SizeName = LabelFont

				return container.NewPadded(label)
			},
			// 绑定数据到单元格（滚动时会被频繁调用）
			func(id widget.ListItemID, item fyne.CanvasObject) {
				text := data[id]
				label := item.(*fyne.Container).Objects[0].(*widget.Label)

				// 如果是日期标题，可以做特殊样式处理
				if strings.HasPrefix(text, "---") {
					label.Alignment = fyne.TextAlignCenter
					label.TextStyle = fyne.TextStyle{Bold: true}
				} else {
					label.Alignment = fyne.TextAlignTrailing
					label.TextStyle = fyne.TextStyle{}
				}

				label.SetText(text)
			},
		)

		// 清除逻辑
		clearBtn := widget.NewButtonWithIcon("清除全部", theme.DeleteIcon(), func() {
			dialog.ShowConfirm("确认", "确定删除吗？", func(ok bool) {
				if ok {
					state.ClearAllHistoryLocal()
					data = []string{} // 清空本地索引
					list.Refresh()    // 刷新列表
				}
			}, historyWin)
		})

		// 布局
		historyWin.SetContent(container.NewBorder(nil, clearBtn, nil, nil, list))
		historyWin.Show()

		// 自动滚动到底部（在列表渲染完成后执行）
		if len(data) > 0 {
			list.ScrollToBottom()
		}
	}

	// 定义历史显示, 使用 RichText 获得更好的排版支持
	richHistory := newAllHistoryClickable()
	richHistory.Wrapping = fyne.TextWrapBreak

	// 修改默认主题，去除上下边框阴影
	colorNameShadow := color.Transparent // color.Transparent
	customTheme := &myTheme{Theme: theme.DefaultTheme(), colorNameShadow: colorNameShadow}
	container.NewThemeOverride(richHistory, customTheme)

	// 定义输入框
	richInput := widget.NewRichText()
	// 设置对齐方式（通过段落样式）
	richInput.ExtendBaseWidget(richInput)
	richInput.Wrapping = fyne.TextWrapWord // 改为按单词换行

	// 定义结果显示
	lblResult := widget.NewLabelWithData(state.Result)
	lblResult.Alignment = fyne.TextAlignTrailing

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

		// 根据当前输入框宽度和文本内容动态调整字体大小
		actualW = richInput.Size().Width - theme.Padding()*4 // 留出一些内边距空间
		changeText, isFinal := updateFontSizeBasedOnWidth(text, richInput)

		// 根据是否是结果模式调整字体大小和样式
		if isBold {
			textresult, _ := state.Result.Get()
			fontSizes := []float32{42, 40, 38, 36, 34, 32, 30, 28, 26, 24, 22, 20, 18}
			for _, size := range fontSizes {
				if measureWidth(textresult, size) <= actualW {
					labelFontSize = size
					break
				}
			}
			richInputFontSize = 18 // 输入模式字体固定为 18，结果模式字体会根据内容自动调整
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

	content := container.New(&ratioLayout{ratio: 0.47}, displayArea, keypadContainer)

	// 创建一个透明的矩形作为底部的“垫片”，高度设置为 20
	bottomSpacer := canvas.NewRectangle(color.Transparent)
	bottomSpacer.SetMinSize(fyne.NewSize(0, 7))

	// 使用 Border 布局，底部（Bottom）放垫片，中间（Center）放你的按钮或计算器面板
	return container.NewBorder(nil, bottomSpacer, nil, nil, content)
}

// style: 0-普通，1-强调，2-警告，3-成功
// 根据文本内容自动设置字体颜色和背景颜色，并创建按钮
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

// 创建一个新的按键布局，包含更多科学计算功能
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

// 创建一个新的按键布局，包含基本的计算功能（4x5 布局）
func createCalculatorGrid(state *CalcState) fyne.CanvasObject {
	grid := container.NewGridWithColumns(4,
		makeBtn("C", nil, 2, state.OnClear),
		makeBtn("⌫", nil, 0, state.OnBackspace),
		makeBtn("%", nil, 0, func() { state.OnTap("%") }),
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

// 定义一个新的 RichText 组件，重写 Tapped 方法实现点击切换颜色的功能
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

// 创建一个新的 allHistoryClickable 实例，并进行必要的初始化
func newAllHistoryClickable() *allHistoryClickable {
	t := &allHistoryClickable{}

	t.ExtendBaseWidget(t) // 必须在这里扩展
	// 关键：触发内部初始化，确保 MinSize 计算正常
	t.Segments = []widget.RichTextSegment{}
	t.Scroll = container.ScrollNone
	return t
}

// 定义一个自定义布局，按照给定的比例分配上下两个区域的空间
type ratioLayout struct {
	ratio float32 // 上部占比，如 0.4
}

// 实现 Layout 方法，根据给定的 size 和 ratio 分配空间
func (r *ratioLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	topHeight := size.Height * r.ratio
	objects[0].Resize(fyne.NewSize(size.Width, topHeight))
	objects[0].Move(fyne.NewPos(0, 0))

	objects[1].Resize(fyne.NewSize(size.Width, size.Height-topHeight))
	objects[1].Move(fyne.NewPos(0, topHeight))
}

// 实现 MinSize 方法，返回一个合理的最小尺寸，避免过小导致布局混乱
func (r *ratioLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(100, 100) // 设置一个基础最小尺寸
}
