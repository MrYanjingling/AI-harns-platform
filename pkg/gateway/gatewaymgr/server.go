package gatewaymgr

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

func InstallHandler(group *gin.RouterGroup, mgr *Manager) {
	group.GET("/gatewayMeta", getGatewayMeta(mgr))
	group.PUT("/gatewayMeta", updateGatewayMeta(mgr))
	group.GET("/systemInfo", getSystemInfo(mgr))
	group.GET("/gatewayCpu", getGatewayCpu(mgr))
	group.GET("/gatewayMem", getGatewayMem(mgr))
	group.GET("/gatewayDisk", getGatewayDisk(mgr))
}

func getGatewayMeta(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		g, err := mgr.GetGatewayMetaVo()
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
			return
		}
		c.Header(apis.ETag, fmt.Sprintf("%s", g.GetVersion()))
		response.WriteOK(c, g)
	}
}

func updateGatewayMeta(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.UpdateGatewayMetaReq
		if err := c.ShouldBindJSON(&req); err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrMalformedJSON)
			return
		}
		g, err := mgr.UpdateGatewayName(c.Request.Context(), req.Name)
		if err != nil {
			writeGatewayMetaErr(c, err)
			return
		}
		c.Header(apis.ETag, fmt.Sprintf("%s", g.GetVersion()))
		response.WriteOK(c, g)
	}
}

func writeGatewayMetaErr(c *gin.Context, err error) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(
				i18n.Lang(c), response.ErrCodeGatewayMetaInvalid, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeGatewayMetaInvalid, items)
		return
	}
	response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
}

func getSystemInfo(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.WriteOK(c, mgr.GetSystemInfoMeta(c.Request.Context()))
	}
}

func getGatewayCpu(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		cpu, err := mgr.getGatewayCpu()
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
			return
		}
		response.WriteOK(c, ResponseModel{Cpus: cpu})
	}
}

func getGatewayMem(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		mem, err := mgr.getGatewayMem()
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
			return
		}
		response.WriteOK(c, ResponseModel{Mem: mem})
	}
}

func getGatewayDisk(mgr *Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		disks, err := mgr.getGatewayDisk()
		if err != nil {
			response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
			return
		}
		response.WriteOK(c, ResponseModel{Disks: disks})
	}
}
