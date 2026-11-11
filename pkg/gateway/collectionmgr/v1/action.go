package v1

// ActionItem is one collection write action.
type ActionItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ActionReq is the collection action request.
type ActionReq struct {
	Actions []ActionItem `json:"actions"`
}

// ActionResp is the collection action result.
type ActionResp struct {
	Accepted int `json:"accepted"`
}
