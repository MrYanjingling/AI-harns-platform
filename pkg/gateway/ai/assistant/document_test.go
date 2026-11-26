package assistant

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"harns-platform/pkg/gateway/ai/tool"
	"harns-platform/pkg/gateway/sqlite"
)

func TestParseDOCXText(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("word/document.xml")
	require.NoError(t, err)
	_, err = file.Write([]byte(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Hello</w:t></w:r></w:p><w:p><w:r><w:t>世界</w:t></w:r></w:p></w:body></w:document>`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	text, err := parseDocumentText(".docx", buffer.Bytes())
	require.NoError(t, err)
	assert.Contains(t, text, "Hello")
	assert.Contains(t, text, "世界")
}

func TestSplitDocumentChunks(t *testing.T) {
	text := string(bytes.Repeat([]byte("a"), documentChunkRunes*2+10))
	chunks := splitDocumentChunks(text)
	require.Len(t, chunks, 3)
	assert.Len(t, []rune(chunks[0]), documentChunkRunes)
	assert.Contains(t, chunks[1], chunks[0][len(chunks[0])-documentChunkOverlap:])
}

func TestDocumentStoreSearchAndSample(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	store := NewDocumentStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	documents, err := store.SaveMany(context.Background(), "user-1", []parsedDocument{{
		Name: "manual.docx",
		Ext:  ".docx",
		Size: 1024,
		Text: "如何设置 Modbus 通讯地址",
		Chunks: []string{
			"如何设置 Modbus 通讯地址",
			"产品协议包含寄存器说明",
			"其他无关内容",
		},
	}})
	require.NoError(t, err)
	require.Len(t, documents, 1)

	items, err := store.Search(context.Background(), documents, "设置地址", 5)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	assert.Equal(t, "manual.docx", items[0].DocumentName)
	assert.Contains(t, items[0].Text, "Modbus")

	samples, err := store.SampleChunks(context.Background(), documents, 2)
	require.NoError(t, err)
	assert.Len(t, samples, 2)

	require.NoError(t, store.Delete(context.Background(), "user-1", documents[0].ID))
	_, err = store.GetMany(context.Background(), "user-1", []string{documents[0].ID})
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestRetrievalTerms(t *testing.T) {
	fts, like := retrievalTerms("如何设置 Modbus 地址")
	assert.NotEmpty(t, fts)
	assert.Contains(t, fts, "mod")
	assert.Contains(t, like, "modbus")
}

func TestParseXLSXText(t *testing.T) {
	file := excelize.NewFile()
	defer file.Close()
	require.NoError(t, file.SetCellValue("Sheet1", "A1", "产品"))
	require.NoError(t, file.SetCellValue("Sheet1", "B1", "协议"))
	buffer, err := file.WriteToBuffer()
	require.NoError(t, err)

	text, err := parseDocumentText(".xlsx", buffer.Bytes())
	require.NoError(t, err)
	assert.Contains(t, text, "产品")
	assert.Contains(t, text, "协议")
}

func TestAppendDocuments(t *testing.T) {
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	store := NewDocumentStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	documents, err := store.SaveMany(context.Background(), "user-1", []parsedDocument{{
		Name:   "guide.pdf",
		Ext:    ".pdf",
		Size:   1024,
		Text:   "如何设置 Modbus 通讯地址",
		Chunks: []string{"如何设置 Modbus 通讯地址", "第四章介绍通讯协议"},
	}})
	require.NoError(t, err)
	service := &Service{documents: store}
	messages := []chatMessage{{Role: "user", Content: "如何设置通讯地址"}}
	require.NoError(t, service.appendDocuments(context.Background(), tool.Principal{UserID: "user-1"}, []string{documents[0].ID}, messages))
	assert.Contains(t, messages[0].Content, "guide.pdf")
	assert.Contains(t, messages[0].Content, "Modbus")
}

func TestParseLegacyDOC(t *testing.T) {
	data, err := os.ReadFile("testdata/chinese.doc")
	require.NoError(t, err)
	text, err := parseDocumentText(".doc", data)
	require.NoError(t, err)
	assert.Contains(t, text, "产品说明书")
	assert.Contains(t, text, "Modbus")
}

func TestParseLegacyXLS(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.xls")
	require.NoError(t, err)
	text, err := parseDocumentText(".xls", data)
	require.NoError(t, err)
	assert.Contains(t, text, "Table")
	assert.Contains(t, text, "code1")
}
