package tools

import (
	"errors"
	"os"

	"github.com/khicago/irr"

	"github.com/bagaking/botheater/call"
	"github.com/bagaking/botheater/call/tool"
)

// LocalFileReader 结构体
type (
	LocalFileReader       struct{}
	LocalFileReaderParams struct {
		Path string `json:"path"`
	}
)

var _ tool.ITool = &LocalFileReader{}

const maxFileSize = 10 * 1024

// Name 返回工具名称
func (l *LocalFileReader) Name() string {
	return "local_file_reader"
}

func (l *LocalFileReader) Usage() string {
	return "获取访问地址对应的文件内容，如果地址是一个目录，则范围目录中的文件列表"
}

func (l *LocalFileReader) Examples() []string {
	return []string{"local_file_reader(.) // 获得根目录想所有文件", "local_file_reader(./README.md) // 读取 README.md 中的内容"}
}

func (l *LocalFileReader) ParamNames() []string {
	return []string{"path"}
}

// Execute 执行文件读取操作
func (l *LocalFileReader) Execute(param map[string]string) (any, error) {
	path, ok := param["path"]
	if !ok {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "parameter 'path' is required in %v", param)
	}
	if path == "" {
		return nil, irr.Wrap(call.ErrExecFailedInvalidParams, "path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("path does not exist")
		}
		return nil, err
	}

	if info.IsDir() {
		return l.readDirectory(path)
	} else {
		return l.readFile(path, info.Size())
	}
}

// readDirectory 读取目录内容
func (l *LocalFileReader) readDirectory(path string) ([]map[string]any, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fileInfos []map[string]any
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		fileInfos = append(fileInfos, map[string]any{
			"name":  info.Name(),
			"isDir": info.IsDir(),
		})
	}

	return fileInfos, nil
}

// readFile 读取文件内容
func (l *LocalFileReader) readFile(path string, size int64) (string, error) {
	if size > maxFileSize {
		return "", errors.New("文件内容过大")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}
