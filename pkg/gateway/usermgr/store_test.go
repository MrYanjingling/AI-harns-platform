package usermgr

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"harns-platform/pkg/apis"
	"harns-platform/pkg/apis/response"
	"harns-platform/pkg/gateway/sqlite"
	"harns-platform/pkg/gateway/usermgr/v1"
)

func newTestService(t *testing.T) (*Service, *Store) {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	require.NoError(t, store.Initialize(context.Background()))
	return NewService(store), store
}

func newBootstrappedService(t *testing.T) (*Service, *Store) {
	t.Helper()
	service, store := newTestService(t)
	require.NoError(t, service.BootstrapSuperuser(context.Background()))
	return service, store
}

func TestStoreUserCRUD(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	created, err := service.Create(ctx, createReq("operator", "secret1", "line one"))
	require.NoError(t, err)
	assert.False(t, created.IsAdmin)

	list, err := service.List(ctx, apis.PageQuery{}, "line")
	require.NoError(t, err)
	assert.Equal(t, int64(1), list.Total)
	require.Len(t, list.List, 1)

	remark := "line two"
	updated, err := service.Update(ctx, created.ID, &v1.UpdateUserReq{Remark: &remark})
	require.NoError(t, err)
	assert.Equal(t, remark, updated.Remark)
	assert.Equal(t, int64(2), updated.Version)

	require.NoError(t, service.Delete(ctx, created.ID))
	_, err = service.Get(ctx, created.ID)
	require.Error(t, err)
}

func TestStoreListKeywordAndPagination(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	for _, name := range []string{"alice", "bob", "carol"} {
		_, err := service.Create(ctx, createReq(name, "secret1", "shift"))
		require.NoError(t, err)
	}
	page, err := service.List(ctx, apis.PageQuery{Page: 1, PageSize: 2}, "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), page.Total)
	assert.Len(t, page.List, 2)

	page, err = service.List(ctx, apis.PageQuery{}, "car")
	require.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	require.Len(t, page.List, 1)
	assert.Equal(t, "carol", page.List[0].Username)
}

func TestBootstrapSuperuserIsIdempotent(t *testing.T) {
	service, store := newTestService(t)
	ctx := context.Background()
	require.NoError(t, service.BootstrapSuperuser(ctx))
	first, err := store.GetByUsername(ctx, SuperuserName)
	require.NoError(t, err)
	assert.True(t, first.IsAdmin)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(first.PasswordHash), []byte(SuperuserPassword)))

	require.NoError(t, service.BootstrapSuperuser(ctx))
	second, err := store.GetByUsername(ctx, SuperuserName)
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, first.PasswordHash, second.PasswordHash)
}

func TestCreateWithoutPasswordUsesDefault(t *testing.T) {
	service, store := newTestService(t)
	created, err := service.Create(context.Background(), &v1.CreateUserReq{Username: "operator"})
	require.NoError(t, err)
	stored, err := store.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("operator@gateway")))
}

func TestCreateRejectsDuplicateAndInvalidFields(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	_, err := service.Create(ctx, createReq("same", "secret1", ""))
	require.NoError(t, err)
	_, err = service.Create(ctx, createReq("same", "secret1", ""))
	assert.Equal(t, int(response.ErrCodeUserNameExists), response.ErrorCode(err))

	_, err = service.Create(ctx, &v1.CreateUserReq{Username: "bad user", Password: "secret1"})
	require.Error(t, err)
}

func TestAdminCannotBeDeletedOrRenamed(t *testing.T) {
	service, store := newBootstrappedService(t)
	admin, err := store.GetByUsername(context.Background(), SuperuserName)
	require.NoError(t, err)
	err = service.Delete(context.Background(), admin.ID)
	assert.Equal(t, int(response.ErrCodeUserProtected), response.ErrorCode(err))

	name := "root"
	_, err = service.Update(context.Background(), admin.ID, &v1.UpdateUserReq{Username: &name})
	assert.Equal(t, int(response.ErrCodeUserProtected), response.ErrorCode(err))
}

func TestChangePasswordRequiresOldAndInvalidatesSessions(t *testing.T) {
	service, store := newBootstrappedService(t)
	admin, err := store.GetByUsername(context.Background(), SuperuserName)
	require.NoError(t, err)
	login, err := service.Login(context.Background(), SuperuserName, SuperuserPassword)
	require.NoError(t, err)
	_, err = service.Authenticate(login.Token)
	require.NoError(t, err)

	_, err = service.ChangePassword(context.Background(), admin.ID, admin.ID, "wrong", "NewPass1")
	assert.Equal(t, int(response.ErrCodeUserPasswordMismatch), response.ErrorCode(err))

	_, err = service.ChangePassword(context.Background(), admin.ID, "other", SuperuserPassword, "NewPass1")
	assert.Equal(t, int(response.ErrCodeForbidden), response.ErrorCode(err))

	_, err = service.ChangePassword(context.Background(), admin.ID, admin.ID, SuperuserPassword, "NewPass1")
	require.NoError(t, err)
	_, err = service.Authenticate(login.Token)
	require.Error(t, err)
}

func TestResetPasswordUsesUsernameAndInvalidatesSessions(t *testing.T) {
	service, store := newBootstrappedService(t)
	created, err := service.Create(context.Background(), createReq("operator", "secret1", ""))
	require.NoError(t, err)
	login, err := service.Login(context.Background(), "operator", "secret1")
	require.NoError(t, err)
	_, err = service.Authenticate(login.Token)
	require.NoError(t, err)

	_, err = service.ResetPassword(context.Background(), created.ID)
	require.NoError(t, err)
	stored, err := store.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("operator@gateway")))
	_, err = service.Authenticate(login.Token)
	require.Error(t, err)
}

func TestSessionStoreSlidesAndExpires(t *testing.T) {
	store := newSessionStore()
	now := time.Now()
	session := &Session{
		Token:             "token",
		UserID:            "user",
		ExpiresAt:         now.Add(DefaultSessionTTL),
		AbsoluteExpiresAt: now.Add(DefaultAbsoluteTTL),
	}
	store.Add(session)

	touched, ok := store.GetAndTouch("token", now.Add(time.Hour))
	require.True(t, ok)
	assert.Equal(t, now.Add(time.Hour+DefaultSessionTTL), touched.ExpiresAt)

	store.Add(&Session{
		Token:             "cap",
		ExpiresAt:         now.Add(DefaultAbsoluteTTL),
		AbsoluteExpiresAt: now.Add(DefaultAbsoluteTTL),
	})
	touched, ok = store.GetAndTouch("cap", now.Add(6*24*time.Hour))
	require.True(t, ok)
	assert.Equal(t, now.Add(DefaultAbsoluteTTL), touched.ExpiresAt)

	_, ok = store.GetAndTouch("cap", now.Add(DefaultAbsoluteTTL+time.Second))
	assert.False(t, ok)

	store.Add(&Session{
		Token:             "idle",
		ExpiresAt:         now.Add(DefaultSessionTTL),
		AbsoluteExpiresAt: now.Add(DefaultAbsoluteTTL),
	})
	_, ok = store.GetAndTouch("idle", now.Add(DefaultSessionTTL+time.Second))
	assert.False(t, ok)
}

func createReq(username, password, remark string) *v1.CreateUserReq {
	return &v1.CreateUserReq{Username: username, Password: password, Remark: remark}
}
