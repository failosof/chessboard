package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/failosof/chessboard"
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
	config, err := chessboard.NewConfig("assets/board/brown.png", "assets/pieces/aquarium")
	if err != nil {
		return err
	}

	config.ShowHints = true
	config.ShowLastMove = true

	board := chessboard.NewWidget(config)

	th := material.NewTheme()

	var frameCount int
	var fps float64
	startTime := time.Now()

	// Frame ticker for FPS calculation
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			elapsed := time.Since(startTime).Seconds()
			fps = float64(frameCount) / elapsed
			frameCount = 0
			startTime = time.Now()
		}
	}()

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			gtx.Execute(op.InvalidateCmd{})
			layout.Background{}.Layout(
				gtx,
				func(gtx layout.Context) layout.Dimensions {
					return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						defer clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops).Pop()
						paint.Fill(gtx.Ops, util.GrayColor)
						return layout.Dimensions{Size: gtx.Constraints.Min}
					})
				},
				func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{}.Layout(
						gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(20)).Layout(gtx, board.Layout)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(20)).Layout(
								gtx,
								func(gtx layout.Context) layout.Dimensions {
									return material.H4(th, fmt.Sprintf("FPS: %.2f", fps)).Layout(gtx)
								},
							)
						}),
					)
				},
			)
			e.Frame(gtx.Ops)
			frameCount++
		}
	}
}
