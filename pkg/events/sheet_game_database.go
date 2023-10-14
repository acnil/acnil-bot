package events

import "google.golang.org/api/sheets/v4"

type SheetGameDatabase struct {
	ReadRange string
	Sheet     string
	SheetID   string
}

func NewSheetGameDatabase(srv *sheets.Service, sheetID string, sheet string) *SheetGameDatabase {
	return &SheetGameDatabase{
		ReadRange: "A:N", // Todo, defined based on number of columns
		Sheet:     sheet,
		SheetID:   sheetID,
	}
}
