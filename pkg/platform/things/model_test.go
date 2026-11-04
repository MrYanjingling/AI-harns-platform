package things

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"harns-platform/pkg/platform/things/v1"
)

func TestNewDeviceKey(t *testing.T) {
	pattern := regexp.MustCompile(`^dk_[a-z0-9]{16}$`)
	for i := 0; i < 32; i++ {
		key, err := NewDeviceKey()
		require.NoError(t, err)
		assert.True(t, strings.HasPrefix(key, DeviceKeyPrefix))
		assert.Len(t, key, len(DeviceKeyPrefix)+16)
		assert.Regexp(t, pattern, key)
		assert.True(t, hasLetterAndDigit(key), key)
		assert.True(t, ValidDeviceKey(key))
	}
}

func TestNewSecretsAreHex(t *testing.T) {
	secret, err := NewDeviceSecret()
	require.NoError(t, err)
	assert.Len(t, secret, 64)

	token, err := NewToken()
	require.NoError(t, err)
	assert.Len(t, token, 64)
}

func TestValidDeviceKey(t *testing.T) {
	cases := map[string]bool{
		"device_001":          true,
		"dk_0123456789abcdef": true,
		"1bad":                false,
		"ab":                  false,
		"bad key":             false,
		"":                    false,
	}
	for key, want := range cases {
		assert.Equal(t, want, ValidDeviceKey(key), key)
	}
}

func TestTopicGeneration(t *testing.T) {
	assert.Equal(t, "iot/device_001/up", UpTopicFor("device_001"))
	assert.Equal(t, "iot/device_001/down", DownTopicFor("device_001"))
}

func TestApplyOnline(t *testing.T) {
	ttl := 60 * time.Second
	now := int64(1_000_000)

	fresh := &v1.ThingVo{}
	applyOnline(fresh, 1, float64(now-30_000), now, ttl)
	assert.Equal(t, 1, fresh.OnlineStatus)
	require.NotNil(t, fresh.LastActiveTime)
	assert.Equal(t, now-30_000, *fresh.LastActiveTime)

	stale := &v1.ThingVo{}
	applyOnline(stale, 1, float64(now-90_000), now, ttl)
	assert.Equal(t, 0, stale.OnlineStatus)
	require.NotNil(t, stale.LastActiveTime)
	assert.Equal(t, now-90_000, *stale.LastActiveTime)

	disabled := &v1.ThingVo{}
	applyOnline(disabled, 0, float64(now-1_000), now, ttl)
	assert.Equal(t, 0, disabled.OnlineStatus)
	require.NotNil(t, disabled.LastActiveTime)

	noScore := &v1.ThingVo{}
	applyOnline(noScore, 1, 0, now, ttl)
	assert.Equal(t, 0, noScore.OnlineStatus)
	assert.Nil(t, noScore.LastActiveTime)
}

func TestDedupeIDs(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, dedupeIDs([]string{" a ", "b", "a", ""}))
}

func TestValidationHelpers(t *testing.T) {
	assert.Empty(t, validateDeviceName("设备1", "deviceName"))
	assert.NotEmpty(t, validateDeviceName("", "deviceName"))
	assert.NotEmpty(t, validateDeviceName(strings.Repeat("a", 65), "deviceName"))
	assert.NotEmpty(t, validateDeviceDesc(strings.Repeat("a", 256), "deviceDesc"))
	assert.Empty(t, validateMQTTValue("client", "mqttClientId", "validate.thing_mqtt_client_id_invalid"))
	assert.NotEmpty(t, validateMQTTValue("", "mqttClientId", "validate.thing_mqtt_client_id_invalid"))
}

func TestValidateImageKey(t *testing.T) {
	assert.Empty(t, validateImageKey("", "image"))
	assert.Empty(t, validateImageKey("images/2026/11/01K.png", "image"))
	assert.NotEmpty(t, validateImageKey("https://example.com/a.png", "image"))
	assert.NotEmpty(t, validateImageKey("photos/a.png", "image"))
	assert.NotEmpty(t, validateImageKey("images/"+strings.Repeat("a", 250)+".png", "image"))
}
