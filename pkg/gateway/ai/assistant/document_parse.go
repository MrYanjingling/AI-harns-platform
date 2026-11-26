package assistant

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	doclib "github.com/bytexlee/doc"
	xls "github.com/extrame/xls"
	"github.com/ledongthuc/pdf"
	"github.com/xuri/excelize/v2"
)

const (
	documentChunkRunes   = 1200
	documentChunkOverlap = 160
)

var supportedDocumentExtensions = map[string]bool{
	".txt":  true,
	".log":  true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".csv":  true,
	".md":   true,
	".xml":  true,
	".doc":  true,
	".docx": true,
	".xls":  true,
	".xlsx": true,
	".pdf":  true,
}

func supportedDocumentExt(ext string) bool {
	return supportedDocumentExtensions[strings.ToLower(ext)]
}

func parseDocumentText(ext string, data []byte) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			text = ""
			err = fmt.Errorf("parse document panic: %v", recovered)
		}
	}()
	switch strings.ToLower(ext) {
	case ".doc":
		return extractDOCBinaryText(data)
	case ".docx":
		return extractDOCXText(data)
	case ".xls":
		return extractXLSBinaryText(data)
	case ".xlsx":
		return extractXLSXText(data)
	case ".pdf":
		return extractPDFText(data)
	default:
		if !utf8.Valid(data) {
			return "", fmt.Errorf("text file is not valid UTF-8")
		}
		return string(data), nil
	}
}

func extractDOCBinaryText(data []byte) (string, error) {
	text, err := doclib.ParseDoc(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(text)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func extractXLSBinaryText(data []byte) (string, error) {
	book, err := xls.OpenReader(bytes.NewReader(data), "utf-8")
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for sheetIndex := 0; sheetIndex < book.NumSheets(); sheetIndex++ {
		sheet := book.GetSheet(sheetIndex)
		if sheet == nil {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString("# ")
		builder.WriteString(sheet.Name)
		builder.WriteByte('\n')
		for rowIndex := 0; rowIndex <= int(sheet.MaxRow); rowIndex++ {
			row := sheet.Row(rowIndex)
			if row == nil {
				continue
			}
			last := row.LastCol()
			if last < 0 {
				continue
			}
			cells := make([]string, 0, last+1)
			for column := 0; column <= last; column++ {
				cells = append(cells, row.Col(column))
			}
			builder.WriteString(strings.Join(cells, "\t"))
			builder.WriteByte('\n')
		}
	}
	return builder.String(), nil
}

func extractDOCXText(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	var document *zip.File
	for _, item := range reader.File {
		if item.Name == "word/document.xml" {
			document = item
			break
		}
	}
	if document == nil {
		return "", fmt.Errorf("word/document.xml not found")
	}
	file, err := document.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()

	decoder := xml.NewDecoder(file)
	var builder strings.Builder
	textDepth := 0
	lastWasNewline := true
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "t":
				textDepth++
			case "tab":
				builder.WriteByte('\t')
				lastWasNewline = false
			case "br":
				builder.WriteByte('\n')
				lastWasNewline = true
			case "p":
				if builder.Len() > 0 && !lastWasNewline {
					builder.WriteByte('\n')
				}
				lastWasNewline = true
			}
		case xml.EndElement:
			if item.Name.Local == "t" && textDepth > 0 {
				textDepth--
			}
			if item.Name.Local == "p" {
				lastWasNewline = true
			}
		case xml.CharData:
			if textDepth > 0 {
				value := string(item)
				builder.WriteString(value)
				lastWasNewline = strings.HasSuffix(value, "\n")
			}
		}
	}
	return builder.String(), nil
}

func extractPDFText(data []byte) (string, error) {
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	text, err := reader.GetPlainText()
	if err != nil {
		return "", err
	}
	content, err := io.ReadAll(text)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func extractXLSXText(data []byte) (string, error) {
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer file.Close()

	var builder strings.Builder
	for _, sheet := range file.GetSheetList() {
		rows, err := file.GetRows(sheet)
		if err != nil {
			return "", err
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString("# ")
		builder.WriteString(sheet)
		builder.WriteByte('\n')
		for _, row := range rows {
			if len(row) == 0 {
				builder.WriteByte('\n')
				continue
			}
			builder.WriteString(strings.Join(row, "\t"))
			builder.WriteByte('\n')
		}
	}
	return builder.String(), nil
}

func normalizeDocumentText(value string) string {
	value = strings.TrimPrefix(value, "\ufeff")
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\u00a0", " ")
	lines := strings.Split(value, "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	value = strings.Join(lines, "\n")
	for strings.Contains(value, "\n\n\n") {
		value = strings.ReplaceAll(value, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(value)
}

func splitDocumentChunks(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	runes := []rune(value)
	if len(runes) <= documentChunkRunes {
		return []string{value}
	}
	chunks := make([]string, 0, len(runes)/documentChunkRunes+1)
	for start := 0; start < len(runes); {
		end := start + documentChunkRunes
		if end >= len(runes) {
			end = len(runes)
		} else {
			searchStart := start + documentChunkRunes/2
			for index := end; index >= searchStart; index-- {
				if runes[index-1] == '\n' {
					end = index
					break
				}
			}
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end >= len(runes) {
			break
		}
		next := end - documentChunkOverlap
		if next <= start {
			next = end
		}
		start = next
	}
	return chunks
}

func documentName(value string) string {
	name := strings.TrimSpace(filepath.Base(value))
	if name == "" || name == "." {
		return "document"
	}
	return name
}
