package chessboard

import (
	"fmt"
	"image"
	"image/color"
	"slices"
	"sync"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Widget struct {
	config Config

	curBoardSize    union.Size
	prevBoardSize   union.Size
	squareSize      union.Size
	halfPointerSize union.Size
	halfPieceSize   union.Size
	hintSize        union.Size

	hintCenter union.Point

	buttonPressed pointer.Buttons
	modifiersUsed key.Modifiers

	annoType    AnnoType
	drawingAnno Annotation
	annotations []*Annotation

	squareOrigins []union.Point

	pieceEventTargets []event.Filter

	boardDrawingOp   op.CallOp
	hintDrawingOp    op.CallOp
	squareDrawingOps []*op.CallOp

	dragID         pointer.ID
	draggingPos    union.Point
	selectedSquare chess.Square
	selectedPiece  chess.Piece

	game         *chess.Game
	prevPosition *chess.Position

	mu sync.Mutex
}

func NewWidget(config Config) *Widget {
	w := Widget{
		game:              chess.NewGame(chess.UseNotation(chess.UCINotation{})),
		config:            config,
		halfPointerSize:   union.SizeFromInt(16 / 2), // assume for now
		squareOrigins:     make([]union.Point, 64),
		pieceEventTargets: make([]event.Filter, 64),
		squareDrawingOps:  make([]*op.CallOp, 64),
		selectedSquare:    chess.NoSquare,
		selectedPiece:     chess.NoPiece,
		annoType:          CircleAnno,
	}

	w.curBoardSize = union.SizeFromMinPt(config.BoardImage.Bounds().Max)
	w.prevBoardSize = w.curBoardSize
	w.squareSize = union.SizeFromFloat(w.curBoardSize.Float / 8)
	w.halfPieceSize = union.SizeFromMinF32(w.squareSize.F32.Div(2))
	w.hintSize = union.SizeFromMinF32(w.squareSize.F32.Div(3))

	w.hintCenter = union.PointFromF32(w.hintSize.F32.Div(2))
	w.draggingPos = union.PointFromF32(w.draggingPos.F32)

	for square := chess.A1; square <= chess.H8; square++ {
		w.squareOrigins[square] = SquareToPoint(square, w.squareSize.Float)
	}

	return &w
}

func (w *Widget) Layout(gtx layout.Context) layout.Dimensions {
	w.mu.Lock()
	defer w.mu.Unlock()

	// todo: add flip support

	curPosition := w.game.Position()
	w.curBoardSize = union.SizeFromMinPt(gtx.Constraints.Max)
	resizeFactor := w.curBoardSize.Float / w.prevBoardSize.Float

	defer func() {
		w.prevBoardSize = w.curBoardSize
		w.prevPosition = curPosition
	}()

	if w.resized() {
		cache := new(op.Ops)
		boardMacro := op.Record(cache)
		factor := w.curBoardSize.F32.Div(w.config.BoardImageSize.Float)
		util.DrawImage(cache, w.config.BoardImage, image.Point{}, factor)
		w.boardDrawingOp = boardMacro.Stop()

		w.squareSize.Scale(resizeFactor)
		w.halfPieceSize.Scale(resizeFactor)
		w.hintSize.Scale(resizeFactor)
		w.hintCenter.Scale(resizeFactor)
		w.draggingPos.Scale(resizeFactor)

		for square := chess.A1; square <= chess.H8; square++ {
			w.squareOrigins[square].Scale(resizeFactor)
		}
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

	if w.selectedSquare != chess.NoSquare && w.selectedPiece.Color() == curPosition.Turn() {
		w.markSquare(gtx, w.selectedSquare, util.GrayColor)
		if w.config.ShowHints {
			for _, move := range curPosition.ValidMoves() {
				if move.S1() == w.selectedSquare {
					position := w.squareOrigins[move.S2()]
					if curPosition.Board().Piece(move.S2()) == chess.NoPiece {
						origin := position.F32.Add(w.halfPieceSize.F32).Sub(w.hintCenter.F32).Round()
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
		anno.Scale(resizeFactor)
		anno.Draw(gtx, w.squareOrigins, w.squareSize, w.resized())
	}
	w.drawingAnno.Scale(resizeFactor)
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

func (w *Widget) resized() bool {
	return w.curBoardSize != w.prevBoardSize
}

func (w *Widget) positionChanged() bool {
	return w.prevPosition != nil && w.prevPosition.Hash() != w.game.Position().Hash()
}

func (w *Widget) drawPieces(gtx layout.Context) {
	if w.game == nil {
		return
	}

	curPosition := w.game.Position()

	if w.resized() || w.positionChanged() {
		clear(w.squareDrawingOps)
		var wg sync.WaitGroup
		for square := chess.A1; square <= chess.H8; square++ {
			origin := w.squareOrigins[square]
			if piece := curPosition.Board().Piece(square); piece != chess.NoPiece {
				wg.Add(1)
				go func(square chess.Square, piece chess.Piece) {
					defer wg.Done()
					cache := new(op.Ops)
					squareMacro := op.Record(cache)
					factor := w.squareSize.F32.Div(w.config.PieceImageSizes[piece].Float)
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
	hoveredSquare := NewSquare(e.Position, w.squareSize.Float).ToChess()
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
	hoveredSquare := NewSquare(e.Position, w.squareSize.Float).ToChess()
	defer gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 30)})

	switch e.Kind {
	case pointer.Press:
		if hoveredSquare != chess.NoSquare {
			w.drawingAnno = Annotation{
				Type:  CrossAnno,
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
			if w.drawingAnno.Type == NoAnno {
				return annotation.Start == hoveredSquare
			} else {
				if w.drawingAnno.Type == ArrowAnno {
					return annotation.Type == ArrowAnno &&
						annotation.Start == w.drawingAnno.Start && annotation.End == w.drawingAnno.End
				} else {
					return annotation.Start == w.drawingAnno.Start
				}
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
		w.draggingPos = union.PointFromF32(e.Position.Add(w.halfPointerSize.F32).Sub(w.halfPieceSize.F32))
		gtx.Execute(pointer.GrabCmd{
			Tag: w.selectedSquare,
			ID:  w.dragID,
		})
	}
}

func (w *Widget) processSecondaryButtonDragging(gtx layout.Context, e pointer.Event) {
	if w.drawingAnno.Type != NoAnno {
		hoveredSquare := NewSquare(e.Position, w.squareSize.Float).ToChess()
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
		w.draggingPos = union.PointFromF32(e.Position.Add(w.halfPointerSize.F32).Sub(w.halfPieceSize.F32))
		gtx.Execute(pointer.GrabCmd{
			Tag: w.selectedSquare,
			ID:  w.dragID,
		})
	}
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
