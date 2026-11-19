package drivermgr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/gateway/drivermgr/v1"
)

func TestExportTemplate(t *testing.T) {
	service, _ := newTestService(t)
	created, err := service.Create(context.Background(), &v1.CreateDriverReq{
		DriverType: "custom",
		Model:      "my-driver",
		PluginName: "my-driver",
		Schema:     json.RawMessage(`{"fields":[{"name":"transport","labelZh":"传输方式","labelEn":"Transport","regex":"^(tcp|udp|serial)$","sort":1,"type":"select","options":[{"value":"tcp","labelZh":"TCP","labelEn":"TCP"}]}]}`),
	})
	require.NoError(t, err)

	name, data, err := service.ExportTemplate(context.Background(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "my-driver.zip", name)

	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	files := make(map[string]string)
	for _, file := range reader.File {
		source, err := file.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(source)
		require.NoError(t, err)
		require.NoError(t, source.Close())
		files[file.Name] = string(content)
	}
	for _, key := range []string{"driver.go", "plugin.go", "README.md", "schema.json", "schema.example.json", "go.mod", "build.sh", "manifest.json", "driver_test.go"} {
		assert.Contains(t, files, key)
	}
	assert.Contains(t, files["README.md"], "concurrency")
	assert.Contains(t, files["README.md"], "GatewayDriver")
	assert.Contains(t, files["README.md"], "plugins/my-driver/linux-amd64/my-driver.so")
	assert.Contains(t, files["README.md"], "schema.json")
	assert.Contains(t, files["README.md"], "Restart the gateway")
	assert.Contains(t, files["plugin.go"], "GatewayDriver")
}
