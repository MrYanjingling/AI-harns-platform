package usermgr

import (
	"errors"
	"time"
)

const (
	SuperuserName         = "admin"
	SuperuserPassword     = "Harns@123"
	SuperuserRemark       = "超级管理员"
	DefaultPasswordSuffix = "@gateway"
	DefaultSessionTTL     = 24 * time.Hour
	DefaultAbsoluteTTL    = 7 * 24 * time.Hour
	sessionContextKey     = "usermgr.session"
	usernameMaxLength     = 64
	passwordMinLength     = 6
	passwordMaxLength     = 64
	remarkMaxLength       = 255
	tokenLength           = 32
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSuperuserExists = errors.New("admin username is occupied by a non-superuser")
)

type User struct {
	ID           string
	Username     string
	PasswordHash string
	Remark       string
	IsAdmin      bool
	Version      int64
	CreatedTime  time.Time
	UpdatedTime  time.Time
}

type Session struct {
	Token             string
	UserID            string
	Username          string
	IsAdmin           bool
	CreatedAt         time.Time
	ExpiresAt         time.Time
	AbsoluteExpiresAt time.Time
}

func defaultPassword(username string) string {
	return username + DefaultPasswordSuffix
}
