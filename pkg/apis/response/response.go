package response

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"harns-platform/pkg/i18n"
)

type responseError struct {
	Code    ErrCode       `json:"code"`
	Message string        `json:"message"`
	Err     error         `json:"-"`
	args    []interface{} `json:"-"`
	data    interface{}   `json:"-"`
}

func (re *responseError) Error() string {
	if re == nil {
		return ""
	}
	s := `{
    "code": ` + strconv.Itoa(int(re.Code)) + `,
    "message": ` + re.Message + `
}`
	return s
}

func (re *responseError) GetCode() ErrCode {
	if re == nil {
		return 0
	}
	return re.Code
}

func (re *responseError) Unwrap() error {
	return re.Err
}

// Localized returns a copy of the error whose message is translated into lang.
// It keeps the original arguments so the reported code stays the same.
func (re *responseError) Localized(lang string) *responseError {
	if re == nil {
		return nil
	}
	return &responseError{
		Code:    re.Code,
		Message: i18n.Translate(lang, messageID(re.Code), re.args...),
		Err:     re.Err,
		args:    re.args,
		data:    re.data,
	}
}

func IsResponseError(err error) bool {
	_, ok := err.(*responseError)
	return ok
}

// MultiError contains multiple errors and implements the error interface. Its
// zero value is ready to use. All its methods are goroutine safe.
type MultiError struct {
	mtx    sync.Mutex
	errors []error
}

func NewMultiError(err ...error) *MultiError {
	return &MultiError{
		errors: err,
	}
}

// Add adds an error to the MultiError.
func (e *MultiError) Add(err ...error) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.errors = append(e.errors, err...)
}

// Len returns the number of errors added to the MultiError.
func (e *MultiError) Len() int {
	if e == nil {
		return 0
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()

	return len(e.errors)
}

// MultiError returns the errors added to the MuliError. The returned slice is a
// copy of the internal slice of errors.
func (e *MultiError) Errors() []error {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	return append(make([]error, 0, len(e.errors)), e.errors...)
}

func (e *MultiError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Errors []error `json:"errors"`
	}{
		Errors: e.errors,
	})
}

func (e *MultiError) UnmarshalJSON(bytes []byte) error {
	errs := struct {
		Errors []*responseError `json:"errors"`
	}{}
	if err := json.Unmarshal(bytes, &errs); err != nil {
		return err
	}
	for _, err := range errs.Errors {
		e.Add(err)
	}
	return nil
}

func (e *MultiError) Error() string {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	es := make([]string, 0, len(e.errors))
	for _, err := range e.errors {
		es = append(es, err.Error())
	}
	return strings.Join(es, "; ")
}

func generateError(code ErrCode, s ...interface{}) *responseError {
	return &responseError{
		Code:    code,
		Message: defaultMessage(code, s...),
		args:    s,
	}
}

func generateErrorWrapper(code ErrCode, err error, s ...interface{}) *responseError {
	re := generateError(code, s...)
	re.Err = err
	return re
}

// defaultMessage formats the message of code in the default language.
func defaultMessage(code ErrCode, s ...interface{}) string {
	return i18n.Translate(i18n.DefaultLang(), messageID(code), s...)
}

// https://golang.org/doc/faq#convert_slice_of_interface
func convert(infos []string) []interface{} {
	s := make([]interface{}, len(infos))
	for i, v := range infos {
		s[i] = v
	}
	return s
}

func ErrResourceExists(resource string) *responseError {
	return generateError(ErrCodeResourceExists, resource)
}
func ErrResourceNotFound(resource string) *responseError {
	return generateError(ErrCodeResourceNotFound, resource)
}
func ErrDeviceNotFound(resource string) *responseError {
	return generateError(ErrCodeDeviceNotFound, resource)
}
func ErrDeviceNotConnect(resource string) *responseError {
	return generateError(ErrCodeDeviceNotConnect, resource)
}
func ErrDeviceOperatorUnSupported(operator string) *responseError {
	return generateError(ErrCodeDeviceOperatorUnSupported, operator)
}

func ErrTooManyJsonPatchOperations(max int) *responseError {
	return generateError(ErrCodeTooManyJsonPatchOperations, max)
}

// ErrInternal wraps an unexpected internal error with a generic code.
func ErrInternal(err error) *responseError {
	return generateErrorWrapper(ErrCodeInternal, err)
}

// ErrUserNotFound reports that a user does not exist.
func ErrUserNotFound(id string) *responseError {
	return generateError(ErrCodeUserNotFound, id)
}

