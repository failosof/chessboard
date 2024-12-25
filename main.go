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

	board := NewWidget(img, pt)

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
						paint.Fill(gtx.Ops, BackgroundColor)
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

/*
package main

import (
	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"image"
	"image/color"
	"os"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Draggable Rectangle"))

		var ops op.Ops
		var dragging bool
		var dragID pointer.ID
		var cursor pointer.Cursor
		//rect := clip.Rect{Min: image.Pt(0, 0), Max: size}
		size := image.Pt(100, 100)
		rect := image.Rect(0, 0, 100, 100)
		prevRect := rect

		for {
			switch e := w.Event().(type) {
			case app.DestroyEvent:
				os.Exit(0)
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				ColorBox(gtx, 1, rect, cursor, GreenColor)

				for {
					// Track mouse events (Press, Release, Move)
					ev, ok := e.Source.Event(pointer.Filter{
						Target: 1,
						Kinds:  pointer.Move | pointer.Press | pointer.Drag | pointer.Release | pointer.Cancel,
					})
					if !ok {
						break
					}

					if e, ok := ev.(pointer.Event); ok {
						switch e.Kind {
						case pointer.Move:
							if !dragging {
								if e.Position.Round().In(rect) {
									cursor = pointer.CursorGrab
								}
							}
						case pointer.Press:
							if !dragging {
								dragging = true
								rect.Min = e.Position.Round().Sub(size.Div(2))
								rect.Max = rect.Min.Add(size)
								dragID = e.PointerID
								cursor = pointer.CursorGrabbing
							}
						case pointer.Drag:
							cursor = pointer.CursorGrabbing
							rect.Min = e.Position.Round().Sub(size.Div(2))
							rect.Max = rect.Min.Add(size)
							if e.Priority < pointer.Grabbed {
								gtx.Execute(pointer.GrabCmd{
									Tag: 1,
									ID:  dragID,
								})
							}
						case pointer.Release:
							if !dragging {
								rect = prevRect
							}
							fallthrough
						case pointer.Cancel:
							prevRect = rect
							cursor = pointer.CursorDefault
							dragging = false
						}
					}
				}

				e.Frame(gtx.Ops)
			}
		}
	}()
	app.Main()
}

func ColorBox(gtx layout.Context, id int, rect image.Rectangle, cursor pointer.Cursor, color color.NRGBA) layout.Dimensions {
	defer clip.Rect(rect).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	event.Op(gtx.Ops, id)
	cursor.Add(gtx.Ops)
	return layout.Dimensions{Size: rect.Max}
}
*/
