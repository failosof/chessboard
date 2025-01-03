package chessboard

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/widget/material"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

// todo: add inside coordinates
// todo: add each square coordinates
// todo: add pawn promotion
// todo: animations

type Widget struct {
	th *material.Theme

	config Config

	redraw bool

	curBoardSize  union.Size
	prevBoardSize union.Size
	squareSize    union.Size
	hintSize      union.Size
	pointerSize   union.Size

	buttonPressed pointer.Buttons
	modifiersUsed key.Modifiers

	annoType    AnnoType
	drawingAnno Annotation
	annotations []*Annotation

	squareOrigins []union.Point

	pieceEventTargets []event.Filter

	coordsDrawingOp  op.CallOp
	boardDrawingOp   op.CallOp
	hintDrawingOp    op.CallOp
	squareDrawingOps []*op.CallOp

	dragID         pointer.ID
	draggingPos    union.Point
	selectedSquare chess.Square
	selectedPiece  chess.Piece

	flipped      bool
	game         *chess.Game
	curPosition  *chess.Position
	prevPosition *chess.Position

	mu sync.Mutex
}

func NewWidget(th *material.Theme, config Config) *Widget {
	w := Widget{
		th:                th,
		config:            config,
		pointerSize:       union.SizeFromInt(16), // assume for now
		squareOrigins:     make([]union.Point, 64),
		pieceEventTargets: make([]event.Filter, 64),
		squareDrawingOps:  make([]*op.CallOp, 64),
		selectedSquare:    chess.NoSquare,
		selectedPiece:     chess.NoPiece,
		annoType:          CircleAnno,
		game:              chess.NewGame(chess.UseNotation(chess.UCINotation{})),
	}

	return &w
}

func (w *Widget) Layout(gtx layout.Context) layout.Dimensions {
	return CoordinatesStyle{
		Type:     w.config.Coordinates,
		Theme:    w.th,
		FontSize: 16,
		Flipped:  w.flipped,
		Board:    w.layout,
	}.Layout(gtx)
}

