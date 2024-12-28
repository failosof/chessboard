package chessboard

import (
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"sync"
	"time"

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/failosof/chessboard/config"
	"github.com/failosof/chessboard/union"
	"github.com/failosof/chessboard/util"
	"github.com/notnil/chess"
)

type Widget struct {
	config config.Chessboard

	curBoardSize    union.Size
	prevBoardSize   union.Size
	squareSize      union.Size
	halfPointerSize union.Size
	halfPieceSize   union.Size
	hintSize        union.Size

	hintCenter union.Point

	annotations []*Annotation

	squareOriginCoordinates []union.Point

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

func NewWidget(config config.Chessboard) *Widget {
	w := Widget{
		game:                    chess.NewGame(chess.UseNotation(chess.UCINotation{})),
		config:                  config,
		halfPointerSize:         union.SizeFromInt(16 / 2), // assume for now
		squareOriginCoordinates: make([]union.Point, 64),
		pieceEventTargets:       make([]event.Filter, 64),
		squareDrawingOps:        make([]*op.CallOp, 64),
		selectedSquare:          chess.NoSquare,
		selectedPiece:           chess.NoPiece,
	}

	img := config.BoardStyle.Image

	w.curBoardSize = union.SizeFromMinPt(img.Bounds().Max)
	w.prevBoardSize = w.curBoardSize
	w.squareSize = union.SizeFromFloat(w.curBoardSize.Float / 8)
	w.halfPieceSize = union.SizeFromMinF32(w.squareSize.F32.Div(2))
	w.hintSize = union.SizeFromMinF32(w.squareSize.F32.Div(3))

	w.hintCenter = union.PointFromF32(w.hintSize.F32.Div(2))
	w.draggingPos = union.PointFromF32(w.draggingPos.F32)

	for square := chess.A1; square <= chess.H8; square++ {
		w.squareOriginCoordinates[square] = SquareToPoint(square, w.squareSize.Float)
	}

	return &w
}

func (w *Widget) Layout(gtx layout.Context) layout.Dimensions {
	w.mu.Lock()
	defer w.mu.Unlock()

	// todo: add flip support

	w.curBoardSize = union.SizeFromMinPt(gtx.Constraints.Max)
	resizeFactor := w.curBoardSize.Float / w.prevBoardSize.Float

	if w.resized() {
		defer func() { w.prevBoardSize = w.curBoardSize }()

		cache := new(op.Ops)
		boardMacro := op.Record(cache)
		img := w.config.BoardStyle.Image
		boardImageSize := union.SizeFromMinPt(img.Bounds().Max)
		boardScaleFactor := w.curBoardSize.F32.Div(boardImageSize.Float)
		util.DrawImage(cache, img, image.Point{}, boardScaleFactor)
		w.boardDrawingOp = boardMacro.Stop()

		w.squareSize.Scale(resizeFactor)
		w.halfPieceSize.Scale(resizeFactor)
		w.hintSize.Scale(resizeFactor)
		w.hintCenter.Scale(resizeFactor)
		w.draggingPos.Scale(resizeFactor)

		for square := chess.A1; square <= chess.H8; square++ {
			w.squareOriginCoordinates[square].Scale(resizeFactor)
		}
	}

	w.boardDrawingOp.Add(gtx.Ops)

	defer clip.Rect(image.Rectangle{Max: w.curBoardSize.Pt}).Push(gtx.Ops).Pop()
	pointer.CursorPointer.Add(gtx.Ops)
	event.Op(gtx.Ops, w)

	if w.config.ShowLastMove {
		lastMove := w.getLastMove()
		if lastMove != nil {
			w.markSquare(gtx, lastMove.S1(), w.config.LastMoveColor)
			w.markSquare(gtx, lastMove.S2(), w.config.LastMoveColor)
		}
	}

	if w.selectedSquare != chess.NoSquare {
		curPosition := w.game.Position()
		w.markSquare(gtx, w.selectedSquare, GrayColor)
		if w.config.ShowLegalMoves {
			for _, move := range curPosition.ValidMoves() {
				if move.S1() == w.selectedSquare {
					position := w.squareOriginCoordinates[move.S2()]
					if curPosition.Board().Piece(move.S2()) == chess.NoPiece {
						origin := position.F32.Add(w.halfPieceSize.F32).Sub(w.hintCenter.F32).Round()
						util.DrawEllipse(gtx.Ops, util.Rect(origin, w.hintSize.Pt), w.config.HintColor)
					} else {
						rect := util.Rect(position.Pt, w.squareSize.Pt)
						util.DrawRectangle(gtx.Ops, rect, w.squareSize.Float/5, w.config.HintColor)
					}
				}
			}
		}
	}

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

	for _, anno := range w.annotations {
		anno.Scale(resizeFactor)
		anno.Draw(gtx, w.squareSize, w.resized())
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
	defer func() { w.prevPosition = curPosition }()

	pieceImageSize := union.SizeFromMinPt(w.config.PiecesStyle.ImageFor(chess.WhitePawn).Bounds().Max)
	pieceScaleFactor := w.squareSize.F32.Div(pieceImageSize.Float)

	if w.resized() || w.positionChanged() {
		clear(w.squareDrawingOps)
		var wg sync.WaitGroup
		for square := chess.A1; square <= chess.H8; square++ {
			origin := w.squareOriginCoordinates[square]
			if piece := curPosition.Board().Piece(square); piece != chess.NoPiece {
				wg.Add(1)
				go func(square chess.Square, piece chess.Piece) {
					defer wg.Done()
					cache := new(op.Ops)
					squareMacro := op.Record(cache)
					img := w.config.PiecesStyle.ImageFor(piece)
					util.DrawImage(cache, img, origin.Pt, pieceScaleFactor)
					ops := squareMacro.Stop()
					w.squareDrawingOps[square] = &ops
				}(square, piece)
			}
		}
		wg.Wait()
	}

	for square, squareDrawingOp := range w.squareDrawingOps {
		if chess.Square(square) != w.selectedSquare && squareDrawingOp != nil {
			squareDrawingOp.Add(gtx.Ops)
		}
	}

	if w.selectedSquare != chess.NoSquare {
		img := w.config.PiecesStyle.ImageFor(w.selectedPiece)
		util.DrawImage(gtx.Ops, img, w.draggingPos.Pt, pieceScaleFactor)
	}

	clear(w.pieceEventTargets)
	for square := range curPosition.Board().SquareMap() {
		origin := w.squareOriginCoordinates[square]
		pieceClip := clip.Rect(util.Rect(origin.Pt, w.squareSize.Pt)).Push(gtx.Ops)
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
	slog.Debug("marking", "square", hoveredSquare, "pos", e.Position)
	w.annotations = []*Annotation{
		//{
		//	Type:  CrossAnno,
		//	Start: SquareToPoint(hoveredSquare, w.squareSize.Float),
		//	Color: Transparentize(RedColor, 0.9),
		//	Width: union.SizeFromFloat(w.squareSize.Float / 7),
		//},
		{
			Type:  CircleAnno,
			Start: SquareToPoint(hoveredSquare, w.squareSize.Float),
			Color: Transparentize(BlueColor, 0.9),
			Width: union.SizeFromFloat(w.squareSize.Float / 7),
		},
		//{
		//	Type:  ArrowAnno,
		//	Start: SquareToPoint(hoveredSquare, w.squareSize.Float),
		//	End:   SquareToPoint(chess.A8, w.squareSize.Float),
		//	Color: Transparentize(GreenColor, 0.7),
		//	Width: union.SizeFromFloat(w.squareSize.Float / 5),
		//},
		//{
		//	Type:  ArrowAnno,
		//	Start: SquareToPoint(hoveredSquare, w.squareSize.Float),
		//	End:   SquareToPoint(chess.H8, w.squareSize.Float),
		//	Color: Transparentize(BlueColor, 0.7),
		//	Width: union.SizeFromFloat(w.squareSize.Float / 5),
		//},
		//{
		//	Type:  ArrowAnno,
		//	Start: SquareToPoint(hoveredSquare, w.squareSize.Float),
		//	End:   SquareToPoint(chess.A1, w.squareSize.Float),
		//	Color: Transparentize(RedColor, 0.7),
		//	Width: union.SizeFromFloat(w.squareSize.Float / 5),
		//},
		//{
		//	Type:  ArrowAnno,
		//	Start: SquareToPoint(hoveredSquare, w.squareSize.Float),
		//	End:   SquareToPoint(chess.H1, w.squareSize.Float),
		//	Color: Transparentize(YellowColor, 0.7),
		//	Width: union.SizeFromFloat(w.squareSize.Float / 5),
		//},
	}
}

func (w *Widget) processLeftClick(
	gtx layout.Context,
	e pointer.Event,
	hoveredSquare chess.Square,
) {
	curPosition := w.game.Position()
	hoveredPiece := curPosition.Board().Piece(hoveredSquare)

	clear(w.annotations)
	w.annotations = nil

	if hoveredPiece.Color() == curPosition.Turn() {
		pointer.CursorGrabbing.Add(gtx.Ops)

		w.dragID = e.PointerID
		w.selectedPiece = hoveredPiece
		w.selectedSquare = hoveredSquare
		w.draggingPos = union.PointFromF32(e.Position.Add(w.halfPointerSize.F32).Sub(w.halfPieceSize.F32))

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
			w.draggingPos = union.PointFromF32(e.Position.Add(w.halfPointerSize.F32).Sub(w.halfPieceSize.F32))
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
	w.draggingPos = SquareToPoint(w.selectedSquare, w.squareSize.Float)
	pointer.CursorPointer.Add(gtx.Ops)
	gtx.Execute(op.InvalidateCmd{At: gtx.Now.Add(time.Second / 25)})
}

func (w *Widget) unselectPiece(gtx layout.Context) {
	w.draggingPos = SquareToPoint(w.selectedSquare, w.squareSize.Float)
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
	util.DrawPane(gtx.Ops, util.Rect(origin.Pt, w.squareSize.Pt), color)
}
