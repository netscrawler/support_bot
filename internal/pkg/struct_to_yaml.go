package pkg

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"
)

func StructToYAMLNode(v any) (*yaml.Node, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct")
	}

	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	rt := rv.Type()

	for i := 0; i < rv.NumField(); i++ {
		field := rt.Field(i)
		value := rv.Field(i)

		yamlKey := field.Tag.Get("yaml")
		if yamlKey == "" || yamlKey == "-" {
			continue
		}

		comment := field.Tag.Get("comment")

		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: yamlKey,
		}

		if comment != "" {
			keyNode.HeadComment = comment
		}

		var valueNode *yaml.Node

		switch value.Kind() {
		case reflect.Struct:
			child, err := StructToYAMLNode(value.Interface())
			if err != nil {
				return nil, err
			}

			valueNode = child

		default:
			valueNode = &yaml.Node{}
			if err := valueNode.Encode(value.Interface()); err != nil {
				return nil, err
			}
		}

		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node, nil
}
