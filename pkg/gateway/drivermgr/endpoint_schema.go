package drivermgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

var endpointDefaultPorts = map[string]int{
	"s7-1500":                       102,
	"s7-200":                        102,
	"opcUa":                         4840,
	"fins":                          9600,
	"omronCip":                      44818,
	"ethernetIp":                    44818,
	"EtherNet/IP-ControlLogic 55xx": 44818,
	"EtherNet/IP-MicroLogix 1400":   44818,
	"ethercat":                      48898,
	"profinet":                      102,
	"dnp3":                          20000,
	"heidenhain":                    19000,
	"fanuc focas":                   8193,
	"MTConnect":                     5000,
	"Kuka":                          54601,
	"codesys-v3":                    11740,
	"ab5000":                        44818,
	"BACnet":                        47808,
	"Beckhoff ADS":                  48898,
	"SRTP":                          18245,
	"SNMP-V2":                       161,
	"IEC61850":                      102,
	"MC-1E":                         2000,
	"MC-3E":                         2000,
	"MC-4E":                         2000,
	"Mewtocol":                      2000,
	"dlt645_2007":                   2010,
	"dlt645_1997":                   2010,
	"cjt188_2004":                   37121,
	"cjt188_2018":                   37121,
	"dlt698_45":                     4059,
	"ppi":                           32502,
	"HJ212":                         9000,
}

func normalizeBuiltinSchema(raw string, model string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return raw, nil
	}
	var schema Schema
	if err := json.Unmarshal([]byte(raw), &schema); err != nil {
		return "", err
	}
	var port *Field
	var source *Field
	for _, field := range schema.Fields {
		if field == nil {
			continue
		}
		switch field.Name {
		case "port":
			port = field
		case "host", "location":
			if source == nil || field.Name == "host" {
				source = field
			}
		}
	}
	if port == nil || source == nil {
		return raw, nil
	}
	link := findField(schema.Fields, "link")
	defaultPort := endpointDefaultPorts[model]
	if defaultPort == 0 {
		defaultPort, _ = strconv.Atoi(strings.TrimSpace(port.PlaceholderZh))
	}
	if defaultPort <= 0 {
		defaultPort = 102
	}
	serial := link != nil
	basePlaceholder := strings.TrimSpace(source.PlaceholderZh)
	if basePlaceholder == "" && strings.Contains(source.Regex, "opc") {
		basePlaceholder = "opc.tcp://IP"
	}
	placeholder := endpointPlaceholder(basePlaceholder, defaultPort)
	field := &Field{
		Name:          "location",
		LabelZh:       "目标地址",
		LabelEn:       "Target Address",
		Regex:         `^\S+$`,
		Sort:          source.Sort,
		Type:          FieldTypeInput,
		Required:      true,
		PlaceholderZh: placeholder,
		PlaceholderEn: endpointPlaceholder(strings.TrimSpace(source.PlaceholderEn), defaultPort),
	}
	if serial {
		field.LabelZh = "目标地址"
		field.LabelEn = "Target Address"
		field.Variants = []*FieldVariant{
			{
				When:          FieldCondition{Field: link.Name, Equals: "0"},
				LabelZh:       "串口设备",
				LabelEn:       "Serial Port",
				Regex:         `^\S+$`,
				PlaceholderZh: "/dev/ttyUSB0",
				PlaceholderEn: "/dev/ttyUSB0",
			},
			{
				When:          FieldCondition{Field: link.Name, Equals: "1"},
				LabelZh:       "目标地址",
				LabelEn:       "Target Address",
				Regex:         `^\S+$`,
				PlaceholderZh: placeholder,
				PlaceholderEn: endpointPlaceholder(strings.TrimSpace(source.PlaceholderEn), defaultPort),
			},
		}
	}
	fields := make([]*Field, 0, len(schema.Fields))
	for _, current := range schema.Fields {
		switch {
		case current == nil:
			continue
		case current.Name == "port":
			continue
		case current == source:
			fields = append(fields, field)
		default:
			fields = append(fields, current)
		}
	}
	schema.Fields = fields
	schema.Endpoint = &EndpointSchema{Source: source.Name, DefaultPort: defaultPort}
	if link != nil {
		schema.Endpoint.Link = link.Name
	}
	normalized, err := json.Marshal(&schema)
	if err != nil {
		return "", err
	}
	if _, err := ParseSchema(normalized); err != nil {
		return "", err
	}
	return string(normalized), nil
}

func findField(fields []*Field, name string) *Field {
	for _, field := range fields {
		if field != nil && field.Name == name {
			return field
		}
	}
	return nil
}

