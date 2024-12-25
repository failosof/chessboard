package main

import (
	"fmt"
	"image"
	"log/slog"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/failosof/chessboard/theme"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Widget struct {
	originalImage image.Image
	piecesTheme   theme.PiecesTheme

	halfPointerSize f32.Point

	curSize    int
	prevSize   int
	squareSize float32

	hint       image.Point
	hintCenter image.Point

	boardDrawingOp   op.CallOp
	hintDrawingOp    op.CallOp
	squareDrawingOps []op.CallOp

	squareOriginCoordinates []image.Point

	game         *chess.Game
	prevPosition *chess.Position

	dragID         pointer.ID
	draggingPos    image.Point
	selectedSquare chess.Square
	selectedPiece  chess.Piece

	mu sync.Mutex
}

func NewWidget(boardImage image.Image, piecesTheme theme.PiecesTheme) *Widget {
	return &Widget{
		game:                    chess.NewGame(chess.UseNotation(chess.UCINotation{})),
		originalImage:           boardImage,
		piecesTheme:             piecesTheme,
		halfPointerSize:         f32.Pt(16, 16).Div(2), // assume for now
		squareOriginCoordinates: make([]image.Point, 64),
		selectedSquare:          chess.NoSquare,
		selectedPiece:           chess.NoPiece,
	}
}

func (w *Widget) Layout(gtx layout.Context) layout.Dimensions {
	w.curSize = util.Min(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)
	if w.sizeChanged() {
		defer func() { w.prevSize = w.curSize }()

		img := util.ResizeImage(w.originalImage, w.curSize, w.curSize)
		w.squareSize = float32(img.Bounds().Dx()) / 8
		cache := new(op.Ops)
		boardMacro := op.Record(cache)
		util.DrawImage(cache, img, image.Point{})
		w.boardDrawingOp = boardMacro.Stop()

		hintSize := util.Round(w.squareSize / 3)
		w.hint = image.Pt(hintSize, hintSize)
		w.hintCenter = w.hint.Div(2)

		for square := chess.A1; square <= chess.H8; square++ {
			w.squareOriginCoordinates[square] = SquareToPosition(square, w.squareSize).Round()
		}
	}

	w.boardDrawingOp.Add(gtx.Ops)

	boardSize := image.Pt(w.curSize, w.curSize)
	defer clip.Rect(image.Rectangle{Max: boardSize}).Push(gtx.Ops).Pop()
	pointer.CursorPointer.Add(gtx.Ops)
	event.Op(gtx.Ops, w)

	w.drawPieces(gtx)

	return layout.Dimensions{Size: boardSize}
}

func (w *Widget) SetGame(game *chess.Game) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.game = game
}

func (w *Widget) sizeChanged() bool {
	return w.curSize != w.prevSize
}

func (w *Widget) drawPieces(gtx layout.Context) {
	// todo: get rid of memory allocations

	if w.game == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	curPosition := w.game.Position()
	imageSize := util.Floor(w.squareSize)
	pieceSize := image.Pt(imageSize, imageSize)

	if w.sizeChanged() || w.prevPosition == nil || curPosition.Hash() != w.prevPosition.Hash() {
		w.squareDrawingOps = make([]op.CallOp, 64)
		for square, piece := range curPosition.Board().SquareMap() {
			if square != w.selectedSquare {
				cache := new(op.Ops)
				squareMacro := op.Record(cache)
				img := w.piecesTheme.ImageFor(piece, imageSize)
				coords := w.squareOriginCoordinates[square]
				util.DrawImage(cache, img, coords)
				w.squareDrawingOps[square] = squareMacro.Stop()
			}
		}
	}

	for _, squareDrawingOp := range w.squareDrawingOps {
		squareDrawingOp.Add(gtx.Ops)
	}

	if w.selectedSquare != chess.NoSquare {
		img := w.piecesTheme.ImageFor(w.selectedPiece, imageSize)
		// todo: draw selection
		util.DrawImage(gtx.Ops, img, w.draggingPos)
	}

	targets := make([]event.Filter, 0, 64)
	for square := range curPosition.Board().SquareMap() {
		coords := w.squareOriginCoordinates[square]
		pieceClip := clip.Rect(image.Rectangle{Min: coords, Max: coords.Add(pieceSize)}).Push(gtx.Ops)
		event.Op(gtx.Ops, square)
		pieceClip.Pop()
		targets = append(targets, pointer.Filter{
			Target: square,
			Kinds:  pointer.Move | pointer.Press | pointer.Release | pointer.Drag | pointer.Cancel,
		})
	}

	for {
		ev, ok := gtx.Event(targets...)
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			square := NewSquare(e.Position, w.squareSize).ToChess()
			piece := curPosition.Board().Piece(square)
			switch e.Kind {
			case pointer.Move:
				if w.selectedSquare == chess.NoSquare && piece.Color() == curPosition.Turn() {
					slog.Debug("hovering", "piece", piece.String())
					pointer.CursorGrab.Add(gtx.Ops)
				}
			case pointer.Press:
				if w.selectedSquare == chess.NoSquare && piece != chess.NoPiece && piece.Color() == curPosition.Turn() {
					pointer.CursorGrabbing.Add(gtx.Ops)
					name := fmt.Sprintf("%s %s", piece.Color().Name(), piece.Type().String())
					slog.Debug("selected", "piece", name)
					w.selectedPiece = piece
					w.selectedSquare = square
					w.draggingPos = e.Position.Sub(f32.Pt(w.squareSize, w.squareSize).Div(2)).Round()
				}
			case pointer.Drag:
				if w.selectedSquare != chess.NoSquare {
					pointer.CursorGrabbing.Add(gtx.Ops)
					halfPieceSize := f32.Pt(w.squareSize, w.squareSize).Div(2)
					w.draggingPos = e.Position.Add(w.halfPointerSize).Sub(halfPieceSize).Round()
					if e.Priority < pointer.Grabbed {
						gtx.Execute(pointer.GrabCmd{
							Tag: w.selectedSquare,
							ID:  w.dragID,
						})
					}
				}
			case pointer.Release:
				if w.selectedSquare != chess.NoSquare {
					slog.Debug("released", "piece", w.selectedPiece, "on", square.String())
					if err := w.game.MoveStr(w.selectedSquare.String() + square.String()); err == nil {
						w.dragID = 0
						w.draggingPos = image.Point{}
						w.selectedSquare = chess.NoSquare
						w.selectedPiece = chess.NoPiece
						pointer.CursorPointer.Add(gtx.Ops)
						gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 10)})
						continue
					} else {
						slog.Error("move", "error", err)
					}
				}
				fallthrough
			case pointer.Cancel:
				w.draggingPos = SquareToPosition(w.selectedSquare, w.squareSize).Round()
				w.selectedSquare = chess.NoSquare
				w.selectedPiece = chess.NoPiece
				gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 10)})
			default:
				slog.Warn("unreachable", "type", e.Kind)
			}
		}
	}
}
