package drivermgr

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"text/template"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/pluginmgr"
)

type templateData struct {
	DriverType  string
	DriverModel string
	PluginName  string
	Schema      string
	GoVersion   string
	ABIVersion  string
}

// ExportTemplate builds the custom driver source template as a ZIP archive.
func (s *Service) ExportTemplate(ctx context.Context, id string) (string, []byte, error) {
	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, response.ErrDriverNotFound(id)
	}
	if err != nil {
		return "", nil, response.ErrInternal(err)
	}
	if item.BuiltinDriver {
		return "", nil, response.ErrDriverInvalid("builtin driver template is not available")
	}
	goVersion := strings.TrimPrefix(runtime.Version(), "go")
	data := &templateData{
		DriverType:  item.DriverType,
		DriverModel: item.Model,
		PluginName:  item.PluginName,
		Schema:      string(item.Schema),
		GoVersion:   goVersion,
		ABIVersion:  pluginmgr.ABIVersion,
	}
	archive, err := buildTemplateArchive(data)
	if err != nil {
		return "", nil, response.ErrInternal(err)
	}
	return item.PluginName + ".zip", archive, nil
}

func buildTemplateArchive(data *templateData) ([]byte, error) {
	files := map[string]string{
		"go.mod":              templateGoMod,
		"driver.go":           templateDriver,
		"plugin.go":           templatePlugin,
		"driver_test.go":      templateDriverTest,
		"README.md":           templateReadme,
		"schema.json":         templateSchema,
		"schema.example.json": templateSchemaExample,
		"build.sh":            templateBuild,
		"manifest.json":       templateManifest,
	}
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for name, source := range files {
		body, err := renderTemplate(name, source, data)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		if _, err := entry.Write(body); err != nil {
			_ = writer.Close()
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func renderTemplate(name, source string, data *templateData) ([]byte, error) {
	tpl, err := template.New(name).Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse template %s: %w", name, err)
	}
	var output bytes.Buffer
	if err := tpl.Execute(&output, data); err != nil {
		return nil, fmt.Errorf("render template %s: %w", name, err)
	}
	return output.Bytes(), nil
}

const templateGoMod = `module {{.PluginName}}

go {{.GoVersion}}

require harns-platform v0.0.0

// Change this path to the local AI-harns-platform source checkout.
replace harns-platform => /path/to/AI-harns-platform
`

const templateManifest = `{
  "driverType": "{{.DriverType}}",
  "driverModel": "{{.DriverModel}}",
  "pluginName": "{{.PluginName}}",
  "abiVersion": "{{.ABIVersion}}"
}
`

const templateBuild = `#!/bin/sh
set -eu

GOOS=${GOOS:-linux}
GOARCH=${GOARCH:-$(go env GOARCH)}
OUTPUT="dist/${GOOS}-${GOARCH}/{{.PluginName}}.so"

mkdir -p "$(dirname "$OUTPUT")"
CGO_ENABLED=1 GOOS="$GOOS" GOARCH="$GOARCH" go build -buildmode=plugin -o "$OUTPUT" .
echo "built $OUTPUT"
`

const templateReadme = `# {{.PluginName}}

AI-harns-platform 自定义驱动插件模板。
AI-harns-platform custom driver plugin template.

## 1. 编写 plugin.go 和 driver.go / Implement plugin.go and driver.go

下载模板后，通常只需要修改 driver.go 和 plugin.go。其他文件用于构建、schema 参考和测试。
After downloading the template, you normally only need to modify driver.go and plugin.go. The other files are build configuration, schema references, and tests.

### plugin.go

保持 package main，并导出以下符号，不要修改函数名和签名：
Keep package main and export the following symbol. Do not change the function name or signature:

~~~go
func GatewayDriver() pluginmgr.Driver
~~~

该函数返回 driver.go 中的 driver 实例即可：
The function only needs to return the driver instance defined in driver.go:

~~~go
func GatewayDriver() pluginmgr.Driver {
	return &driver{}
}
~~~

### driver.go

driver.go 负责协议本身。宿主管轮询、连接池、状态、重试、计算变量和 trace，插件不要重复实现这些能力。
driver.go implements the protocol. The host owns polling, connection pooling, status, retries, calculated variables, and tracing. Do not reimplement those features in the plugin.

按方法实现：
Implement the methods as follows:

- Descriptor: 返回插件标识。driverType、driverModel、pluginName 必须与平台中创建的驱动完全一致，否则启动时会拒绝加载。Version 可自行维护。
  Descriptor: returns plugin identity. driverType, driverModel, and pluginName must exactly match the driver created in the platform, otherwise startup rejects the plugin. Version is yours to maintain.

- ValidateConfig: 解析协议配置并校验必填项。协议配置字段名必须与平台 schema 的字段名一致。模板包含 transport、location、concurrency、timeoutMs。
  ValidateConfig: parses the protocol config and validates required values. Config field names must match the platform schema field names. The template includes transport, location, concurrency, and timeoutMs.

- TransportKind: 返回 tcp、udp、serial 或 custom。tcp/udp 使用连接池；serial/custom 由宿主强制单连接。
  TransportKind: returns tcp, udp, serial, or custom. tcp/udp use the connection pool; serial/custom are forced to a single connection by the host.

- NewTransport: 建立并返回一个连接对象。连接对象必须实现 Exchange、Available 和 Close。不要在插件里自行启动轮询 goroutine。
  NewTransport: opens and returns one connection. The connection must implement Exchange, Available, and Close. Do not start polling goroutines in the plugin.

- BuildReadFrames: 将变量按协议请求分组，返回一个或多个 ReadFrame。每个 ReadFrame 包含请求字节、对应变量和可选 Context，供 DecodeRead 使用。
  BuildReadFrames: groups variables into protocol requests and returns one or more ReadFrame values. Each ReadFrame contains request bytes, its variables, and optional Context for DecodeRead.

- DecodeRead: 解码响应，为每个变量返回 DecodedValue。可以返回已成功解析的部分值以及一个错误，宿主会保留成功值并记录采集错误。
  DecodeRead: decodes the response and returns one DecodedValue per variable. It may return successfully decoded values together with an error; the host keeps the successful values and records a collection error.

- BuildWrite: 将一个写操作编码为协议请求字节。
  BuildWrite: encodes one write action into protocol request bytes.

- DecodeWrite: 校验写响应。成功返回 nil，失败返回错误。
  DecodeWrite: validates the write response. Return nil on success and an error on failure.

推荐实现顺序：Descriptor -> ValidateConfig -> TransportKind -> NewTransport -> BuildReadFrames -> DecodeRead -> BuildWrite -> DecodeWrite。
Recommended order: Descriptor -> ValidateConfig -> TransportKind -> NewTransport -> BuildReadFrames -> DecodeRead -> BuildWrite -> DecodeWrite.

## 2. 编译为 .so / Build the .so

Go plugin 只支持 Linux，必须开启 CGO，并且插件与网关使用完全相同的 Go 版本和依赖源码。
Go plugins are supported only on Linux, require CGO, and must use exactly the same Go version and dependency source as the gateway.

先修改 go.mod 中的 replace 路径，指向本机 AI-harns-platform 源码目录：
First update the replace path in go.mod to point to the local AI-harns-platform source checkout:

~~~text
replace harns-platform => /path/to/AI-harns-platform
~~~

编译当前架构：
Build for the current architecture:

~~~bash
chmod +x build.sh
./build.sh
~~~

交叉编译其他架构时，必须在具备对应 C 工具链的 Linux 环境或 CI 中执行：
To build another architecture, run on Linux or CI with the matching C toolchain:

~~~bash
GOOS=linux GOARCH=amd64 ./build.sh
GOOS=linux GOARCH=arm64 ./build.sh
~~~

输出文件：
Output files:

~~~text
dist/linux-amd64/{{.PluginName}}.so
dist/linux-arm64/{{.PluginName}}.so
~~~

上传的是 .so 文件，不是模板 zip。amd64 和 arm64 需要分别编译、分别上传。
Upload the .so file, not the template zip. Build and upload amd64 and arm64 separately.

## 3. 上传到平台 / Upload to the platform

在驱动管理页面打开对应驱动的“上传插件”，选择编译好的 .so。
Open the driver page and use “Upload plugin” to select the compiled .so.

上传成功后，默认保存路径为：
After a successful upload, the default path is:

~~~text
plugins/{{.PluginName}}/linux-amd64/{{.PluginName}}.so
plugins/{{.PluginName}}/linux-arm64/{{.PluginName}}.so
~~~

plugins 目录默认与 data、log 同级，相对于网关工作目录。可以通过网关配置项 plugin-dir 修改。
The plugins directory is beside data and log by default, relative to the gateway working directory. It can be changed with the plugin-dir config option.

再次上传同一插件会覆盖对应架构的 .so 文件。
Uploading the same plugin again overwrites the .so file for that architecture.

## 4. 修改 schema / Update the schema

模板中的 schema.json 只是当前平台驱动 schema 的参考副本。它不会编译进 .so，也不会随 .so 上传。
schema.json in the template is only a reference copy of the driver schema in the platform. It is not compiled into the .so and is not uploaded with it.

需要修改协议配置字段、变量地址规则、数据类型时：
To change protocol config fields, variable address rules, or data types:

1. 在平台驱动管理页面编辑该驱动的 schema 并保存。
   Edit and save the driver schema in the platform driver page.
2. 同步修改本地 schema.json，方便后续维护和重新下载模板时对比。
   Keep the local schema.json in sync for maintenance and comparison.
3. 确保 driver.go 中 protocolConfig 的字段名与 schema 字段名一致。
   Make sure protocolConfig field names in driver.go match the schema field names.

修改 schema 不需要重新编译 .so，但修改协议实现代码需要重新编译并上传 .so。
Changing the schema does not require rebuilding the .so. Changing protocol implementation code does.

## 5. 重启网关 / Restart the gateway

插件上传后不会热加载。重启网关：
Uploaded plugins are not hot-loaded. Restart the gateway:

~~~bash
./gateway
~~~

启动时会从同一 plugin-dir 加载自定义驱动。成功时日志包含：
At startup the gateway loads custom drivers from the same plugin-dir. On success the log contains:

~~~text
Loaded custom driver plugin: driverType=... driverModel=... plugin=... path=...
~~~

如果未加载，检查：
If it is not loaded, check:

- .so 是否在对应 linux-GOARCH 目录下。
  Whether the .so is in the matching linux-GOARCH directory.
- 是否在 Linux 上运行网关并使用 Linux 插件。
  Whether the gateway and plugin both run on Linux.
- Go 版本、GOARCH、ABI 是否与网关一致。
  Whether Go version, GOARCH, and ABI match the gateway.
- Descriptor 中的 driverType、driverModel、pluginName 是否与平台驱动一致。
  Whether descriptor driverType, driverModel, and pluginName match the platform driver.

## 其他操作 / Other operations

- 修改插件代码后：运行 go test ./...，重新编译 .so -> 再次上传覆盖 -> 重启网关。
  After changing plugin code: run go test ./..., rebuild the .so, upload it again, and restart the gateway.
- 修改 schema 后：在平台保存即可，不需要重启网关。
  After changing the schema: save it in the platform. No gateway restart is required.
- 删除驱动只会删除平台记录，不会自动删除 plugins 下的 .so。确认不再使用后可手动删除对应目录。
  Deleting a driver removes the platform record but does not delete the .so under plugins. Remove the directory manually after confirming it is no longer needed.
- serial/custom 传输的 concurrency 由宿主强制为 1。
  The host forces concurrency to 1 for serial/custom transports.
- manifest.json 仅用于说明插件标识，运行时不读取。
  manifest.json is only a reference for plugin identity and is not read at runtime.

## 必填协议字段 / Required protocol fields

- transport: 必填，tcp、udp 或 serial。
  transport: required, one of tcp, udp, or serial.
- location: 必填。TCP/UDP 为 host:port；串口为 /dev/ttyS0 或 COM1。
  location: required. Use host:port for TCP/UDP, or /dev/ttyS0 / COM1 for serial.
- concurrency: 连接池大小，默认 1。TCP/UDP 表示并发连接数；serial/custom 固定为 1。
  concurrency: connection pool size, default 1. For TCP/UDP it is the number of concurrent connections; serial/custom are always 1.

参见 schema.example.json 中的最小 schema 示例。
See schema.example.json for the minimum schema example.

## ABI

- Plugin symbol: GatewayDriver
- ABI version: {{.ABIVersion}}
- Driver type: {{.DriverType}}
- Driver model: {{.DriverModel}}
- Plugin name: {{.PluginName}}

## 错误行为 / Error behavior

从对应方法返回错误：
Return an error from the corresponding method:

- 配置错误或建帧失败：宿主标记为 error。
  Invalid config or frame build failure: the host sets error.
- 连接失败：宿主标记为 unconnected。
  Connection failure: the host sets unconnected.
- 读取或解码失败：宿主标记为 collectingError，并保留已成功解析的值。
  Read or decode failure: the host sets collectingError and keeps successfully decoded values.
- 插件 panic：宿主 recover 后标记为 collectingError。
  Plugin panic: the host recovers and records collectingError.

不要在插件中调用状态接口。
Do not call status APIs from the plugin.
`
const templateSchema = `{{.Schema}}`

const templateSchemaExample = `{
  "fields": [
    {
      "name": "transport",
      "labelZh": "传输方式",
      "labelEn": "Transport",
      "regex": "^(tcp|udp|serial)$",
      "sort": 1,
      "type": "select",
      "required": true,
      "options": [
        {"value":"tcp","labelZh":"TCP","labelEn":"TCP"},
        {"value":"udp","labelZh":"UDP","labelEn":"UDP"},
        {"value":"serial","labelZh":"串口","labelEn":"Serial"}
      ]
    },
    {
      "name": "location",
      "labelZh": "地址",
      "labelEn": "Address",
      "regex": "^.+$",
      "sort": 2,
      "type": "input",
      "required": true
    },
    {
      "name": "concurrency",
      "labelZh": "并发数",
      "labelEn": "Concurrency",
      "regex": "^[1-9][0-9]*$",
      "sort": 3,
      "type": "input"
    },
    {
      "name": "timeoutMs",
      "labelZh": "超时(ms)",
      "labelEn": "Timeout (ms)",
      "regex": "^[1-9][0-9]*$",
      "sort": 4,
      "type": "input"
    }
  ],
  "variableAddress": {
    "labelZh": "变量地址",
    "labelEn": "Variable Address",
    "regex": "^.+$",
    "placeholderZh": "按插件协议填写",
    "placeholderEn": "Depends on the plugin protocol"
  },
  "variableDataTypes": [
    {"value":"INT16","labelZh":"int16","labelEn":"int16","addressRegex":"^.+$"},
    {"value":"UINT16","labelZh":"uint16","labelEn":"uint16","addressRegex":"^.+$"},
    {"value":"INT32","labelZh":"int32","labelEn":"int32","addressRegex":"^.+$"},
    {"value":"UINT32","labelZh":"uint32","labelEn":"uint32","addressRegex":"^.+$"},
    {"value":"FLOAT32","labelZh":"float32","labelEn":"float32","addressRegex":"^.+$"},
    {"value":"FLOAT64","labelZh":"float64","labelEn":"float64","addressRegex":"^.+$"},
    {"value":"BOOL","labelZh":"bool","labelEn":"bool","addressRegex":"^.+$"},
    {"value":"STRING","labelZh":"string","labelEn":"string","addressRegex":"^.+$"}
  ]
}`

const templatePlugin = `package main

import "harns-platform/pkg/gateway/pluginmgr"

// GatewayDriver is the exported plugin symbol.
func GatewayDriver() pluginmgr.Driver {
	return &driver{}
}
`

const templateDriverTest = `package main

import "testing"

func TestDriverDescriptor(t *testing.T) {
	driver := GatewayDriver()
	if driver == nil {
		t.Fatal("driver is nil")
	}
	descriptor := driver.Descriptor()
	if descriptor.ABIVersion == "" {
		t.Fatal("ABI version is empty")
	}
}
`

const templateDriver = `package main

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"

	"harns-platform/pkg/gateway/collection"
	"harns-platform/pkg/gateway/pluginmgr"
)

type protocolConfig struct {
	Transport   string ` + "`json:\"transport\"`" + `
	Location    string ` + "`json:\"location\"`" + `
	Concurrency int    ` + "`json:\"concurrency\"`" + `
	TimeoutMs   int    ` + "`json:\"timeoutMs\"`" + `
}

type driver struct {
	config protocolConfig
}

func (d *driver) Descriptor() pluginmgr.Descriptor {
	return pluginmgr.Descriptor{
		DriverType:  "{{.DriverType}}",
		DriverModel: "{{.DriverModel}}",
		PluginName:  "{{.PluginName}}",
		Version:     "1.0.0",
		ABIVersion:  pluginmgr.ABIVersion,
		GOOS:        runtime.GOOS,
		GOARCH:      runtime.GOARCH,
		GoVersion:   runtime.Version(),
	}
}

func (d *driver) ValidateConfig(raw json.RawMessage) error {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("invalid protocol config: %w", err)
	}
	if config.Location == "" {
		return fmt.Errorf("location is required")
	}
	switch config.Transport {
	case "tcp", "udp", "serial":
	default:
		return fmt.Errorf("transport must be tcp, udp or serial")
	}
	if config.Concurrency < 0 {
		return fmt.Errorf("concurrency must be greater than 0")
	}
	d.config = config
	return nil
}

func (d *driver) TransportKind(raw json.RawMessage) (pluginmgr.TransportKind, error) {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return "", err
	}
	switch config.Transport {
	case "tcp":
		return pluginmgr.TransportTCP, nil
	case "udp":
		return pluginmgr.TransportUDP, nil
	case "serial":
		return pluginmgr.TransportSerial, nil
	default:
		return "", fmt.Errorf("transport must be tcp, udp or serial")
	}
}

func (d *driver) NewTransport(ctx context.Context, raw json.RawMessage) (pluginmgr.Transport, error) {
	var config protocolConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	// TODO: open the TCP/UDP/serial connection and implement:
	//   Exchange(ctx context.Context, request []byte) ([]byte, error)
	//   Available() bool
	//   Close() error
	return nil, fmt.Errorf("NewTransport is not implemented")
}

func (d *driver) BuildReadFrames(variables []*collection.Variable, raw json.RawMessage) ([]pluginmgr.ReadFrame, error) {
	// TODO: validate/group addresses, build request bytes, and keep the
	// variables that belong to each frame.
	return nil, fmt.Errorf("BuildReadFrames is not implemented")
}

func (d *driver) DecodeRead(frame pluginmgr.ReadFrame, response []byte) ([]pluginmgr.DecodedValue, error) {
	// TODO: parse the response and return one value per variable.
	// Return successful values together with the first decode error.
	return nil, fmt.Errorf("DecodeRead is not implemented")
}

func (d *driver) BuildWrite(action collection.ActionValue, raw json.RawMessage) ([]byte, error) {
	// TODO: encode one write request.
	return nil, fmt.Errorf("BuildWrite is not implemented")
}

func (d *driver) DecodeWrite(response []byte) error {
	// TODO: validate the write response.
	return fmt.Errorf("DecodeWrite is not implemented")
}
`