// ErrUserNameExists reports that a user name is already taken.
func ErrUserNameExists(name string) *responseError {
	return generateError(ErrCodeUserNameExists, name)
}

// ErrUserNoExists reports that a user number is already taken.
func ErrUserNoExists(userNo string) *responseError {
	return generateError(ErrCodeUserNoExists, userNo)
}

// ErrVersionMismatch reports an optimistic-lock conflict (If-Match failed).
func ErrVersionMismatch() *responseError {
	return generateError(ErrCodeVersionMismatch)
}

// ErrRoleNotFound reports that a role does not exist.
func ErrRoleNotFound(id string) *responseError {
	return generateError(ErrCodeRoleNotFound, id)
}

// ErrRoleNameExists reports that a role name is already taken.
func ErrRoleNameExists(name string) *responseError {
	return generateError(ErrCodeRoleNameExists, name)
}

// ErrRoleCodeExists reports that a role code is already taken.
func ErrRoleCodeExists(code string) *responseError {
	return generateError(ErrCodeRoleCodeExists, code)
}

// ErrOrgNotFound reports that an organization does not exist.
func ErrOrgNotFound(id string) *responseError {
	return generateError(ErrCodeOrgNotFound, id)
}

// ErrOrgCodeExists reports that an organization code is already taken.
func ErrOrgCodeExists(code string) *responseError {
	return generateError(ErrCodeOrgCodeExists, code)
}

// ErrOrgHasChildren reports that an organization still has sub organizations.
func ErrOrgHasChildren(id string) *responseError {
	return generateError(ErrCodeOrgHasChildren, id)
}

// ErrOrgParentNotFound reports that the specified parent organization does not exist.
func ErrOrgParentNotFound(id string) *responseError {
	return generateError(ErrCodeOrgParentNotFound, id)
}

// ErrOrgParentCycle reports an invalid parent that would create a cycle.
func ErrOrgParentCycle(id string) *responseError {
	return generateError(ErrCodeOrgParentCycle, id)
}

// ErrLoginExpired reports a missing or expired login session.
func ErrLoginExpired() *responseError {
	return generateError(ErrCodeLoginExpired)
}

// ErrUnauthorized reports an unauthenticated request.
func ErrUnauthorized() *responseError {
	return generateError(ErrCodeUnauthorized)
}

// ErrUsernamePasswordError reports wrong credentials.
func ErrUsernamePasswordError() *responseError {
	return generateError(ErrCodeUsernamePasswordError)
}

// ErrAccountDisabled reports a disabled account.
func ErrAccountDisabled() *responseError {
	return generateError(ErrCodeAccountDisabled)
}

// ErrAccountLocked reports a locked account and exposes the unlock timestamp
// (Unix milliseconds) to clients.
func ErrAccountLocked(unlockAt time.Time) *responseError {
	re := generateError(ErrCodeAccountLocked)
	re.data = map[string]int64{"unlockAt": unlockAt.UnixMilli()}
	return re
}

// ErrLoginFailed wraps an unexpected login error.
func ErrLoginFailed(err error) *responseError {
	return generateErrorWrapper(ErrCodeLoginFailed, err)
}

// ErrCaptchaEmpty reports a missing captcha key.
func ErrCaptchaEmpty() *responseError {
	return generateError(ErrCodeCaptchaEmpty)
}

// ErrCaptchaExpired reports an expired or unknown captcha.
func ErrCaptchaExpired() *responseError {
	return generateError(ErrCodeCaptchaExpired)
}

// ErrCaptchaError reports a wrong captcha answer.
func ErrCaptchaError() *responseError {
	return generateError(ErrCodeCaptchaError)
}

// ErrNotFound reports an unknown route.
func ErrNotFound() *responseError {
	return generateError(ErrCodeNotFound)
}

// ErrMethodNotAllowed reports an unsupported HTTP method for a route.
func ErrMethodNotAllowed() *responseError {
	return generateError(ErrCodeMethodNotAllowed)
}

func ErrBooleanInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeBooleanInvalid, convert(infos)...)
}

func ErrInteger16Invalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeInteger16Invalid, convert(infos)...)
}

func ErrInteger32Invalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeInteger32Invalid, convert(infos)...)
}

func ErrInteger64Invalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeInteger64Invalid, convert(infos)...)
}

func ErrFloat32Invalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeFloat32Invalid, convert(infos)...)
}

