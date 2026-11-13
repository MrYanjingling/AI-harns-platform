package usermgr

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/oklog/ulid/v2"
	"golang.org/x/crypto/bcrypt"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/usermgr/v1"
	"harns-platform/pkg/gateway/validation"
	"harns-platform/pkg/logging"
)

type Service struct {
	store       *Store
	sessions    *sessionStore
	authEnabled bool
}

func NewService(store *Store, authEnabled bool) *Service {
	return &Service{
		store:       store,
		sessions:    newSessionStore(),
		authEnabled: authEnabled,
	}
}

func (s *Service) BootstrapSuperuser(ctx context.Context) error {
	defer logging.Trace("usermgr.BootstrapSuperuser")()

	has, err := s.store.HasSuperuser(ctx)
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	existing, err := s.store.GetByUsername(ctx, SuperuserName)
	if err == nil {
		if existing.IsAdmin {
			return nil
		}
		return ErrSuperuserExists
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(SuperuserPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	return s.store.Create(ctx, &User{
		ID:           ulid.Make().String(),
		Username:     SuperuserName,
		PasswordHash: string(hash),
		Remark:       SuperuserRemark,
		IsAdmin:      true,
		Version:      1,
		CreatedTime:  now,
		UpdatedTime:  now,
	})
}

func (s *Service) Login(ctx context.Context, username, password string) (*v1.LoginVo, error) {
	defer logging.Trace("usermgr.Login", "username", username)()

	username = strings.TrimSpace(username)
	user, err := s.store.GetByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrUsernamePasswordError()
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, response.ErrUsernamePasswordError()
	}

	token, err := newToken()
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	now := time.Now()
	session := &Session{
		Token:             token,
		UserID:            user.ID,
		Username:          user.Username,
		IsAdmin:           user.IsAdmin,
		CreatedAt:         now,
		ExpiresAt:         now.Add(DefaultSessionTTL),
		AbsoluteExpiresAt: now.Add(DefaultAbsoluteTTL),
	}
	s.sessions.Add(session)
	return &v1.LoginVo{
		Token:     token,
		ExpiresAt: session.ExpiresAt,
		User:      toVo(user),
	}, nil
}

func (s *Service) Logout(token string) {
	s.sessions.Delete(token)
}

func (s *Service) Authenticate(token string) (*Session, error) {
	session, ok := s.sessions.GetAndTouch(token, time.Now())
	if !ok {
		return nil, response.ErrLoginExpired()
	}
	return session, nil
}

func (s *Service) Create(ctx context.Context, req *v1.CreateUserReq) (*v1.UserVo, error) {
	defer logging.Trace("usermgr.Create", "username", req.Username)()

	username := strings.TrimSpace(req.Username)
	remark := strings.TrimSpace(req.Remark)
	password := req.Password
	if password == "" {
		if err := validateUser(username, "", remark, false); err != nil {
			return nil, err
		}
		password = defaultPassword(username)
	} else if err := validateUser(username, password, remark, true); err != nil {
		return nil, err
	}
	exists, err := s.store.UsernameExists(ctx, username, "")
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if exists {
		return nil, response.ErrUserNameExists(username)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	now := time.Now().UTC()
	user := &User{
		ID:           ulid.Make().String(),
		Username:     username,
		PasswordHash: string(hash),
		Remark:       remark,
		Version:      1,
		CreatedTime:  now,
		UpdatedTime:  now,
	}
	if err := s.store.Create(ctx, user); err != nil {
		if isUniqueError(err) {
			return nil, response.ErrUserNameExists(username)
		}
		return nil, response.ErrInternal(err)
	}
	return toVo(user), nil
}

func (s *Service) List(ctx context.Context, q apis.PageQuery, keyword string) (*apis.PageResp[*v1.UserVo], error) {
	defer logging.Trace("usermgr.List", "page", q.Page, "pageSize", q.PageSize, "keyword", keyword)()

	q.Normalize()
	items, total, err := s.store.List(ctx, q, keyword)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	list := make([]*v1.UserVo, 0, len(items))
	for _, item := range items {
		list = append(list, toVo(item))
	}
	return apis.NewPageResp(list, total, q), nil
}

func (s *Service) Get(ctx context.Context, id string) (*v1.UserVo, error) {
	defer logging.Trace("usermgr.Get", "id", id)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrUserNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	return toVo(item), nil
}

func (s *Service) Update(ctx context.Context, id string, req *v1.UpdateUserReq) (*v1.UserVo, error) {
	defer logging.Trace("usermgr.Update", "id", id)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrUserNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}

	username := item.Username
	remark := item.Remark
	if req.Username != nil {
		username = strings.TrimSpace(*req.Username)
	}
	if req.Remark != nil {
		remark = strings.TrimSpace(*req.Remark)
	}
	if item.IsAdmin && username != item.Username {
		return nil, response.ErrUserProtected()
	}
	if err := validateUser(username, "", remark, false); err != nil {
		return nil, err
	}
	exists, err := s.store.UsernameExists(ctx, username, id)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if exists {
		return nil, response.ErrUserNameExists(username)
	}

	item.Username = username
	item.Remark = remark
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.Update(ctx, item); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, response.ErrUserNotFound(id)
		}
		if isUniqueError(err) {
			return nil, response.ErrUserNameExists(username)
		}
		return nil, response.ErrInternal(err)
	}
	item.Version++
	return toVo(item), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	defer logging.Trace("usermgr.Delete", "id", id)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return response.ErrUserNotFound(id)
	}
	if err != nil {
		return response.ErrInternal(err)
	}
	if item.IsAdmin {
		return response.ErrUserProtected()
	}
	if err := s.store.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return response.ErrUserNotFound(id)
		}
		return response.ErrInternal(err)
	}
	s.sessions.DeleteByUser(id)
	return nil
}

