package internal

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// 自定义主题结构体
type CustomTheme struct{}

// 实现 Theme 接口的方法
func (CustomTheme) Color(colorName fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch colorName {
	case theme.ColorNameDisabled:
		return color.Black // 设置禁用状态下的字体颜色为黑色
	case theme.ColorNameBackground:
		return color.White // 设置前景色为白色
	default:
		return theme.DefaultTheme().Color(colorName, variant)
	}
}

func (CustomTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (CustomTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (CustomTheme) Size(size fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(size)
}