func (w *Widget) layout(gtx layout.Context) layout.Dimensions {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.curBoardSize = union.SizeFromMinPt(gtx.Constraints.Max)
	w.curPosition = w.game.Position()
	w.redraw = w.redraw || !w.curBoardSize.Eq(w.prevBoardSize) || w.positionChanged()
	defer func() {
		w.redraw = false
		w.prevBoardSize = w.curBoardSize
		w.prevPosition = w.curPosition
	}()

	if w.redraw {
		w.squareSize = union.SizeFromFloat(w.curBoardSize.Float / 8)
		w.hintSize = union.SizeFromMinF32(w.squareSize.F32.Div(3))
		w.draggingPos = union.PointFromF32(w.draggingPos.F32)

		for square := chess.A1; square <= chess.H8; square++ {
			w.squareOrigins[square] = union.PointFromF32(util.SquareToPoint(square, w.squareSize.Float, w.flipped))
		}

		cache := new(op.Ops)
		boardMacro := op.Record(cache)
		factor := w.curBoardSize.F32.Div(w.config.BoardImageSize.Float)
		util.DrawImage(cache, w.config.BoardImage, image.Point{}, factor)
		w.boardDrawingOp = boardMacro.Stop()
	}

	w.boardDrawingOp.Add(gtx.Ops)

	defer clip.Rect(image.Rectangle{Max: w.curBoardSize.Pt}).Push(gtx.Ops).Pop()
	event.Op(gtx.Ops, w)

	if w.config.ShowLastMove {
		lastMove := w.getLastMove()
		if lastMove != nil {
			w.markSquare(gtx, lastMove.S1(), w.config.Color.LastMove)
			w.markSquare(gtx, lastMove.S2(), w.config.Color.LastMove)
		}
	}

	if w.selectedSquare != chess.NoSquare && w.selectedPiece.Color() == w.curPosition.Turn() {
		w.markSquare(gtx, w.selectedSquare, util.GrayColor)
		if w.config.ShowHints {
			for _, move := range w.curPosition.ValidMoves() {
				if move.S1() == w.selectedSquare {
					position := w.squareOrigins[move.S2()]
					if w.curPosition.Board().Piece(move.S2()) == chess.NoPiece {
						origin := position.F32.Add(w.squareSize.Half.F32).Sub(w.hintSize.Half.F32).Round()
						util.DrawEllipse(gtx.Ops, util.Rect(origin, w.hintSize.Pt), w.config.Color.Hint)
					} else {
						rect := util.Rect(position.Pt, w.squareSize.Pt)
						util.DrawRectangle(gtx.Ops, rect, w.squareSize.Float/5, w.config.Color.Hint)
					}
				}
			}
		}
	}

	w.drawPieces(gtx)

	for _, anno := range w.annotations {
		anno.Width = union.SizeFromFloat(w.squareSize.Float / 7)
		anno.Draw(gtx, w.squareOrigins, w.squareSize, w.redraw)
	}
	w.drawingAnno.Width = union.SizeFromFloat(w.squareSize.Float / 9)
	w.drawingAnno.Draw(gtx, w.squareOrigins, w.squareSize, w.drawingAnno.Type != NoAnno)

	for {
		ev, ok := gtx.Event(pointer.Filter{
			Target: w,
			Kinds:  pointer.Move | pointer.Press | pointer.Release | pointer.Drag,
		})
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Move:
				pointer.CursorPointer.Add(gtx.Ops)
			case pointer.Drag:
				if w.buttonPressed == pointer.ButtonSecondary {
					w.processSecondaryButtonDragging(gtx, e)
				}
			case pointer.Press:
				w.buttonPressed = e.Buttons
				w.modifiersUsed = e.Modifiers
				fallthrough
			default:
				if w.buttonPressed == pointer.ButtonPrimary {
					w.processPrimaryButtonClick(gtx, e)
				} else if w.buttonPressed == pointer.ButtonSecondary {
					w.processSecondaryButtonClick(gtx, e)
				}
			}
		}
	}

	for {
		ev, ok := gtx.Event(w.pieceEventTargets...)
		if !ok {
			break
		}

		if e, ok := ev.(pointer.Event); ok {
			switch e.Kind {
			case pointer.Move:
				pointer.CursorGrab.Add(gtx.Ops)
			case pointer.Drag:
				if w.buttonPressed == pointer.ButtonPrimary {
					w.processPrimaryButtonDragging(gtx, e)
				}
			case pointer.Release:
				w.processPrimaryButtonClick(gtx, e)
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

func (w *Widget) Flip(gtx layout.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flipped = !w.flipped
	w.redraw = true
	gtx.Execute(op.InvalidateCmd{})
}

func (w *Widget) positionChanged() bool {
	return w.prevPosition != nil && w.prevPosition.Hash() != w.curPosition.Hash()
}

func (w *Widget) drawPieces(gtx layout.Context) {
	if w.game == nil {
		return
	}

	if w.redraw {
		clear(w.squareDrawingOps)
		var wg sync.WaitGroup
		for square := chess.A1; square <= chess.H8; square++ {
			origin := w.squareOrigins[square]
			if piece := w.curPosition.Board().Piece(square); piece != chess.NoPiece {
				wg.Add(1)
				go func(square chess.Square, piece chess.Piece) {
					defer wg.Done()
					factor := w.squareSize.F32.Div(w.config.PieceImageSizes[piece].Float)
					cache := new(op.Ops)
					squareMacro := op.Record(cache)
					util.DrawImage(cache, w.config.PieceImages[piece], origin.Pt, factor)
					ops := squareMacro.Stop()
					w.squareDrawingOps[square] = &ops
				}(square, piece)
			}
		}
		wg.Wait()
	}

	clear(w.pieceEventTargets)
	for square := chess.A1; square <= chess.H8; square++ {
		squareDrawingOp := w.squareDrawingOps[square]
		if squareDrawingOp != nil {
			origin := w.squareOrigins[square]
			pieceClip := clip.Rect(util.Rect(origin.Pt, w.squareSize.Pt)).Push(gtx.Ops)
			event.Op(gtx.Ops, square)
			pieceClip.Pop()
			w.pieceEventTargets = append(w.pieceEventTargets, pointer.Filter{
				Target: square,
				Kinds:  pointer.Move | pointer.Drag | pointer.Release,
			})

			if square != w.selectedSquare {
				squareDrawingOp.Add(gtx.Ops)
			}
		}
	}

	if w.selectedSquare != chess.NoSquare {
		img := w.config.PieceImages[w.selectedPiece]
		factor := w.squareSize.F32.Div(w.config.PieceImageSizes[w.selectedPiece].Float)
		util.DrawImage(gtx.Ops, img, w.draggingPos.Pt, factor)
	}
}

func (w *Widget) processPrimaryButtonClick(gtx layout.Context, e pointer.Event) {
	hoveredSquare := util.PointToSquare(e.Position, w.squareSize.Float, w.flipped)
	if hoveredSquare == chess.NoSquare {
		return
	}
	hoveredPiece := w.game.Position().Board().Piece(hoveredSquare)

	switch e.Kind {
	case pointer.Press:
		clear(w.annotations)
		w.annotations = nil
		w.drawingAnno.Type = NoAnno

		if w.selectedPiece == chess.NoPiece || w.selectedPiece.Color() == hoveredPiece.Color() {
			if hoveredPiece != chess.NoPiece {
				w.selectPiece(gtx, e, hoveredPiece, hoveredSquare)
				return
			}
		}

		fallthrough
	case pointer.Release:
		if w.selectedSquare == hoveredSquare {
			w.putSelectedPieceBack(gtx)
			return
		}

		if w.selectedSquare != chess.NoSquare {
			if err := w.moveSelectedPieceTo(hoveredSquare); err != nil {
				w.putSelectedPieceBack(gtx)
				if hoveredPiece != chess.NoPiece {
					w.selectPiece(gtx, e, hoveredPiece, hoveredSquare)
					return
				}
			}
		}

		fallthrough
	default:
		w.unselectPiece(gtx)
		w.buttonPressed = 0
		w.modifiersUsed = 0
	}
}

func (w *Widget) processSecondaryButtonClick(gtx layout.Context, e pointer.Event) {
	hoveredSquare := util.PointToSquare(e.Position, w.squareSize.Float, w.flipped)
	defer gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 30)})

	switch e.Kind {
	case pointer.Press:
		if hoveredSquare != chess.NoSquare {
			w.drawingAnno = Annotation{
				Type:  w.annoType,
				Start: hoveredSquare,
				Color: util.Transparentize(w.selectAnnotationColor(), 0.5),
				Width: union.SizeFromFloat(w.squareSize.Float / 9),
			}
			w.dragID = e.PointerID
		}
	case pointer.Release:
		w.drawingAnno.Width = union.SizeFromFloat(w.squareSize.Float / 7)
		w.drawingAnno.Color = w.selectAnnotationColor()
		if hoveredSquare != chess.NoSquare {
			w.drawingAnno.End = hoveredSquare
		}

		i := slices.IndexFunc(w.annotations, func(annotation *Annotation) bool {
			if w.drawingAnno.Type != annotation.Type {
				return false
			}
			if w.drawingAnno.Type == ArrowAnno {
				return annotation.Start == w.drawingAnno.Start && annotation.End == w.drawingAnno.End
			} else {
				return annotation.Start == w.drawingAnno.Start
			}
		})

		anno := w.drawingAnno.Copy()
		if i > -1 {
			if w.annotations[i].Equal(&w.drawingAnno) {
				w.annotations = slices.Delete(w.annotations, i, i+1)
			} else {
				w.annotations[i] = &anno
			}
		} else {
			w.annotations = append(w.annotations, &anno)
		}

		w.drawingAnno = Annotation{}
		w.dragID = 0
		w.buttonPressed = 0
		w.modifiersUsed = 0
	}
}

