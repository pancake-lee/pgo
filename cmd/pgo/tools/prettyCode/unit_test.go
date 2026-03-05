package prettyCode

import (
	"testing"
)

func TestCommentLine(t *testing.T) {
	lineMap := map[string]bool{
		`	// --------------------------`:    true,
		`		// --------------------------`:   true,
		`// --------------------------`:     true,
		`	/* --------------------------`:    true,
		`	--------------------------*/`:     true,
		`	/*--------------------------*/`:   true,
		`	/*`:                               false,
		`	// abc--------------------------`: false,
		`	// --------------------------abc`: false,
	}
	for line, expected := range lineMap {
		newLine, modified := processLine(line)
		if modified != expected {
			t.Errorf("Expected modified[%v] for line[%s]", expected, line)
		}

		tmpLine, modified := processLine(newLine)
		if modified && tmpLine != newLine {
			t.Errorf("Expected no modification for line[%s] after processing, got [%v]", newLine, tmpLine)
		}
	}
}
