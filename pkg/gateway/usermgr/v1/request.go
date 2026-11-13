package v1

import (
	"harns-platform/pkg/apis"
)

type LoginReq struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"required,max=64"`
}

type CreateUserReq struct {
	Username string `json:"username" binding:"required,max=64"`
	Password string `json:"password" binding:"omitempty,min=6,max=64"`
	Remark   string `json:"remark" binding:"omitempty,max=255"`
}

type UpdateUserReq struct {
	Username *string `json:"username" binding:"omitempty,max=64"`
	Remark   *string `json:"remark" binding:"omitempty,max=255"`
}

type ChangePasswordReq struct {
	OldPassword string `json:"oldPassword" binding:"required,max=64"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=64"`
}

type UserQuery struct {
	apis.PageQuery
	Keyword string `json:"keyword" form:"keyword" binding:"omitempty,max=64"`
}
