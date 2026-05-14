package gsheets

import (
	"context"
	"fmt"
	"log"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	credentialsFile = "google-credentials.json"
	sheetTab        = "T52026"
	mssvColumn      = "C" // MSSV is in Column C
	fundColumn      = "F" // Quỹ (Checkbox) is in Column F
)

// MarkAsPaid finds the row for a given MSSV and checks the "Quỹ" box.
func MarkAsPaid(spreadsheetId string, mssv string) error {
	ctx := context.Background()

	// 1. Authenticate with Service Account
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(sheets.SpreadsheetsScope))
	if err != nil {
		return fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	// 2. Find the row index by searching Column C
	// We fetch the entire Column C to find the MSSV
	readRange := fmt.Sprintf("%s!%s:%s", sheetTab, mssvColumn, mssvColumn)
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	if len(resp.Values) == 0 {
		return fmt.Errorf("no data found in column %s", mssvColumn)
	}

	rowIndex := -1
	for i, row := range resp.Values {
		if len(row) > 0 {
			// Convert to string and compare
			val := fmt.Sprintf("%v", row[0])
			if val == mssv {
				rowIndex = i + 1 // Sheets API is 1-indexed
				break
			}
		}
	}

	if rowIndex == -1 {
		return fmt.Errorf("MSSV %s not found in spreadsheet", mssv)
	}

	// 3. Update Column F for that row to TRUE (checks the box)
	updateRange := fmt.Sprintf("%s!%s%d", sheetTab, fundColumn, rowIndex)
	
	var vr sheets.ValueRange
	vr.Values = append(vr.Values, []interface{}{true})

	_, err = srv.Spreadsheets.Values.Update(spreadsheetId, updateRange, &vr).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("unable to update cell: %v", err)
	}

	log.Printf("GOOGLE SHEETS: Successfully marked MSSV %s as paid in row %d", mssv, rowIndex)
	return nil
}

// ExistsMSSV checks if the given MSSV exists in Column C of the sheetTab.
func ExistsMSSV(spreadsheetId string, mssv string) (bool, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile(credentialsFile), option.WithScopes(sheets.SpreadsheetsReadonlyScope))
	if err != nil {
		return false, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}

	readRange := fmt.Sprintf("%s!%s:%s", sheetTab, mssvColumn, mssvColumn)
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		return false, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}

	for _, row := range resp.Values {
		if len(row) > 0 {
			if fmt.Sprintf("%v", row[0]) == mssv {
				return true, nil
			}
		}
	}
	return false, nil
}
