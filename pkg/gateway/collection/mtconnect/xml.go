package mtconnect

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type xmlNode struct {
	Name     string
	Attrs    map[string]string
	Text     string
	Children []*xmlNode
}

func parseXML(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	root := &xmlNode{Name: "#document"}
	stack := []*xmlNode{root}
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			node := &xmlNode{Name: localName(item.Name.Local), Attrs: map[string]string{}}
			for _, attr := range item.Attr {
				node.Attrs[localName(attr.Name.Local)] = attr.Value
			}
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, node)
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(item)
			}
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return root, nil
}

func localName(name string) string {
	if index := strings.LastIndex(name, ":"); index >= 0 {
		return name[index+1:]
	}
	return name
}

func findXPath(root *xmlNode, expression string) (string, error) {
	segments := splitPath(expression)
	if len(segments) == 0 {
		return "", fmt.Errorf("empty XPath")
	}
	nodes := []*xmlNode{root}
	for index, segment := range segments {
		name, attr, value, err := parseSegment(segment)
		if err != nil {
			return "", err
		}
		next := make([]*xmlNode, 0)
		for _, node := range nodes {
			collectMatches(node, name, attr, value, &next)
		}
		if len(next) == 0 {
			for _, node := range nodes {
				collectDescendants(node, name, attr, value, &next)
			}
		}
		if len(next) == 0 {
			return "", fmt.Errorf("no XML nodes match %q", expression)
		}
		if index == len(segments)-1 {
			for _, node := range next {
				text := strings.TrimSpace(node.Text)
				if text != "" {
					return text, nil
				}
			}
			return "", fmt.Errorf("XML node has no text")
		}
		nodes = next
	}
	return "", fmt.Errorf("no XML nodes match %q", expression)
}

func splitPath(expression string) []string {
	expression = strings.TrimSpace(strings.ReplaceAll(expression, "//", "/"))
	expression = strings.Trim(expression, "/")
	parts := strings.Split(expression, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}

func parseSegment(segment string) (string, string, string, error) {
	segment = strings.TrimSpace(segment)
	name := segment
	attr := ""
	value := ""
	if index := strings.Index(segment, "["); index >= 0 {
		name = segment[:index]
		predicate := strings.TrimSuffix(segment[index+1:], "]")
		predicate = strings.TrimSpace(predicate)
		if !strings.HasPrefix(predicate, "@") {
			return "", "", "", fmt.Errorf("unsupported XPath predicate %q", predicate)
		}
		predicate = strings.TrimPrefix(predicate, "@")
		if equal := strings.Index(predicate, "="); equal >= 0 {
			attr = strings.TrimSpace(predicate[:equal])
			value = strings.Trim(strings.TrimSpace(predicate[equal+1:]), `"'`)
		} else {
			attr = strings.TrimSpace(predicate)
		}
	}
	name = localName(strings.TrimSpace(name))
	if name == "" || name == "*" {
		return "", "", "", fmt.Errorf("invalid XPath element %q", segment)
	}
	return name, localName(attr), value, nil
}

func collectMatches(node *xmlNode, name, attr, value string, out *[]*xmlNode) {
	for _, child := range node.Children {
		if child.Name != name {
			continue
		}
		if attr != "" {
			actual, ok := child.Attrs[attr]
			if !ok || (value != "" && actual != value) {
				continue
			}
		}
		*out = append(*out, child)
	}
}

func collectDescendants(node *xmlNode, name, attr, value string, out *[]*xmlNode) {
	for _, child := range node.Children {
		collectMatches(child, name, attr, value, out)
		collectDescendants(child, name, attr, value, out)
	}
}

func convertValue(text, dataType string) (any, error) {
	switch strings.ToLower(strings.TrimSpace(dataType)) {
	case "bit":
		number, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil {
			return nil, err
		}
		return number, nil
	case "bool":
		switch strings.ToLower(strings.TrimSpace(text)) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid bool %q", text)
		}
	case "int8", "int16", "int32", "int64":
		return strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	case "uint8", "uint16", "uint32", "uint64":
		return strconv.ParseUint(strings.TrimSpace(text), 10, 64)
	case "float", "float32", "double", "float64":
		return strconv.ParseFloat(strings.TrimSpace(text), 64)
	case "string":
		return text, nil
	default:
		return text, nil
	}
}