func endpointPlaceholder(base string, port int) string {
	base = strings.TrimSpace(base)
	if base != "" {
		if strings.Contains(base, "://") {
			return strings.TrimRight(base, "/") + ":" + strconv.Itoa(port)
		}
		return base + ":" + strconv.Itoa(port)
	}
	return "IP:" + strconv.Itoa(port)
}

// SplitEndpointBody converts the merged location field into the host/location
// and port fields still consumed by built-in drivers.
func (s *Schema) SplitEndpointBody(body []byte) ([]byte, error) {
	if s == nil || s.Endpoint == nil {
		return body, nil
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var protocol map[string]json.RawMessage
	if raw, ok := payload["protocolConfig"]; ok {
		if err := json.Unmarshal(raw, &protocol); err != nil {
			return nil, err
		}
	}
	if protocol == nil {
		return body, nil
	}
	var location string
	if raw, ok := protocol["location"]; ok {
		_ = json.Unmarshal(raw, &location)
	}
	if strings.TrimSpace(location) == "" {
		return body, nil
	}
	if s.Endpoint.Link != "" && endpointLinkSerial(protocol, s.Endpoint.Link) {
		return body, nil
	}
	defaultPort := s.Endpoint.DefaultPort
	if raw, ok := protocol["port"]; ok {
		var configuredPort int
		if json.Unmarshal(raw, &configuredPort) == nil && configuredPort > 0 {
			defaultPort = configuredPort
		}
	}
	host, port, value, err := splitEndpointLocation(location, defaultPort, s.Endpoint.Source)
	if err != nil {
		return nil, err
	}
	protocol["location"] = mustJSON(value)
	protocol["host"] = mustJSON(host)
	if port > 0 {
		protocol["port"] = mustJSON(port)
	}
	payload["protocolConfig"] = mustJSON(protocol)
	return json.Marshal(payload)
}

// MergeEndpointConfig rewrites a stored driver config to the merged location
// representation used by the API and UI.
func (s *Schema) MergeEndpointConfig(raw []byte) ([]byte, error) {
	if s == nil || s.Endpoint == nil || len(bytes.TrimSpace(raw)) == 0 {
		return raw, nil
	}
	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return nil, err
	}
	if config == nil {
		return raw, nil
	}
	var link int
	if s.Endpoint.Link != "" {
		_ = json.Unmarshal(config[s.Endpoint.Link], &link)
	}
	if s.Endpoint.Link != "" && link == 0 {
		config["location"] = config[sourceKey(s.Endpoint.Source)]
		delete(config, "host")
		delete(config, "port")
		return json.Marshal(config)
	}
	source := "location"
	if s.Endpoint.Source == "host" {
		source = "host"
	}
	var value string
	_ = json.Unmarshal(config[source], &value)
	var port int
	_ = json.Unmarshal(config["port"], &port)
	if s.Endpoint.DefaultPort > 0 && port == 0 {
		port = s.Endpoint.DefaultPort
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return raw, nil
	}
	if strings.Contains(value, "://") {
		if !strings.HasSuffix(value, ":"+strconv.Itoa(port)) {
			value = strings.TrimRight(value, "/") + ":" + strconv.Itoa(port)
		}
	} else {
		value = net.JoinHostPort(value, strconv.Itoa(port))
	}
	config["location"] = mustJSON(value)
	delete(config, "host")
	delete(config, "port")
	return json.Marshal(config)
}

func endpointLinkSerial(protocol map[string]json.RawMessage, name string) bool {
	raw, ok := protocol[name]
	if !ok {
		return true
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text == "0"
	}
	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number == 0
	}
	return false
}

func splitEndpointLocation(value string, defaultPort int, source string) (string, int, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, "", nil
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || parsed.Hostname() == "" {
			return "", 0, "", fmt.Errorf("invalid endpoint %q", value)
		}
		port := defaultPort
		if parsed.Port() != "" {
			port, err = strconv.Atoi(parsed.Port())
			if err != nil {
				return "", 0, "", fmt.Errorf("invalid endpoint port %q", parsed.Port())
			}
		}
		location := value
		if source == "host" {
			location = parsed.Hostname()
		}
		return parsed.Hostname(), port, location, nil
	}
	host := value
	port := defaultPort
	if separator := strings.LastIndex(value, ":"); separator > 0 && separator < len(value)-1 {
		if parsedPort, err := strconv.Atoi(value[separator+1:]); err == nil {
			host = value[:separator]
			port = parsedPort
		}
	}
	return host, port, host, nil
}

func sourceKey(source string) string {
	if source == "host" {
		return "host"
	}
	return "location"
}

func mustJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}
