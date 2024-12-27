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
	"github.com/failosof/chessboard/size"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Widget struct {
	config config.Chessboard

	curBoardSize    size.Union
	prevBoardSize   size.Union
	squareSize      size.Union
	halfPointerSize size.Union
	halfPieceSize   size.Union
	hintSize        size.Union

	hintCenter f32.Point

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
		halfPointerSize:         size.FromInt(16 / 2), // assume for now
		squareOriginCoordinates: make([]image.Point, 64),
		pieceEventTargets:       make([]event.Filter, 64),
		squareDrawingOps:        make([]*op.CallOp, 64),
		selectedSquare:          chess.NoSquare,
		selectedPiece:           chess.NoPiece,
	}
}

func (w *Widget) Layout(gtx layout.Context) layout.Dimensions {
	w.curBoardSize = size.FromMinPt(gtx.Constraints.Max)

	if w.resized() {
		w.draggingPos = w.draggingPos.Mul(w.curBoardSize.Float / w.prevBoardSize.Float)

		defer func() { w.prevBoardSize = w.curBoardSize }()

		cache := new(op.Ops)
		boardMacro := op.Record(cache)
		img := w.config.BoardStyle.Image
		boardImageSize := size.FromMinPt(img.Bounds().Max)
		boardScaleFactor := w.curBoardSize.F32Pt.Div(boardImageSize.Float)
		util.DrawImage(cache, img, image.Point{}, boardScaleFactor)
		w.boardDrawingOp = boardMacro.Stop()

		w.squareSize = size.FromFloat(w.curBoardSize.Float / 8)
		w.halfPieceSize = size.FromMinF32Pt(w.squareSize.F32Pt.Div(2))
		w.hintSize = size.FromMinF32Pt(w.squareSize.F32Pt.Div(3))
		w.hintCenter = w.hintSize.F32Pt.Div(2)
	}

	w.boardDrawingOp.Add(gtx.Ops)

	defer clip.Rect(image.Rectangle{Max: w.curBoardSize.Pt}).Push(gtx.Ops).Pop()
	pointer.CursorPointer.Add(gtx.Ops)
	event.Op(gtx.Ops, w)

	w.drawPieces(gtx)

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: w,
			Kinds:  pointer.Move | pointer.Press | pointer.Drag,
		})
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			hoveredSquare := NewSquare(e.Position, w.squareSize.Float).ToChess()
			if e.Buttons.Contain(pointer.ButtonPrimary) {
				w.processLeftClick(gtx, e, hoveredSquare)
			} else if e.Buttons.Contain(pointer.ButtonSecondary) {
				w.processRightClick(gtx, e, hoveredSquare)
			}
		}
	}

	return layout.Dimensions{Size: w.curBoardSize.Pt}
}

func (w *Widget) SetGame(game *chess.Game) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.game = game
}

func (w *Widget) resized() bool {
	return w.curBoardSize != w.prevBoardSize
}

func (w *Widget) drawPieces(gtx layout.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.game == nil {
		return
	}

	curPosition := w.game.Position()
	defer func() {
		w.prevPosition = curPosition
	}()

	if w.config.ShowLastMove {
		lastMove := w.getLastMove()
		if lastMove != nil {
			w.markSquare(gtx, lastMove.S1(), w.config.LastMoveColor)
			w.markSquare(gtx, lastMove.S2(), w.config.LastMoveColor)
		}
	}

	pieceImageSize := size.FromMinPt(w.config.PiecesStyle.ImageFor(chess.WhitePawn).Bounds().Max)
	pieceScaleFactor := w.squareSize.F32Pt.Div(pieceImageSize.Float)

	// todo: add flip support

	if w.resized() || w.prevPosition == nil || curPosition.Hash() != w.prevPosition.Hash() {
		clear(w.squareDrawingOps)
		var wg sync.WaitGroup
		for square := chess.A1; square <= chess.H8; square++ {
			coords := SquareToPosition(square, w.squareSize.Float).Round()
			w.squareOriginCoordinates[square] = coords
			if square != w.selectedSquare {
				if piece := curPosition.Board().Piece(square); piece != chess.NoPiece {
					wg.Add(1)
					go func(square chess.Square, piece chess.Piece) {
						defer wg.Done()
						cache := new(op.Ops)
						squareMacro := op.Record(cache)
						img := w.config.PiecesStyle.ImageFor(piece)
						util.DrawImage(cache, img, coords, pieceScaleFactor)
						ops := squareMacro.Stop()
						w.squareDrawingOps[square] = &ops
					}(square, piece)
				}
			}
		}
		wg.Wait()
	}

	for _, squareDrawingOp := range w.squareDrawingOps {
		if squareDrawingOp != nil {
			squareDrawingOp.Add(gtx.Ops)
		}
	}

	// fixme: draw hints before pieces
	if w.selectedSquare != chess.NoSquare {
		w.markSquare(gtx, w.selectedSquare, GrayColor)

		img := w.config.PiecesStyle.ImageFor(w.selectedPiece)
		util.DrawImage(gtx.Ops, img, w.draggingPos.Round(), pieceScaleFactor)

		if w.config.ShowLegalMoves {
			for _, move := range curPosition.ValidMoves() {
				if move.S1() == w.selectedSquare {
					position := w.squareOriginCoordinates[move.S2()]
					if curPosition.Board().Piece(move.S2()) == chess.NoPiece {
						position = position.Add(w.halfPieceSize.Pt).Sub(w.hintCenter.Round())
						util.DrawEllipse(gtx.Ops, util.Rect(position, w.hintSize.Pt), w.config.HintColor)
					} else {
						rect := util.Rect(position, w.squareSize.Pt)
						util.DrawRectangle(gtx.Ops, rect, w.squareSize.Float/10, w.config.HintColor)
					}
				}
			}
		}
	}

	clear(w.pieceEventTargets)
	for square := range curPosition.Board().SquareMap() {
		coords := w.squareOriginCoordinates[square]
		pieceClip := clip.Rect(util.Rect(coords, w.squareSize.Pt)).Push(gtx.Ops)
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
			hoveredSquare := NewSquare(e.Position, w.squareSize.Float).ToChess()
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
		w.draggingPos = e.Position.Add(w.halfPointerSize.F32Pt).Sub(w.halfPieceSize.F32Pt)

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
			w.draggingPos = e.Position.Add(w.halfPointerSize.F32Pt).Sub(w.halfPieceSize.F32Pt)
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
	w.draggingPos = SquareToPosition(w.selectedSquare, w.squareSize.Float)
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
	util.DrawPane(gtx.Ops, util.Rect(origin, w.squareSize.Pt), color)
}