func (w *Widget) processPrimaryButtonDragging(gtx layout.Context, e pointer.Event) {
	if w.dragID == e.PointerID && w.selectedSquare != chess.NoSquare {
		pointer.CursorGrabbing.Add(gtx.Ops)
		w.dragTo(gtx, e.Position)
	}
}

func (w *Widget) processSecondaryButtonDragging(gtx layout.Context, e pointer.Event) {
	if w.drawingAnno.Type != NoAnno {
		hoveredSquare := util.PointToSquare(e.Position, w.squareSize.Float, w.flipped)
		if hoveredSquare != chess.NoSquare {
			if w.dragID == e.PointerID {
				w.drawingAnno.End = hoveredSquare
				if w.drawingAnno.Start == w.drawingAnno.End {
					w.drawingAnno.Type = w.annoType
				} else {
					w.drawingAnno.Type = ArrowAnno
				}
				gtx.Execute(op.InvalidateCmd{})
			}
		}
	}
}

func (w *Widget) selectPiece(gtx layout.Context, e pointer.Event, piece chess.Piece, square chess.Square) {
	if piece != chess.NoPiece && square != chess.NoSquare {
		pointer.CursorGrabbing.Add(gtx.Ops)
		w.dragID = e.PointerID
		w.selectedPiece = piece
		w.selectedSquare = square
		w.dragTo(gtx, e.Position)
	}
}

func (w *Widget) dragTo(gtx layout.Context, pos f32.Point) {
	w.draggingPos = union.PointFromF32(pos.Add(w.pointerSize.Half.F32).Sub(w.squareSize.Half.F32))
	gtx.Execute(pointer.GrabCmd{
		Tag: w.selectedSquare,
		ID:  w.dragID,
	})
}

func (w *Widget) putSelectedPieceBack(gtx layout.Context) {
	if w.selectedSquare != chess.NoSquare {
		w.draggingPos = w.squareOrigins[w.selectedSquare]
	}

	pointer.CursorPointer.Add(gtx.Ops)
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 25)})
}

func (w *Widget) unselectPiece(gtx layout.Context) {
	if w.selectedSquare != chess.NoSquare {
		w.draggingPos = w.squareOrigins[w.selectedSquare]
	}

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

func (w *Widget) moveSelectedPieceTo(to chess.Square) error {
	if w.selectedSquare == chess.NoSquare {
		return fmt.Errorf("no square selected")
	} else {
		move := w.selectedSquare.String() + to.String()
		return w.game.MoveStr(move)
	}
}

func (w *Widget) markSquare(gtx layout.Context, square chess.Square, color color.NRGBA) {
	if square != chess.NoSquare {
		origin := w.squareOrigins[square]
		util.DrawPane(gtx.Ops, util.Rect(origin.Pt, w.squareSize.Pt), color)
	}
}

func (w *Widget) selectAnnotationColor() color.NRGBA {
	if w.modifiersUsed == 0 {
		return w.config.Color.Primary
	}

	if w.modifiersUsed&key.ModAlt == key.ModAlt {
		return w.config.Color.Warning
	} else if w.modifiersUsed&key.ModShift == key.ModShift {
		return w.config.Color.Info
	} else if w.modifiersUsed&key.ModCtrl == key.ModCtrl {
		return w.config.Color.Danger
	} else {
		return w.config.Color.Primary
	}
}
