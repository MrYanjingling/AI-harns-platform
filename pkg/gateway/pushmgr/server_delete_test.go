package pushmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/pushmgr/v1"
	"harns-platform/pkg/i18n"
)

func TestDeleteConnectorConflictUsesConnectorName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := newPushTestService(t)
	ctx := t.Context()

	connector, err := service.CreateConnector(ctx, &v1.CreateConnectorReq{
		Name: "remote",
		PushConfig: json.RawMessage(`{
			"brokerUrl":"tcp://127.0.0.1:1883",
			"dataTopic":"data"
		}`),
	})
	require.NoError(t, err)
	_, err = service.CreateGroup(ctx, &v1.CreatePushGroupReq{
		Name:           "group",
		ConnectorID:    connector.ID,
		DataTopic:      "data/group",
		AllCollections: true,
	})
	require.NoError(t, err)

	engine := gin.New()
	engine.Use(i18n.Middleware())
	InstallHandler(engine.Group("/api/v1"), service)
	req := httptest.NewRequest(http.MethodDelete,
		"/api/v1/push/connectors/"+connector.ID+"?lang=zh-CN", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	var result response.Result
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, int(response.ErrCodePushConnectorConflict), result.Code)
	assert.Equal(t, "推送连接器 [remote] 仍被推送组引用。", result.Message)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, result.Message, result.Errors[0].Message)
}
