package chessboard

import (
	"fmt"
	"image"
	"image/color"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/failosof/chessboard/config"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Widget struct {
	config config.Chessboard

	curSize         int
	prevSize        int
	squareSize      float32
	halfPointerSize f32.Point
	halfPieceSize   f32.Point
	hintSize        f32.Point
	hintCenter      f32.Point

	squareOriginCoordinates []image.Point

	pieceEventTargets []event.Filter

	boardDrawingOp   op.CallOp
	hintDrawingOp    op.CallOp
	squareDrawingOps []*op.CallOp

	dragID         pointer.ID
	draggingPos    f32.Point
	selectedSquare chess.Square
	selectedPiece  chess.Piece

	game         *chess.Game
	prevPosition *chess.Position

	mu sync.Mutex
}

func NewWidget(theme config.Chessboard) *Widget {
	return &Widget{
		game:                    chess.NewGame(chess.UseNotation(chess.UCINotation{})),
		config:                  theme,
		halfPointerSize:         f32.Pt(16, 16).Div(2), // assume for now
		squareOriginCoordinates: make([]image.Point, 64),
		pieceEventTargets:       make([]event.Filter, 64),
		squareDrawingOps:        make([]*op.CallOp, 64),
		selectedSquare:          chess.NoSquare,
		selectedPiece:           chess.NoPiece,
	}
}

func (w *Widget) Layout(gtx layout.Context) layout.Dimensions {
	w.curSize = util.Min(gtx.Constraints.Max.X, gtx.Constraints.Max.Y)

	if w.sizeChanged() {
		w.draggingPos = w.draggingPos.Mul(float32(w.curSize) / float32(w.prevSize))

		defer func() { w.prevSize = w.curSize }()

		img := w.config.BoardStyle.ImageFor(w.curSize)
		w.squareSize = float32(img.Bounds().Dx()) / 8
		w.halfPieceSize = f32.Pt(w.squareSize, w.squareSize).Div(2)
		cache := new(op.Ops)
		boardMacro := op.Record(cache)
		util.DrawImage(cache, img, image.Point{})
		w.boardDrawingOp = boardMacro.Stop()

		w.hintSize = f32.Pt(w.squareSize, w.squareSize).Div(3)
		w.hintCenter = w.hintSize.Div(2)

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

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: w,
			Kinds:  pointer.Press | pointer.Drag,
		})
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			hoveredSquare := NewSquare(e.Position, w.squareSize).ToChess()
			if e.Buttons.Contain(pointer.ButtonPrimary) {
				w.processLeftClick(gtx, e, hoveredSquare)
			} else if e.Buttons.Contain(pointer.ButtonSecondary) {
				w.processRightClick(gtx, e, hoveredSquare)
			}
		}
	}

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
	if w.game == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	curPosition := w.game.Position()
	imageSize := util.Floor(w.squareSize)
	pieceSize := image.Pt(imageSize, imageSize)

	if w.config.ShowLastMove {
		lastMove := w.getLastMove()
		if lastMove != nil {
			w.markSquare(gtx, lastMove.S1(), w.config.LastMoveColor)
			w.markSquare(gtx, lastMove.S2(), w.config.LastMoveColor)
		}
	}

	// todo: add flip support
	if w.sizeChanged() || w.prevPosition == nil || curPosition.Hash() != w.prevPosition.Hash() {
		clear(w.squareDrawingOps)
		for square, piece := range curPosition.Board().SquareMap() {
			if square != w.selectedSquare {
				cache := new(op.Ops)
				squareMacro := op.Record(cache)
				img := w.config.PiecesStyle.ImageFor(piece, imageSize)
				coords := w.squareOriginCoordinates[square]
				util.DrawImage(cache, img, coords)
				ops := squareMacro.Stop()
				w.squareDrawingOps[square] = &ops
			}
		}
	}

	for _, squareDrawingOp := range w.squareDrawingOps {
		if squareDrawingOp != nil {
			squareDrawingOp.Add(gtx.Ops)
		}
	}

	if w.game.Outcome() != chess.NoOutcome {
		return
	}

	if w.selectedSquare != chess.NoSquare {
		w.markSquare(gtx, w.selectedSquare, GrayColor)

		img := w.config.PiecesStyle.ImageFor(w.selectedPiece, imageSize)
		util.DrawImage(gtx.Ops, img, w.draggingPos.Round())

		if w.config.ShowLegalMoves {
			for _, move := range curPosition.ValidMoves() {
				if move.S1() == w.selectedSquare {
					position := w.squareOriginCoordinates[move.S2()]
					if curPosition.Board().Piece(move.S2()) == chess.NoPiece {
						position = position.Add(w.halfPieceSize.Round()).Sub(w.hintCenter.Round())
						rect := image.Rectangle{
							Min: position,
							Max: position.Add(w.hintSize.Round()),
						}
						util.DrawEllipse(gtx.Ops, rect, w.config.HintColor)
					} else {
						rect := image.Rectangle{Min: position, Max: position.Add(pieceSize)}
						util.DrawRectangle(gtx.Ops, rect, w.squareSize/10, w.config.HintColor)
					}
				}
			}
		}
	}

	clear(w.pieceEventTargets)
	for square := range curPosition.Board().SquareMap() {
		coords := w.squareOriginCoordinates[square]
		pieceClip := clip.Rect(image.Rectangle{Min: coords, Max: coords.Add(pieceSize)}).Push(gtx.Ops)
		event.Op(gtx.Ops, square)
		pieceClip.Pop()
		w.pieceEventTargets = append(w.pieceEventTargets, pointer.Filter{
			Target: square,
			Kinds:  pointer.Move | pointer.Press | pointer.Release | pointer.Drag | pointer.Cancel,
		})
	}

	for {
		ev, ok := gtx.Event(w.pieceEventTargets...)
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			hoveredSquare := NewSquare(e.Position, w.squareSize).ToChess()
			switch e.Kind {
			case pointer.Move:
				pointer.CursorGrab.Add(gtx.Ops)
			case pointer.Press:
				if e.Buttons.Contain(pointer.ButtonPrimary) {
					w.processLeftClick(gtx, e, hoveredSquare)
				}
			default:
				w.processDragging(gtx, e, hoveredSquare)
			}
		}
	}
}

