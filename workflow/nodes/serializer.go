package nodes

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/bagaking/goulp/jsonex"
	"github.com/khicago/got/util/typer"
	"github.com/khicago/irr"
	"gopkg.in/yaml.v3"

	"github.com/bagaking/botheater/workflow"
)

type (
	// WFSerializer
	// jie
	WFSerializer struct {
		mode SerializeMode
	}

	SerializeMode string
)

const (
	SerializeModeAnyLst         SerializeMode = "any_lst"
	SerializeModeJsonStrLst     SerializeMode = "json_str_lst"
	SerializeModeYamlStrLst     SerializeMode = "yaml_str_lst"
	SerializeModeMarkdownStrLst SerializeMode = "markdown_str_lst"
	SerializeModeDefaultStrLst  SerializeMode = "default_str_lst"

	SerializeModeJsonStr     SerializeMode = "json_str"
	SerializeModeYamlStr     SerializeMode = "yaml_str"
	SerializeModeMarkdownStr SerializeMode = "markdown_str"
	SerializeModeDefaultStr  SerializeMode = "default_str"
)

var _ workflow.NodeDef = &WFSerializer{}

func NewSerializerNode(outMode SerializeMode) *WFSerializer {
	return &WFSerializer{mode: outMode}
}

func (n *WFSerializer) Execute(ctx context.Context, params workflow.ParamsTable, signal workflow.SignalTarget) (log string, err error) {
	finished := false
	defer func() {
		if err == nil && !finished {
			err = irr.Error("node is not finish")
		}
	}()

	input, ok := params[workflow.SingleNodeParamName]
	if !ok {
		return "", irr.Error("input param %s is not set", workflow.SingleNodeParamName)
	}

	var output any
	switch n.mode {
	case SerializeModeAnyLst:
		result := make([]any, 0, 1)
		_ = typer.DoAsSlice(input, func(val any) (e error) {
			result = append(result, val)
			return
		})
		output = result
	case SerializeModeJsonStr:
		output, err = toJsonStr(input)
	case SerializeModeJsonStrLst:
		output, err = processSlice(input, toJsonStr)
	case SerializeModeYamlStr:
		output, err = toYamlStr(input)
	case SerializeModeYamlStrLst:
		output, err = processSlice(input, toYamlStr)
	case SerializeModeMarkdownStr:
		output, err = toMarkdownStr(input)
	case SerializeModeMarkdownStrLst:
		output, err = processSlice(input, toMarkdownStr)
	case SerializeModeDefaultStr:
		output = fmt.Sprintf("%v", input)
	case SerializeModeDefaultStrLst:
		output, err = processSlice(input, func(item any) (string, error) {
			return fmt.Sprintf("%v", item), nil
		})
	default:
		return "", irr.Error("unsupported serialize mode %s", n.mode)
	}

	if err != nil {
		return "", err
	}

	finished, err = signal(ctx, workflow.SingleNodeParamName, output)
	if err != nil {
		return "", err
	}

	return "success, data serialized", nil
}

func processSlice(input any, processFunc func(any) (string, error)) (any, error) {
	var resultList []string
	err := typer.DoAsSlice[any](input, func(val any) error {
		result, err := processFunc(val)
		if err != nil {
			return err
		}
		resultList = append(resultList, result)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resultList, nil
}

func toJsonStr(input any) (string, error) {
	return jsonex.MarshalToString(input)
}

func toYamlStr(input any) (string, error) {
	var builder strings.Builder
	encoder := yaml.NewEncoder(&builder)
	encoder.SetIndent(2) // 设置缩进为 2
	defer encoder.Close()
	if err := encoder.Encode(input); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// toMarkdownStr converts various types of input to a Markdown formatted string.
func toMarkdownStr(input any) (string, error) {
	v := reflect.ValueOf(input)
	switch v.Kind() {
	case reflect.String:
		return fmt.Sprintf("`%s`", input), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("`%d`", input), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("`%d`", input), nil
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("`%f`", input), nil
	case reflect.Bool:
		return fmt.Sprintf("`%t`", input), nil
	case reflect.Slice, reflect.Array:
		var builder strings.Builder
		builder.WriteString("- List:\n")
		for i := 0; i < v.Len(); i++ {
			elem, err := toMarkdownStr(v.Index(i).Interface())
			if err != nil {
				return "", err
			}
			builder.WriteString(fmt.Sprintf("  - %s\n", elem))
		}
		return builder.String(), nil
	case reflect.Map:
		var builder strings.Builder
		builder.WriteString("- Map:\n")
		for _, key := range v.MapKeys() {
			val, err := toMarkdownStr(v.MapIndex(key).Interface())
			if err != nil {
				return "", err
			}
			builder.WriteString(fmt.Sprintf("  - `%v`: %s\n", key, val))
		}
		return builder.String(), nil
	case reflect.Struct:
		var builder strings.Builder
		builder.WriteString("- Struct:\n")
		for i := 0; i < v.NumField(); i++ {
			fieldName := v.Type().Field(i).Name
			fieldValue, err := toMarkdownStr(v.Field(i).Interface())
			if err != nil {
				return "", err
			}
			builder.WriteString(fmt.Sprintf("  - `%s`: %s\n", fieldName, fieldValue))
		}
		return builder.String(), nil
	default:
		return fmt.Sprintf("`%v`", input), nil
	}
}

func (n *WFSerializer) Name() string {
	return fmt.Sprintf("serializer(%s)", n.mode)
}

func (n *WFSerializer) InNames() []string {
	return []string{workflow.SingleNodeParamName}
}

func (n *WFSerializer) OutNames() []string {
	return []string{workflow.SingleNodeParamName}
}
