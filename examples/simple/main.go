package main

import (
	"log/slog"
	"os"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/failosof/chessboard"
	"github.com/failosof/chessboard/theme"
	"github.com/failosof/chessboard/util"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	go func() {
		if err := draw(new(app.Window)); err != nil {
			slog.Error("can't draw window", "error", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

func draw(window *app.Window) error {
	filename := "assets/board/brown.png"
	img, err := util.OpenImage(filename)
	if err != nil {
		return err
	}

	pt, err := theme.LoadPiecesTheme("assets/pieces/aquarium")
	if err != nil {
		return err
	}

	board := chessboard.NewWidget(img, pt)

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			layout.Background{}.Layout(
				gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, chessboard.BackgroundColor)
						return layout.Dimensions{Size: gtx.Constraints.Min}
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return board.Layout(gtx)
					})
				},
			)
			e.Frame(gtx.Ops)
		}
	}
}
