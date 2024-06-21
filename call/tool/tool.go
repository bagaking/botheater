package tool

type (
	ITool interface {
		Execute(params map[string]string) (any, error)
		Name() string
		Usage() string
		Examples() []string
		ParamNames() []string
	}
)
