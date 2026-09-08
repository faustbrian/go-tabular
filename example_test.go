package tabular_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	tabular "github.com/faustbrian/go-tabular"
)

func ExampleNewDelimitedReader() {
	reader, err := tabular.NewDelimitedReader(strings.NewReader("name;city\nAda;Helsinki\n"), tabular.DelimitedConfig{
		Delimiter:      ';',
		MaxRecordBytes: 64 << 10,
		MaxFieldBytes:  16 << 10,
		Header: &tabular.HeaderConfig{
			Case:             tabular.HeaderCaseLower,
			RejectEmpty:      true,
			RejectDuplicates: true,
		},
	})
	if err != nil {
		panic(err)
	}
	header, err := reader.Header()
	if err != nil {
		panic(err)
	}
	fmt.Println(header)
	for {
		row, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			panic(readErr)
		}
		fmt.Println(row)
	}
	// Output:
	// [name city]
	// [Ada Helsinki]
}

func ExampleNewFixedWidthReader() {
	reader, err := tabular.NewFixedWidthReader(strings.NewReader("001Ada       Helsinki  \n"), tabular.FixedWidthConfig{
		Fields: []tabular.FixedWidthField{
			{Name: "id", Start: 0, End: 3},
			{Name: "name", Start: 3, End: 13, TrimSpace: true},
			{Name: "city", Start: 13, End: 23, TrimSpace: true},
		},
	})
	if err != nil {
		panic(err)
	}
	row, err := reader.Read()
	if err != nil {
		panic(err)
	}
	fmt.Println(reader.Fields())
	fmt.Println(row)
	// Output:
	// [id name city]
	// [001 Ada Helsinki]
}

func ExampleOpenZIP() {
	file, err := os.Open("testdata/archive/import.zip")
	if err != nil {
		panic(err)
	}
	defer closeOrPanic(file)
	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	archive, err := tabular.OpenZIP(file, info.Size(), tabular.ZIPConfig{
		MaxEntries:          20,
		MaxEntryBytes:       64 << 20,
		MaxTotalBytes:       64 << 20,
		MaxCompressionRatio: 100,
		RejectSymlinks:      true,
	})
	if err != nil {
		panic(err)
	}
	entry, err := archive.Open("orders.csv")
	if err != nil {
		panic(err)
	}
	defer closeOrPanic(entry)

	reader, err := tabular.NewDelimitedReader(entry, tabular.DelimitedConfig{
		Delimiter:       ';',
		FieldsPerRecord: 2,
		MaxRecordBytes:  64 << 10,
		MaxFieldBytes:   16 << 10,
		Header: &tabular.HeaderConfig{
			RejectEmpty:      true,
			RejectDuplicates: true,
		},
	})
	if err != nil {
		panic(err)
	}
	header, err := reader.Header()
	if err != nil {
		panic(err)
	}
	row, err := reader.Read()
	if err != nil {
		panic(err)
	}
	fmt.Println(header)
	fmt.Println(row)
	// Output:
	// [id amount]
	// [1 12,50]
}

func ExampleOpenSpreadsheet() {
	file, err := os.Open("testdata/spreadsheet/sample.xlsx")
	if err != nil {
		panic(err)
	}
	defer closeOrPanic(file)
	info, err := file.Stat()
	if err != nil {
		panic(err)
	}

	reader, err := tabular.OpenSpreadsheet(file, info.Size(), tabular.SpreadsheetConfig{
		Format: tabular.FormatXLSX,
		Sheet:  "Orders",
		Header: &tabular.HeaderConfig{
			Case:             tabular.HeaderCaseLower,
			RejectEmpty:      true,
			RejectDuplicates: true,
		},
		MaxRecordBytes: 64 << 10,
		MaxFieldBytes:  16 << 10,
		MaxSheets:      8,
		ZIP: tabular.ZIPConfig{
			MaxEntryBytes:       32 << 20,
			MaxTotalBytes:       64 << 20,
			MaxCompressionRatio: 100,
			RejectSymlinks:      true,
		},
	})
	if err != nil {
		panic(err)
	}
	defer closeOrPanic(reader)
	header, err := reader.Header()
	if err != nil {
		panic(err)
	}
	row, err := reader.Read()
	if err != nil {
		panic(err)
	}
	fmt.Println(header)
	fmt.Println(row)
	// Output:
	// [name city amount active]
	// [Alice Helsinki 12.5 1]
}

func closeOrPanic(closer io.Closer) {
	if err := closer.Close(); err != nil {
		panic(err)
	}
}
