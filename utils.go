package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// IncludeTag 用于标记需要包含的文件
type IncludeTag struct {
	Path string `yaml:"path"`
}

// UnmarshalYAML 实现自定义的解码逻辑
func (i *IncludeTag) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var path string
	if err := unmarshal(&path); err != nil {
		return err
	}
	i.Path = path
	return nil
}

// LoadYAMLFile 读取并解析 YAML 文件
func LoadYAMLFile(filename string, out interface{}) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// 自定义解码器
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	for {
		var node yaml.Node
		if err = decoder.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if err = processNode(&node, filepath.Dir(filename)); err != nil {
			return err
		}

		if err = node.Decode(out); err != nil {
			return err
		}
	}

	return nil
}

// processNode 处理 YAML 节点，支持 !include 标签
func processNode(node *yaml.Node, baseDir string) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!include" {
		includePath := node.Value
		if !filepath.IsAbs(includePath) {
			includePath = filepath.Join(baseDir, includePath)
		}

		data, err := ioutil.ReadFile(includePath)
		if err != nil {
			return err
		}

		var includedNode yaml.Node
		if err := yaml.Unmarshal(data, &includedNode); err != nil {
			return err
		}

		*node = includedNode
	}

	for _, child := range node.Content {
		if err := processNode(child, baseDir); err != nil {
			return err
		}
	}

	return nil
}
