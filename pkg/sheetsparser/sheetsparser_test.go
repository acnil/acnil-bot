package sheetsparser

import (
	"testing"
	"time"
)

type testDataHappy struct {
	Ignored string

	Col1  string    `col:"0"`
	Col2  string    `col:"1"`
	Col3  string    `col:"2"`
	Date4 time.Time `col:"3"`
}

func TestSheetParser_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	g := testDataHappy{}
	err := p.Unmarshal([]interface{}{"r1col1", "r1col2", "r1col3", "10/12/2022"}, &g)

	if err != nil {
		t.Error(err)
	}
	if g.Col1 != "r1col1" {
		t.Errorf("g.Col1 is %s but must be r1col1", g.Col1)
		t.FailNow()
	}
	if g.Col2 != "r1col2" {
		t.Errorf("g.Col3 is %s but must be r1col2", g.Col2)
		t.FailNow()
	}
	if g.Col3 != "r1col3" {
		t.Errorf("g.Col3 is %s but must be r1col3", g.Col3)
		t.FailNow()
	}
	if g.Date4 != time.Date(2022, 12, 10, 0, 0, 0, 0, time.UTC) {
		t.Errorf("g.Date4 is %s but must be the given timestamp", g.Date4)
		t.FailNow()
	}

}

func TestSheetParser_Marshal(t *testing.T) {
	p := &SheetParser{}
	g := testDataHappy{
		Col1:  "r1Col1",
		Col2:  "r1Col2",
		Col3:  "r1Col3",
		Date4: time.Date(2022, 12, 10, 0, 0, 0, 0, time.UTC),
	}

	out, err := p.Marshal(&g)
	if err != nil {
		t.Error(err)
	}

	if out[0].(string) != "r1Col1" {
		t.Errorf("field 0 is %s but must be r1Col1", out[0])
		t.FailNow()
	}
	if out[1].(string) != "r1Col2" {
		t.Errorf("field 1 is %s but must be r1Col1", out[1])
		t.FailNow()
	}
	if out[2].(string) != "r1Col3" {
		t.Errorf("field 2 is %s but must be r1Col1", out[2])
		t.FailNow()
	}
	if out[3].(string) != "10/12/2022" {
		t.Errorf("field 3 is %s but must be 10/12/2022", out[3])
		t.FailNow()
	}
}

type testDataGaps struct {
	Ignored string

	Col1 string `col:"0"`
	// Col2  string    `col:"1"`
	Col3  string    `col:"2"`
	Date4 time.Time `col:"3"`
}

func TestSheetParserGaps_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	g := testDataGaps{}
	err := p.Unmarshal([]interface{}{"r1col1", "this is ignored", "r1col3", "10/12/2022"}, &g)

	if err != nil {
		t.Error(err)
	}
	if g.Col1 != "r1col1" {
		t.Errorf("g.Col1 is %s but must be r1col1", g.Col1)
		t.FailNow()
	}
	if g.Col3 != "r1col3" {
		t.Errorf("g.Col3 is %s but must be r1col3", g.Col3)
		t.FailNow()
	}
	if g.Date4 != time.Date(2022, 12, 10, 0, 0, 0, 0, time.UTC) {
		t.Errorf("g.Date4 is %s but must be the given timestamp", g.Date4)
		t.FailNow()
	}

}

func TestSheetParserGaps_Marshal(t *testing.T) {
	p := &SheetParser{}
	g := testDataGaps{
		Col1:  "r1Col1",
		Col3:  "r1Col3",
		Date4: time.Date(2022, 12, 10, 0, 0, 0, 0, time.UTC),
	}

	out, err := p.Marshal(&g)
	if err != nil {
		t.Error(err)
	}

	if out[0].(string) != "r1Col1" {
		t.Errorf("field 0 is %s but must be r1Col1", out[0])
		t.FailNow()
	}
	if out[1] != nil {
		t.Errorf("field 1 is %s but must be nil", out[1])
		t.FailNow()
	}
	if out[2].(string) != "r1Col3" {
		t.Errorf("field 2 is %s but must be r1Col1", out[2])
		t.FailNow()
	}
	if out[3].(string) != "10/12/2022" {
		t.Errorf("field 3 is %s but must be 10/12/2022", out[3])
		t.FailNow()
	}
}
