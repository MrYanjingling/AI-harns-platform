package v1

// SetEnableReq toggles collection auto-start eligibility.
type SetEnableReq struct {
	Enable bool `json:"enable"`
}

// EnableVo is the collection enable state.
type EnableVo struct {
	Enable bool `json:"enable"`
}
