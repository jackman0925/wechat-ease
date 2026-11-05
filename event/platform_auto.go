package event

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// ParsePlatformEvent 自动解析已验签、解密后的小程序平台通知正文。
// 正文以 { 开头时按 JSON 解析，以 < 开头时按 XML 解析，其他格式返回错误。
func ParsePlatformEvent(body []byte) (PlatformEvent, error) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, ErrInvalidPlatformEvent
	}
	switch body[0] {
	case '{':
		return ParsePlatformEventJSON(body)
	case '<':
		return parsePlatformEventXML(body)
	default:
		return nil, ErrInvalidPlatformEvent
	}
}

func parsePlatformEventXML(body []byte) (PlatformEvent, error) {
	fields, err := parsePlatformEventXMLFields(body)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizePlatformEventXMLFields(fields)
	if err != nil {
		return nil, err
	}
	return ParsePlatformEventJSON(normalized)
}

func parsePlatformEventXMLFields(body []byte) (map[string][]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	fields := make(map[string][]string)
	rootFound := false
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("%w: decode XML: %v", ErrInvalidPlatformEvent, err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if !rootFound {
				if value.Name.Local != "xml" {
					return nil, fmt.Errorf("%w: XML root must be xml", ErrInvalidPlatformEvent)
				}
				rootFound = true
				continue
			}
			var fieldValue string
			if err := decoder.DecodeElement(&fieldValue, &value); err != nil {
				return nil, fmt.Errorf("%w: decode XML field %s: %v", ErrInvalidPlatformEvent, value.Name.Local, err)
			}
			fields[value.Name.Local] = append(fields[value.Name.Local], fieldValue)
		case xml.EndElement:
			if rootFound && value.Name.Local == "xml" {
				return finishPlatformEventXML(decoder, fields)
			}
		}
	}
}

func finishPlatformEventXML(decoder *xml.Decoder, fields map[string][]string) (map[string][]string, error) {
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return fields, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%w: decode XML: %v", ErrInvalidPlatformEvent, err)
		}
		if text, ok := token.(xml.CharData); ok && len(bytes.TrimSpace(text)) != 0 {
			return nil, fmt.Errorf("%w: unexpected content after XML root", ErrInvalidPlatformEvent)
		}
	}
}

func normalizePlatformEventXMLFields(fields map[string][]string) ([]byte, error) {
	normalized := make(map[string]any, len(fields))
	for name, values := range fields {
		if isPlatformEventNumberField(name) {
			if len(values) != 1 {
				return nil, fmt.Errorf("%w: XML field %s must have one value", ErrInvalidPlatformEvent, name)
			}
			value, err := strconv.ParseInt(values[0], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%w: XML field %s must be a number", ErrInvalidPlatformEvent, name)
			}
			normalized[name] = value
			continue
		}
		if len(values) == 1 {
			normalized[name] = values[0]
			continue
		}
		normalized[name] = values
	}
	body, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("%w: normalize XML: %v", ErrInvalidPlatformEvent, err)
	}
	return body, nil
}

func isPlatformEventNumberField(name string) bool {
	switch name {
	case "CreateTime", "event_type", "punish_time":
		return true
	default:
		return false
	}
}
