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
	"github.com/failosof/chessboard/config"
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
	th, err := config.Load("assets/board/brown.png", "assets/pieces/aquarium")
	if err != nil {
		return err
	}

	th.ShowLegalMoves = true
	th.ShowLastMove = true
	th.HintColor = chessboard.Transparentize(chessboard.GrayColor, 0.5)
	th.LastMoveColor = chessboard.Transparentize(chessboard.YellowColor, 0.7)

	board := chessboard.NewWidget(th)

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
						paint.Fill(gtx.Ops, chessboard.GrayColor)
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
