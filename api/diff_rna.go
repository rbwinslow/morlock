package api

import (
	"fmt"
	"github.com/waigani/diffparser"
)

type diffRNA struct {
	origLines, newLines                []*diffparser.DiffLine
	positionCursor, lastPos            int
	origCursor, newCursor              int
	resultCursorHunk, resultCursorLine int
	result                             *Timelapse
}

func newDiffRNA(origLines, newLines []*diffparser.DiffLine, result *Timelapse) (*diffRNA, error) {
	obj := diffRNA{
		origLines:      origLines,
		newLines:       newLines,
		result:         result,
		positionCursor: 1,
	}
	if result == nil {
		return nil, fmt.Errorf("New diffRNA passed nil Timelapse pointer")
	}

	if origLines[len(origLines)-1].Position > newLines[len(newLines)-1].Position {
		obj.lastPos = origLines[len(origLines)-1].Position
	} else {
		obj.lastPos = newLines[len(newLines)-1].Position
	}

	obj.resultCursorHunk, obj.resultCursorLine = mustSeekAddedLine(result, newLines[0].Number)

	return &obj, nil
}

func (dn *diffRNA) advance() error {
	if !dn.eof() {
		dn.positionCursor++
		if len(dn.origLines) > (dn.origCursor+1) && dn.origLines[dn.origCursor].Position <= dn.positionCursor-1 {
			dn.origCursor++
		}
		if len(dn.newLines) > (dn.newCursor+1) && dn.newLines[dn.newCursor].Position == dn.positionCursor-1 {
			dn.newCursor++
			if dn.newLines[dn.newCursor].Mode == diffparser.UNCHANGED {
				if len((*dn.result)[dn.resultCursorHunk].Lines) > dn.resultCursorLine+1 {
					dn.resultCursorLine++
				} else if len(*dn.result) > dn.resultCursorHunk+1 {
					dn.resultCursorHunk++
					dn.resultCursorLine = 0
				} else {
					return fmt.Errorf("Difference analysis error; unchanged lines overran timelapse at diff line \"%s\"", dn.newLines[dn.newCursor])
				}
			}
		}
	}

	return nil
}

func (dn *diffRNA) advanceOrigWhile(mode diffparser.DiffLineMode) (int, error) {
	max := len(dn.origLines)
	for dn.origCursor < max && dn.origLines[dn.origCursor+1].Mode == mode && dn.origLines[dn.origCursor+1].Position == dn.origLines[dn.origCursor].Position+1 {
		if err := dn.advance(); err != nil {
			return dn.origCursor, err
		}
	}
	return dn.origCursor, nil
}

func (dn *diffRNA) dump() string {
	origLine := dn.origLines[dn.origCursor].Content
	newLine := dn.newLines[dn.newCursor].Content

	truncOrig := 10
	if truncOrig > len(origLine) {
		truncOrig = len(origLine)
	}
	truncNew := 10
	if truncNew > len(newLine) {
		truncNew = len(newLine)
	}

	format := "At position %d: orig line %d of %d (%s), new line %d of %d (%s), result hunk %d of %d, line %d of %d (%s); EOF %t"
	hunkLines := (*dn.result)[dn.resultCursorHunk].Lines
	lineAtCursor := hunkLines[dn.resultCursorLine]
	return fmt.Sprintf(format, dn.positionCursor, dn.origCursor, len(dn.origLines), origLine[:truncOrig], dn.newCursor, len(dn.newLines), newLine[:truncNew], dn.resultCursorHunk, len(*dn.result), dn.resultCursorLine, len(hunkLines), lineAtCursor, dn.eof())
}

func (dn *diffRNA) eof() bool {
	return dn.positionCursor >= dn.lastPos
}

func (dn *diffRNA) hasOrig() bool {
	return dn.origLines[dn.origCursor].Position == dn.positionCursor
}

func (dn *diffRNA) hasNew() bool {
	return dn.newLines[dn.newCursor].Position == dn.positionCursor
}

func (dn *diffRNA) orig() *diffparser.DiffLine {
	return dn.origLines[dn.origCursor]
}

func (dn *diffRNA) spliceTimelapse(deleteHowMany int, newHunkCursor int, what ...TimelapseHunk) {
	*dn.result = append(append((*dn.result)[:dn.resultCursorHunk], noNilHunks(what...)...), (*dn.result)[dn.resultCursorHunk + deleteHowMany:]...)
	dn.resultCursorHunk = newHunkCursor
	dn.resultCursorLine = 0
}

func (dn *diffRNA) transcribe() error {
	for !dn.eof() {
		if dn.hasOrig() && dn.orig().Mode == diffparser.REMOVED {
			if dn.hasNew() {
				return fmt.Errorf("Found deleted line with same position in diff hunk as new line!")
			}
			start := dn.origCursor
			end, err := dn.advanceOrigWhile(diffparser.REMOVED)
			if err != nil {
				return err
			}
			lines := make([]string, (end-start)+1)
			for i := 0; i < (end-start)+1; i++ {
				lines[i] = dn.origLines[start+i].Content
			}
			delhunk := TimelapseHunk{
				DELETED,
				lines,
			}
			forehunk, afthunk := splitHunk((*dn.result)[dn.resultCursorHunk], dn.resultCursorLine)
			dn.spliceTimelapse(1, dn.resultCursorHunk + 2, forehunk, delhunk, afthunk)
		}
		if err := dn.advance(); err != nil {
			return err
		}
	}

	return nil
}

func mustSeekAddedLine(tl *Timelapse, lineNumber int) (hunkIndex, lineIndex int) {
	for lineno := lineNumber; hunkIndex < len(*tl); hunkIndex++ {
		thisHunk := &(*tl)[hunkIndex]
		if (*thisHunk).Disposition == DELETED {
			continue
		}
		if len((*thisHunk).Lines) < lineno {
			lineno -= len((*thisHunk).Lines)
		} else {
			lineIndex = lineno - 1
			return
		}
	}
	panic(fmt.Sprintf("Seeking added line %d went past end of Timelapse with %s hunks", lineNumber, len(*tl)))
}

func noNilHunks(hunks ...TimelapseHunk) []TimelapseHunk {
	var result []TimelapseHunk
	for _, h := range hunks {
		if h.Lines != nil {
			result = append(result, h)
		}
	}
	return result
}

func splitHunk(h TimelapseHunk, line int) (before, after TimelapseHunk) {
	if line > 0 {
		before = TimelapseHunk{h.Disposition, h.Lines[:line]}
	}
	if line < len(h.Lines) {
		after = TimelapseHunk{h.Disposition, h.Lines[line:]}
	}
	return
}
