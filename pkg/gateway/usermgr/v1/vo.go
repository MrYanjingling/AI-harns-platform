package v1

import "time"

type UserVo struct {
	ID          string    `json:"id"`
	Version     int64     `json:"version"`
	Username    string    `json:"username"`
	Remark      string    `json:"remark"`
	IsAdmin     bool      `json:"isAdmin"`
	CreatedTime time.Time `json:"createdTime"`
	UpdatedTime time.Time `json:"updatedTime"`
}

type LoginVo struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      *UserVo   `json:"user"`
}
