package v1

// Device authentication results understood by EMQX.
const (
	AuthResultAllow = "allow"
	AuthResultDeny  = "deny"
)

// ACL rule constants understood by EMQX.
const (
	ACLPermissionAllow = "allow"
	ACLActionAll       = "all"
)

// DeviceAuthReq is the EMQX HTTP authenticator request body.
type DeviceAuthReq struct {
	ClientID    string                 `json:"clientid"`
	Username    string                 `json:"username"`
	Password    string                 `json:"password"`
	ClientAttrs map[string]interface{} `json:"client_attrs,omitempty"`
	CN          string                 `json:"cn,omitempty"`
}

// DeviceAuthResp is the EMQX HTTP authenticator response body.
type DeviceAuthResp struct {
	Result      string            `json:"result"`
	Superuser   bool              `json:"is_superuser"`
	ClientAttrs map[string]string `json:"client_attrs,omitempty"`
	ExpireAt    *int64            `json:"expire_at,omitempty"`
	ACL         []ACLRule         `json:"acl,omitempty"`
}

// ACLRule is a single EMQX ACL entry.
type ACLRule struct {
	Permission string `json:"permission"`
	Action     string `json:"action"`
	Topic      string `json:"topic"`
	QoS        []int  `json:"qos"`
	Retain     bool   `json:"retain"`
}
