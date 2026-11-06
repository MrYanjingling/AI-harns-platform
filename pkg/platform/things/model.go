// Package things implements thing (device) management: batch creation,
// querying, updating and batch deletion.
package things

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"harns-platform/pkg/apis"
)

const (
	// MaxBatchSize is the maximum number of things created in one request.
	MaxBatchSize = 100
	// DeviceKeyPrefix prefixes auto-generated device keys.
	DeviceKeyPrefix = "dk_"
	// TopicPrefix prefixes auto-generated MQTT topics.
	TopicPrefix = "iot/"
	// LastActiveKey is the Redis sorted set holding last report timestamps.
	LastActiveKey = "iot:thing:last-active"
)

// Thing is a device entity.
type Thing struct {
	apis.Meta
	ThingModelID string                     `json:"thingModelId"`
	DeviceName   string                     `json:"deviceName"`
	DeviceKey    string                     `json:"deviceKey"`
	DeviceSecret string                     `json:"-"`
	Token        string                     `json:"-"`
	UpTopic      string                     `json:"upTopic"`
	DownTopic    string                     `json:"downTopic"`
	DeviceDesc   string                     `json:"deviceDesc"`
	Features     map[string]json.RawMessage `json:"features,omitempty"`
	Image        string                     `json:"image"`
	MQTTClientID string                     `json:"mqttClientId"`
	MQTTUsername string                     `json:"mqttUsername"`
	MQTTPassword string                     `json:"-"`
	Enable       int                        `json:"enable"`
	Debug        int                        `json:"debug"`
}

// ThingBasic is a thing joined with model and creator/updater display fields.
type ThingBasic struct {
	Thing
	ThingModelKey       string
	ThingModelName      string
	CreatedUserName     string
	CreatedUserNickName string
	UpdatedUserName     string
	UpdatedUserNickName string
}

// ThingIdentity is the non-secret identity cached for token lookups.
type ThingIdentity struct {
	ID            string `json:"id"`
	ThingModelID  string `json:"thingModelId"`
	ThingModelKey string `json:"thingModelKey"`
	DeviceKey     string `json:"deviceKey"`
	DeviceName    string `json:"deviceName"`
	MQTTClientID  string `json:"mqttClientId"`
	MQTTUsername  string `json:"mqttUsername"`
	UpTopic       string `json:"upTopic"`
	DownTopic     string `json:"downTopic"`
	Enable        int    `json:"enable"`
	Debug         int    `json:"debug"`
}

// ThingSnapshot is the cached, credential-inclusive view of a thing.
type ThingSnapshot struct {
	ID            string                     `json:"id"`
	Version       int64                      `json:"version"`
	ThingModelID  string                     `json:"thingModelId"`
	ThingModelKey string                     `json:"thingModelKey"`
	DeviceName    string                     `json:"deviceName"`
	DeviceKey     string                     `json:"deviceKey"`
	DeviceSecret  string                     `json:"deviceSecret"`
	Token         string                     `json:"token"`
	UpTopic       string                     `json:"upTopic"`
	DownTopic     string                     `json:"downTopic"`
	DeviceDesc    string                     `json:"deviceDesc"`
	Features      map[string]json.RawMessage `json:"features,omitempty"`
	Image         string                     `json:"image"`
	MQTTClientID  string                     `json:"mqttClientId"`
	MQTTUsername  string                     `json:"mqttUsername"`
	MQTTPassword  string                     `json:"mqttPassword"`
	Enable        int                        `json:"enable"`
	Debug         int                        `json:"debug"`
	CreatedUser   string                     `json:"createdUser"`
	UpdatedUser   string                     `json:"updatedUser"`
	CreatedTime   time.Time                  `json:"createdTime"`
	UpdatedTime   time.Time                  `json:"updatedTime"`
}

// AuthThing is the authentication view of a thing, including its model key.
type AuthThing struct {
	Thing
	ThingModelKey string
}

// FieldIssue is a single field-level validation failure.
type FieldIssue struct {
	Field string
	MsgID string
	Args  []interface{}
}

// ValidationError carries field-level validation failures.
type ValidationError struct {
	Issues []FieldIssue
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	parts := make([]string, 0, len(e.Issues))
	for _, issue := range e.Issues {
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Field, issue.MsgID))
	}
	return strings.Join(parts, "; ")
}

// NewValidationError returns nil when there are no issues.
func NewValidationError(issues []FieldIssue) *ValidationError {
	if len(issues) == 0 {
		return nil
	}
	return &ValidationError{Issues: issues}
}

var deviceKeyPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{3,63}$`)

// NewID returns a new ULID.
func NewID() string {
	return ulid.Make().String()
}

const (
	deviceKeyCharset       = "0123456789abcdefghijklmnopqrstuvwxyz"
	deviceKeySuffixLength  = 16
	deviceKeyCharsetModulo = 252 // largest multiple of len(deviceKeyCharset) <= 255
)

// NewDeviceKey returns an auto-generated device key.
func NewDeviceKey() (string, error) {
	for {
		suffix, err := newDeviceKeySuffix(deviceKeySuffixLength)
		if err != nil {
			return "", err
		}
		if hasLetterAndDigit(suffix) {
			return DeviceKeyPrefix + suffix, nil
		}
	}
}

func newDeviceKeySuffix(length int) (string, error) {
	out := make([]byte, length)
	var buf [1]byte
	for i := 0; i < length; {
		if _, err := rand.Read(buf[:]); err != nil {
			return "", err
		}
		if int(buf[0]) >= deviceKeyCharsetModulo {
			continue
		}
		out[i] = deviceKeyCharset[int(buf[0])%len(deviceKeyCharset)]
		i++
	}
	return string(out), nil
}

func hasLetterAndDigit(value string) bool {
	hasLetter, hasDigit := false, false
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			hasLetter = true
		case char >= '0' && char <= '9':
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}

// NewDeviceSecret returns a 32-byte random secret as hex.
func NewDeviceSecret() (string, error) {
	return randomHex(32)
}

// NewToken returns a 32-byte random token as hex.
func NewToken() (string, error) {
	return randomHex(32)
}

// UpTopicFor returns the upstream topic (device -> platform) of a device.
func UpTopicFor(deviceKey string) string {
	return TopicPrefix + deviceKey + "/up"
}

// DownTopicFor returns the downstream topic (platform -> device) of a device.
func DownTopicFor(deviceKey string) string {
	return TopicPrefix + deviceKey + "/down"
}

// ValidDeviceKey reports whether a user supplied device key is well formed.
func ValidDeviceKey(key string) bool {
	return deviceKeyPattern.MatchString(key)
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
