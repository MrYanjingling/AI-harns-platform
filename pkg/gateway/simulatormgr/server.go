package simulatormgr

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/simulatormgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/i18n"
)

func InstallHandler(group *gin.RouterGroup, svc *Service) {
	group.GET("/simulator", getState(svc))
	group.POST("/simulator/open", open(svc))
	group.POST("/simulator/close", closeSimulator(svc))
	group.POST("/simulator/points", createPoint(svc))
	group.DELETE("/simulator/points/:address", deletePoint(svc))
}

func getState(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		response.WriteOK(c, svc.State())
	}
}

func open(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.OpenReq
		if c.Request.ContentLength != 0 {
			if err := c.ShouldBindJSON(&req); err != nil {
				writeSimulatorErr(c, err)
				return
			}
		}
		state, err := svc.Open(c.Request.Context(), req.Port)
		if err != nil {
			writeSimulatorErr(c, err)
			return
		}
		response.WriteOK(c, state)
	}
}

func closeSimulator(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		state, err := svc.Close(c.Request.Context())
		if err != nil {
			writeSimulatorErr(c, err)
			return
		}
		response.WriteOK(c, state)
	}
}

func createPoint(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req v1.CreatePointReq
		if err := c.ShouldBindJSON(&req); err != nil {
			writeSimulatorErr(c, err)
			return
		}
		point, err := svc.AddPoint(req.Address, req.Name, req.Description, req.DataType, Mode(req.Mode))
		if err != nil {
			writeSimulatorErr(c, err)
			return
		}
		response.WriteOK(c, point)
	}
}

func deletePoint(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		address, err := strconv.ParseUint(c.Param("address"), 10, 16)
		if err != nil || address == 0 {
			writeSimulatorErr(c, validation.New([]validation.FieldIssue{{
				Field: "address", MsgID: "validate.simulator_address_invalid",
			}}))
			return
		}
		if err := svc.DeletePoint(uint16(address)); err != nil {
			writeSimulatorErr(c, err)
			return
		}
		response.WriteOK(c, gin.H{"deleted": true})
	}
}

func writeSimulatorErr(c *gin.Context, err error) {
	var validationErr *validation.ValidationError
	if errors.As(err, &validationErr) {
		items := make([]response.ErrorItem, 0, len(validationErr.Issues))
		for _, issue := range validationErr.Issues {
			items = append(items, response.NewFieldItem(
				i18n.Lang(c), response.ErrCodeValidationFailed, issue.Field, issue.MsgID, issue.Args...))
		}
		response.WriteFieldErrors(c, http.StatusBadRequest, response.ErrCodeValidationFailed, items)
		return
	}
	response.WriteError(c, http.StatusBadRequest, response.ErrInternal(err))
}