func (s *Service) ChangePassword(ctx context.Context, id, actorID, oldPassword, newPassword string) (*v1.UserVo, error) {
	defer logging.Trace("usermgr.ChangePassword", "id", id)()

	if s.authEnabled && id != actorID {
		return nil, response.ErrForbidden()
	}
	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrUserNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(item.PasswordHash), []byte(oldPassword)) != nil {
		return nil, response.ErrUserPasswordMismatch()
	}
	if err := validatePassword(newPassword); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdatePassword(ctx, id, string(hash), item.UpdatedTime); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, response.ErrUserNotFound(id)
		}
		return nil, response.ErrInternal(err)
	}
	s.sessions.DeleteByUser(id)
	item.Version++
	return toVo(item), nil
}

func (s *Service) ResetPassword(ctx context.Context, id string) (*v1.UserVo, error) {
	defer logging.Trace("usermgr.ResetPassword", "id", id)()

	item, err := s.store.Get(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, response.ErrUserNotFound(id)
	}
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(defaultPassword(item.Username)), bcrypt.DefaultCost)
	if err != nil {
		return nil, response.ErrInternal(err)
	}
	item.UpdatedTime = time.Now().UTC()
	if err := s.store.UpdatePassword(ctx, id, string(hash), item.UpdatedTime); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, response.ErrUserNotFound(id)
		}
		return nil, response.ErrInternal(err)
	}
	s.sessions.DeleteByUser(id)
	item.Version++
	return toVo(item), nil
}

func toVo(item *User) *v1.UserVo {
	if item == nil {
		return nil
	}
	return &v1.UserVo{
		ID:          item.ID,
		Version:     item.Version,
		Username:    item.Username,
		Remark:      item.Remark,
		IsAdmin:     item.IsAdmin,
		CreatedTime: item.CreatedTime,
		UpdatedTime: item.UpdatedTime,
	}
}

func validateUser(username, password, remark string, passwordRequired bool) error {
	issues := make([]validation.FieldIssue, 0, 3)
	switch {
	case username == "":
		issues = append(issues, validation.FieldIssue{Field: "username", MsgID: "validate.user_name_required"})
	case len(username) > usernameMaxLength:
		issues = append(issues, validation.FieldIssue{Field: "username", MsgID: "validate.user_name_invalid"})
	case strings.IndexFunc(username, unicode.IsSpace) >= 0:
		issues = append(issues, validation.FieldIssue{Field: "username", MsgID: "validate.user_name_invalid"})
	}
	if passwordRequired && password == "" {
		issues = append(issues, validation.FieldIssue{Field: "password", MsgID: "validate.user_password_required"})
	} else if password != "" && (len(password) < passwordMinLength || len(password) > passwordMaxLength) {
		issues = append(issues, validation.FieldIssue{Field: "password", MsgID: "validate.user_password_invalid"})
	}
	if utf8.RuneCountInString(remark) > remarkMaxLength {
		issues = append(issues, validation.FieldIssue{Field: "remark", MsgID: "validate.user_remark_invalid"})
	}
	if err := validation.New(issues); err != nil {
		return err
	}
	return nil
}

func validatePassword(password string) error {
	if len(password) < passwordMinLength || len(password) > passwordMaxLength {
		return validation.New([]validation.FieldIssue{{
			Field: "newPassword",
			MsgID: "validate.user_password_invalid",
		}})
	}
	return nil
}

func isUniqueError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed: users.username")
}
