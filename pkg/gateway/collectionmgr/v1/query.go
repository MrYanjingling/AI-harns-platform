package v1

import "harns-platform/pkg/apis"

// CollectionQuery is the paged collection query.
type CollectionQuery struct {
	apis.PageQuery
	Keyword     string `form:"keyword" json:"keyword" binding:"omitempty,max=64"`
	DriverType  string `form:"driverType" json:"driverType" binding:"omitempty,max=64"`
	DriverModel string `form:"driverModel" json:"driverModel" binding:"omitempty,max=64"`
}

// VariableQuery is the paged collection variable query.
type VariableQuery struct {
	apis.PageQuery
	Keyword string `form:"keyword" json:"keyword" binding:"omitempty,max=64"`
}
