package things

import (
	"encoding/json"
	"time"

	"harns-platform/pkg/platform/thingmodel/tsl"
)

const uploadFormatPropertyLimit = 3

func buildUploadFormat(properties []tsl.Property, nowMillis int64) (string, error) {
	values := make(map[string]any, len(properties))
	for _, property := range properties {
		if property.Identifier == "" {
			continue
		}
		values[property.Identifier] = uploadSampleValue(property.DataType)
	}
	document := map[string]any{
		"type": "property",
		"payload": []any{
			map[string]any{
				"time":     nowMillis,
				"property": values,
			},
		},
	}
	encoded, err := json.Marshal(document)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func uploadSampleValue(dataType tsl.DataTypeDetail) any {
	switch dataType.Type {
	case tsl.DataTypeInt, tsl.DataTypeLong:
		return int64(0)
	case tsl.DataTypeFloat, tsl.DataTypeDouble:
		return float64(0)
	case tsl.DataTypeBool:
		return false
	case tsl.DataTypeEnum:
		if dataType.EnumSpec != nil && len(dataType.EnumSpec.Values) > 0 {
			return dataType.EnumSpec.Values[0].Value
		}
		return 0
	case tsl.DataTypeText, tsl.DataTypeTime:
		return ""
	case tsl.DataTypeArray:
		return []any{}
	case tsl.DataTypeStruct:
		return map[string]any{}
	default:
		return nil
	}
}

func uploadFormatNow() int64 {
	return time.Now().UnixMilli()
}