func (w *Widget) processRightClick(gtx layout.Context, e pointer.Event, hoveredSquare chess.Square) {
	// todo: impl drawing on the board
}

func (w *Widget) processLeftClick(
	gtx layout.Context,
	e pointer.Event,
	hoveredSquare chess.Square,
) {
	curPosition := w.game.Position()
	hoveredPiece := curPosition.Board().Piece(hoveredSquare)

	if hoveredPiece.Color() == curPosition.Turn() {
		pointer.CursorGrabbing.Add(gtx.Ops)

		w.dragID = e.PointerID
		w.selectedPiece = hoveredPiece
		w.selectedSquare = hoveredSquare
		w.draggingPos = e.Position.Add(w.halfPointerSize).Sub(w.halfPieceSize)

		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 30)})
	} else if w.selectedSquare != hoveredSquare {
		if err := w.moveSelectedPieceTo(hoveredSquare); err != nil {
			w.unselectPiece(gtx)
		}

		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 25)}) // todo: make animation
	}
}

func (w *Widget) processDragging(gtx layout.Context, e pointer.Event, hoveredSquare chess.Square) {
	switch e.Kind {
	case pointer.Drag:
		if w.dragID == e.PointerID && w.selectedSquare != chess.NoSquare {
			pointer.CursorGrabbing.Add(gtx.Ops)
			w.draggingPos = e.Position.Add(w.halfPointerSize).Sub(w.halfPieceSize)
			if e.Priority < pointer.Grabbed {
				gtx.Execute(pointer.GrabCmd{
					Tag: w.selectedSquare,
					ID:  w.dragID,
				})
			}
		}
	case pointer.Release:
		if w.selectedSquare != chess.NoSquare {
			if err := w.moveSelectedPieceTo(hoveredSquare); err != nil {
				w.putPieceBack(gtx)
			} else {
				w.unselectPiece(gtx)
			}
		}
	case pointer.Cancel:
		w.unselectPiece(gtx)
		gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 25)})
	}
}

func (w *Widget) moveSelectedPieceTo(to chess.Square) error {
	if w.selectedSquare == chess.NoSquare {
		return fmt.Errorf("no square selected")
	}
	move := w.selectedSquare.String() + to.String()
	return w.game.MoveStr(move)
}

func (w *Widget) putPieceBack(gtx layout.Context) {
	w.draggingPos = SquareToPosition(w.selectedSquare, w.squareSize)
	pointer.CursorPointer.Add(gtx.Ops)
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 25)})
}

func (w *Widget) unselectPiece(gtx layout.Context) {
	w.draggingPos = f32.Point{}
	w.selectedSquare = chess.NoSquare
	w.selectedPiece = chess.NoPiece
	w.dragID = 0
	pointer.CursorPointer.Add(gtx.Ops)
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 25)})
}

func (w *Widget) getLastMove() (m *chess.Move) {
	moves := w.game.Moves()
	if len(moves) > 0 {
		m = moves[len(moves)-1]
	}
	return
}

func (w *Widget) markSquare(gtx layout.Context, square chess.Square, color color.NRGBA) {
	origin := w.squareOriginCoordinates[square]
	size := origin.Add(image.Pt(util.Round(w.squareSize), util.Round(w.squareSize)))
	selection := image.Rectangle{Min: origin, Max: size}
	util.DrawPane(gtx.Ops, selection, color)
}
