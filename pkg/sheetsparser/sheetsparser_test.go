package sheetsparser

import (
	"testing"
	"time"
)

type testDataHappy struct {
	Ignored string

	Col1       string    `col:"0"`
	Col2       string    `col:"1"`
	Col3       string    `col:"2"`
	Date4      time.Time `col:"3"`
	IsEmpty    *string   `col:"4"`
	IsNotEmpty *string   `col:"5"`
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
	S := "r1Col5"
	p := &SheetParser{}
	g := testDataHappy{
		Col1:       "r1Col1",
		Col2:       "r1Col2",
		Col3:       "r1Col3",
		Date4:      time.Date(2022, 12, 10, 0, 0, 0, 0, time.UTC),
		IsNotEmpty: &S,
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
	if out[4] != nil {
		t.Errorf("field 4 is %s but must be nil", out[4])
		t.FailNow()
	}
	if out[5].(string) != "r1Col5" {
		t.Errorf("field 5 is %s but must be r1Col5", out[3])
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

type testReadOnly struct {
	ReadOnly string `col:"0,ro"`
}

func TestSheetReadOnly_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	test := testReadOnly{}

	err := p.Unmarshal([]interface{}{"rovalue"}, &test)

	if err != nil {
		t.Error(err)
	}
	if test.ReadOnly != "rovalue" {
		t.Errorf("g.ReadOnly is %s but must be rovalue", test.ReadOnly)
		t.FailNow()
	}
}

func TestSheetReadOnly_Marshal(t *testing.T) {
	p := &SheetParser{}
	test := testReadOnly{
		ReadOnly: "new value",
	}

	out, err := p.Marshal(&test)
	if err != nil {
		t.Error(err)
	}

	if out[0] != nil {
		t.Errorf("field 0 is %s but must be nil", out[0])
		t.FailNow()
	}
}

// Must error?
// Must parse both?
type testWriteOnly struct {
	WriteOnly string `col:"0,wo"`
}

func TestSheetWriteOnly_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	test := testWriteOnly{}

	err := p.Unmarshal([]interface{}{"wovalue", "value"}, &test)

	if err != nil {
		t.Error(err)
	}
	if test.WriteOnly != "" {
		t.Errorf("g.WriteOnly is %s but must be empty", test.WriteOnly)
		t.FailNow()
	}
}

func TestSheetWriteOnly_Marshal(t *testing.T) {
	p := &SheetParser{}
	test := testWriteOnly{
		WriteOnly: "new value",
	}

	out, err := p.Marshal(&test)
	if err != nil {
		t.Error(err)
	}

	if out[0].(string) != "new value" {
		t.Errorf("field 0 is %s but must be new value", out[0])
		t.FailNow()
	}
}

type testReadAndWrite struct {
	Before    string `col:"0"`
	ReadOnly  string `col:"1,ro"`
	WriteOnly string `col:"1,wo"`
	After     string `col:"3"`
}

func TestSheetReadAndWrite_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	test := testReadAndWrite{}

	err := p.Unmarshal([]interface{}{"before", "value"}, &test)

	if err != nil {
		t.Error(err)
	}
	if test.WriteOnly != "" {
		t.Errorf("g.WriteOnly is %s but must be empty", test.WriteOnly)
		t.FailNow()
	}
	if test.ReadOnly != "value" {
		t.Errorf("g.ReadOnly is %s but must be empty", test.ReadOnly)
		t.FailNow()
	}
}

func TestSheetReadAndWrite_Marshal(t *testing.T) {
	p := &SheetParser{}
	test := testReadAndWrite{
		Before:    "before",
		ReadOnly:  "read value",
		WriteOnly: "write value",
		After:     "after",
	}

	out, err := p.Marshal(&test)
	if err != nil {
		t.Error(err)
	}
	if len(out) != 4 {
		t.Errorf("Expected 2 fields but got %d, %s", len(out), out)
		t.FailNow()
	}

	if out[1].(string) != "write value" {
		t.Errorf("field 0 is %s but must be write value", out[0])
		t.FailNow()
	}
}

type testWriteAndRead struct {
	// In this order the code fails :blow:
	WriteOnly string `col:"0,wo"`
	ReadOnly  string `col:"0,ro"`
}

func TestSheetWriteAndRead_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	test := testWriteAndRead{}

	err := p.Unmarshal([]interface{}{"value"}, &test)

	if err != nil {
		t.Error(err)
	}
	if test.WriteOnly != "" {
		t.Errorf("g.WriteOnly is %s but must be empty", test.WriteOnly)
		t.FailNow()
	}
	if test.ReadOnly != "value" {
		t.Errorf("g.ReadOnly is %s but must be empty", test.ReadOnly)
		t.FailNow()
	}
}

func TestSheetWriteAndRead_Marshal(t *testing.T) {
	p := &SheetParser{}
	test := testWriteAndRead{
		WriteOnly: "write value",
		ReadOnly:  "read value",
	}

	out, err := p.Marshal(&test)
	if err != nil {
		t.Error(err)
	}

	if v, ok := out[0].(string); !ok || v != "write value" {
		t.Errorf("field 0 is %s but must be write value", out[0])
		t.FailNow()
	}
}

type testGame struct {
	ReturnDate        time.Time `col:"1,ro"`
	ReturnDateFormula *string   `col:"1,wo"`
}

func TestGameWriteAndRead_Unmarshal(t *testing.T) {
	p := &SheetParser{}
	test := testGame{}

	err := p.Unmarshal([]interface{}{nil, "15/04/2023"}, &test)

	if err != nil {
		t.Error(err)
	}
	if test.ReturnDate.Format(p.DateFormat) == "15/04/2023" {
		t.Errorf("g.ReturnDate is %s but must be 15/04/2023", test.ReturnDate.Format(p.DateFormat))
		t.FailNow()
	}
	if test.ReturnDateFormula != nil {
		t.Errorf("g.ReturnDateFormula is %#v but must be nil", test.ReturnDateFormula)
		t.FailNow()
	}
}

func TestGameWriteAndRead_Marshal(t *testing.T) {
	p := &SheetParser{}
	test := testGame{
		ReturnDate:        time.Time{},
		ReturnDateFormula: nil,
	}

	out, err := p.Marshal(&test)
	if err != nil {
		t.Error(err)
	}

	if len(out) != 2 {
		t.Errorf("Must return 2 fields but returned %d, %#v", len(out), out)
		t.FailNow()
	}

	if out[1] != nil {
		t.Errorf("field 1 must be nil but it is %#v", out[1])
		t.FailNow()
	}

}
