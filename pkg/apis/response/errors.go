package response

import "strconv"

// messageIDs maps every error code to its i18n message ID. The actual message
// templates live in pkg/i18n/locales/*.yaml, keyed by these IDs.
var messageIDs = map[ErrCode]string{
	ErrCodeMalformedJSON:               "error.malformed_json",
	ErrCodeRequestBody:                 "error.request_body",
	ErrCodeResourceExists:              "error.resource_exists",
	ErrCodeResourceNotFound:            "error.resource_not_found",
	ErrCodeLegalActionNotFound:         "error.legal_action_not_found",
	ErrCodeBooleanInvalid:              "error.boolean_invalid",
	ErrCodeInteger16Invalid:            "error.integer16_invalid",
	ErrCodeInteger32Invalid:            "error.integer32_invalid",
	ErrCodeInteger64Invalid:            "error.integer64_invalid",
	ErrCodeFloat32Invalid:              "error.float32_invalid",
	ErrCodeFloat64Invalid:              "error.float64_invalid",
	ErrCodeDeviceNotFound:              "error.device_not_found",
	ErrCodeVariableNotWritable:         "error.variable_not_writable",
	ErrCodeDeviceNotConnect:            "error.device_not_connect",
	ErrCodeDeviceOperatorUnSupported:   "error.device_operator_unsupported",
	ErrCodeTooManyJsonPatchOperations:  "error.too_many_json_patch_operations",
	ErrCodeInternal:                    "error.internal",
	ErrCodeUserNotFound:                "error.user_not_found",
	ErrCodeUserNameExists:              "error.user_name_exists",
	ErrCodeUserNoExists:                "error.user_no_exists",
	ErrCodeVersionMismatch:             "error.version_mismatch",
	ErrCodeRoleNotFound:                "error.role_not_found",
	ErrCodeRoleNameExists:              "error.role_name_exists",
	ErrCodeRoleCodeExists:              "error.role_code_exists",
	ErrCodeOrgNotFound:                 "error.org_not_found",
	ErrCodeOrgCodeExists:               "error.org_code_exists",
	ErrCodeOrgHasChildren:              "error.org_has_children",
	ErrCodeOrgParentNotFound:           "error.org_parent_not_found",
	ErrCodeOrgParentCycle:              "error.org_parent_cycle",
	ErrCodeLoginExpired:                "error.login_expired",
	ErrCodeUnauthorized:                "error.unauthorized",
	ErrCodeUsernamePasswordError:       "error.username_password_error",
	ErrCodeAccountDisabled:             "error.account_disabled",
	ErrCodeAccountLocked:               "error.account_locked",
	ErrCodeLoginFailed:                 "error.login_failed",
	ErrCodeCaptchaEmpty:                "error.captcha_empty",
	ErrCodeCaptchaExpired:              "error.captcha_expired",
	ErrCodeCaptchaError:                "error.captcha_error",
	ErrCodeNotFound:                    "error.not_found",
	ErrCodeMethodNotAllowed:            "error.method_not_allowed",
	ErrCodeValidationFailed:            "error.validation_failed",
	ErrCodeThingModelNotFound:          "error.thing_model_not_found",
	ErrCodeThingModelNameExists:        "error.thing_model_name_exists",
	ErrCodeThingModelPropertyNotFound:  "error.thing_model_property_not_found",
	ErrCodeThingModelInvalid:           "error.thing_model_invalid",
	ErrCodeThingModelImportInvalid:     "error.thing_model_import_invalid",
	ErrCodeThingModelKeyExists:         "error.thing_model_key_exists",
	ErrCodeThingNotFound:               "error.thing_not_found",
	ErrCodeThingDeviceNameExists:       "error.thing_device_name_exists",
	ErrCodeThingDeviceKeyExists:        "error.thing_device_key_exists",
	ErrCodeThingTopicExists:            "error.thing_topic_exists",
	ErrCodeThingMQTTClientIDExists:     "error.thing_mqtt_client_id_exists",
	ErrCodeThingMQTTUsernameExists:     "error.thing_mqtt_username_exists",
	ErrCodeThingInvalid:                "error.thing_invalid",
	ErrCodeFileTooLarge:                "error.file_too_large",
	ErrCodeFileTypeInvalid:             "error.file_type_invalid",
	ErrCodeFileUploadFailed:            "error.file_upload_failed",
	ErrCodeDriverNotFound:              "error.driver_not_found",
	ErrCodeDriverConflict:              "error.driver_conflict",
	ErrCodeDriverBuiltinReadOnly:       "error.driver_builtin_read_only",
	ErrCodeDriverInvalid:               "error.driver_invalid",
	ErrCodeCollectionNotFound:          "error.collection_not_found",
	ErrCodeCollectionConflict:          "error.collection_conflict",
	ErrCodeCollectionInvalid:           "error.collection_invalid",
	ErrCodePushConnectorNotFound:       "error.push_connector_not_found",
	ErrCodePushConnectorConflict:       "error.push_connector_conflict",
	ErrCodePushConnectorInvalid:        "error.push_connector_invalid",
	ErrCodePushGroupNotFound:           "error.push_group_not_found",
	ErrCodePushGroupConflict:           "error.push_group_conflict",
	ErrCodePushGroupInvalid:            "error.push_group_invalid",
	ErrCodePushConnectorAlreadyRunning: "error.push_connector_already_running",
	ErrCodePushConnectorAlreadyStopped: "error.push_connector_already_stopped",
	ErrCodePushConnectorStartFailed:    "error.push_connector_start_failed",
	ErrCodePushConnectorStopFailed:     "error.push_connector_stop_failed",
	ErrCodeSystemLogFileUnavailable:    "error.system_log_file_unavailable",
}

func messageID(code ErrCode) string {
	if id, ok := messageIDs[code]; ok {
		return id
	}
	return strconv.Itoa(int(code))
}

// !!! IMPORTANT PLEASE READ FIRST !!!
// You SHOULD add new code at the end of enum firstly.

var (
	ErrMalformedJSON       = generateError(ErrCodeMalformedJSON)
	ErrRequestBody         = generateError(ErrCodeRequestBody)
	ErrLegalActionNotFound = generateError(ErrCodeLegalActionNotFound)
)