func ErrFloat64Invalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeFloat64Invalid, convert(infos)...)
}

// ErrThingModelNotFound reports that a thing model does not exist.
func ErrThingModelNotFound(id string) *responseError {
	return generateError(ErrCodeThingModelNotFound, id)
}

// ErrThingModelNameExists reports that a thing model name is already taken.
func ErrThingModelNameExists(name string) *responseError {
	return generateError(ErrCodeThingModelNameExists, name)
}

// ErrThingModelPropertyNotFound reports that a thing model property does not exist.
func ErrThingModelPropertyNotFound(identifier string) *responseError {
	return generateError(ErrCodeThingModelPropertyNotFound, identifier)
}

// ErrThingModelInvalid reports that the thing model failed validation.
func ErrThingModelInvalid() *responseError {
	return generateError(ErrCodeThingModelInvalid)
}

// ErrThingModelImportInvalid reports that the imported thing model cannot be parsed.
func ErrThingModelImportInvalid(reason string) *responseError {
	return generateError(ErrCodeThingModelImportInvalid, reason)
}

// ErrThingModelKeyExists reports that a thing model key is already taken.
func ErrThingModelKeyExists(key string) *responseError {
	return generateError(ErrCodeThingModelKeyExists, key)
}

// ErrThingNotFound reports that a thing does not exist.
func ErrThingNotFound(id string) *responseError {
	return generateError(ErrCodeThingNotFound, id)
}

// ErrThingDeviceNameExists reports that a device name is already taken in the model.
func ErrThingDeviceNameExists(name string) *responseError {
	return generateError(ErrCodeThingDeviceNameExists, name)
}

// ErrThingDeviceKeyExists reports that a device key is already taken.
func ErrThingDeviceKeyExists(key string) *responseError {
	return generateError(ErrCodeThingDeviceKeyExists, key)
}

// ErrThingTopicExists reports that a topic is already taken.
func ErrThingTopicExists(topic string) *responseError {
	return generateError(ErrCodeThingTopicExists, topic)
}

// ErrThingMQTTClientIDExists reports that an MQTT client id is already taken.
func ErrThingMQTTClientIDExists(id string) *responseError {
	return generateError(ErrCodeThingMQTTClientIDExists, id)
}

// ErrThingMQTTUsernameExists reports that an MQTT username is already taken.
func ErrThingMQTTUsernameExists(username string) *responseError {
	return generateError(ErrCodeThingMQTTUsernameExists, username)
}

// ErrThingInvalid reports that the thing failed validation.
func ErrThingInvalid() *responseError {
	return generateError(ErrCodeThingInvalid)
}

// ErrFileTooLarge reports that an uploaded file exceeds the configured limit.
func ErrFileTooLarge(limit int64) *responseError {
	return generateError(ErrCodeFileTooLarge, limit)
}

// ErrFileTypeInvalid reports that an uploaded file type is not allowed.
func ErrFileTypeInvalid() *responseError {
	return generateError(ErrCodeFileTypeInvalid)
}

// ErrFileUploadFailed wraps an unexpected upload failure.
func ErrFileUploadFailed(err error) *responseError {
	return generateErrorWrapper(ErrCodeFileUploadFailed, err)
}

// ErrDriverNotFound reports that a driver does not exist.
func ErrDriverNotFound(id string) *responseError {
	return generateError(ErrCodeDriverNotFound, id)
}

// ErrDriverConflict reports that a driver type/model or driver name is taken.
func ErrDriverConflict(value string) *responseError {
	return generateError(ErrCodeDriverConflict, value)
}

// ErrDriverBuiltinReadOnly reports that a built-in driver cannot be modified.
func ErrDriverBuiltinReadOnly() *responseError {
	return generateError(ErrCodeDriverBuiltinReadOnly)
}

// ErrDriverInvalid reports invalid driver metadata.
func ErrDriverInvalid(reason string) *responseError {
	return generateError(ErrCodeDriverInvalid, reason)
}

// ErrCollectionNotFound reports that a collection does not exist.
func ErrCollectionNotFound(id string) *responseError {
	return generateError(ErrCodeCollectionNotFound, id)
}

// ErrCollectionConflict reports a collection or variable conflict.
func ErrCollectionConflict(value string) *responseError {
	return generateError(ErrCodeCollectionConflict, value)
}

// ErrCollectionInvalid reports invalid collection metadata.
func ErrCollectionInvalid() *responseError {
	return generateError(ErrCodeCollectionInvalid)
}
