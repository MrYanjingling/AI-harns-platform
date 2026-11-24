package bacnet

import (
	"fmt"
	"strconv"
	"strings"
)

const propertyPresentValue = 85

var objectTypes = map[string]uint16{
	"AI": 0, "AO": 1, "AV": 2, "BI": 3, "BO": 4, "BV": 5,
	"DEV": 8, "MSI": 13, "MSO": 14, "MSV": 19, "ACC": 23,
}

var propertyNames = map[string]uint32{
	"present_value":                   85,
	"object_name":                     77,
	"description":                     28,
	"device_type":                     31,
	"status_flags":                    111,
	"event_state":                     36,
	"out_of_service":                  81,
	"update_interval":                 118,
	"min_pres_value":                  69,
	"max_pres_value":                  65,
	"resolution":                      106,
	"cov_increment":                   22,
	"time_delay":                      113,
	"notification_class":              17,
	"notify_type":                     72,
	"high_limit":                      45,
	"low_limit":                       59,
	"deadband":                        25,
	"units":                           117,
	"reliability":                     103,
	"object_type":                     79,
	"polarity":                        84,
	"system_status":                   112,
	"vendor_name":                     121,
	"vendor_identifier":               120,
	"model_name":                      70,
	"firmware_revision":               44,
	"application_software_version":    12,
	"location":                        58,
	"protocol_version":                98,
	"protocol_conformance_class":      95,
	"protocol_service_supported":      97,
	"protocol_object_types_supported": 96,
	"serial_number":                   372,
	"max_apdu_length_accepted":        62,
	"segmentation_supported":          107,
	"local_time":                      57,
	"local_date":                      56,
	"utc_offset":                      119,
	"daylight_savings_status":         24,
	"apdu_segment_timeout":            10,
	"apdu_timeout":                    11,
	"object_list":                     76,
	"property_list":                   371,
}

type Point struct {
	ObjectType uint16
	Instance   uint32
	Property   uint32
}

func ParseAddress(value string) (Point, error) {
	value = strings.TrimSpace(value)
	dot := strings.IndexByte(value, '.')
	head := value
	propertyText := ""
	if dot >= 0 {
		head = value[:dot]
		propertyText = value[dot+1:]
	}
	prefix := ""
	instanceText := ""
	for index, char := range head {
		if char < '0' || char > '9' {
			continue
		}
		prefix = strings.ToUpper(head[:index])
		instanceText = head[index:]
		break
	}
	if prefix == "" || instanceText == "" {
		return Point{}, fmt.Errorf("invalid BACnet object address %q", value)
	}
	objectType, ok := objectTypes[prefix]
	if !ok {
		return Point{}, fmt.Errorf("unsupported BACnet object type %q", prefix)
	}
	instance, err := strconv.ParseUint(instanceText, 10, 32)
	if err != nil || instance > 4194302 {
		return Point{}, fmt.Errorf("invalid BACnet object instance %q", instanceText)
	}
	property := uint32(propertyPresentValue)
	if propertyText != "" {
		property, err = parseProperty(propertyText)
		if err != nil {
			return Point{}, err
		}
	} else if prefix == "DEV" {
		return Point{}, fmt.Errorf("BACnet DEV address requires an explicit property")
	}
	return Point{ObjectType: objectType, Instance: uint32(instance), Property: property}, nil
}

func parseProperty(value string) (uint32, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "custom.") {
		id, err := strconv.ParseUint(value[len("custom."):], 10, 32)
		if err != nil || id > 4194303 {
			return 0, fmt.Errorf("invalid BACnet custom property id %q", value)
		}
		return uint32(id), nil
	}
	if id, ok := propertyNames[strings.ToLower(value)]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("unsupported BACnet property %q", value)
}
