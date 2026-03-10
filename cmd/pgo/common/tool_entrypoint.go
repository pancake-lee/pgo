package common

import (
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"github.com/spf13/cobra"
)

type ToolEntrypointOption struct {
	ToolName       string
	Use            string
	Aliases        []string
	Short          string
	CacheKeyPrefix string
	ParamList      []ParamItem
	Run            func(values ParamMap) error

	// 除了param配置外，还可以通过InteractiveHook做额外的交互式配置
	InteractiveHook func(values ParamMap) ParamMap

	// 除了param配置外，还可以通过CobraSetup做额外的cobra配置
	CobraSetup func(cmd *cobra.Command) func(values ParamMap) ParamMap
}

type ToolEntrypoint struct {
	option ToolEntrypointOption
}

func NewToolEntrypoint(option ToolEntrypointOption) *ToolEntrypoint {
	if option.ToolName == "" {
		option.ToolName = option.Use
	}
	return &ToolEntrypoint{option: option}
}

func (x *ToolEntrypoint) RunInteractive() {
	cachePath := pconfig.GetDefaultCachePath()
	putil.Interact.Infof("using cache file: %v", cachePath)
	values := GetCachedParamMap(
		cachePath,
		x.option.CacheKeyPrefix,
		x.option.ParamList)

	if x.option.InteractiveHook != nil {
		nextValues := x.option.InteractiveHook(values)
		if nextValues != nil {
			values = nextValues
		}
	}

	err := x.option.Run(values)
	if err != nil {
		plogger.Errorf("%s failed: %v", x.option.ToolName, err)
	}
}

func (x *ToolEntrypoint) NewCobraCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     x.option.Use,
		Aliases: x.option.Aliases,
		Short:   x.option.Short,
	}

	flagRefs := RegParamToCobra(cmd, x.option.ParamList)

	var cobraHook func(values ParamMap) ParamMap
	if x.option.CobraSetup != nil {
		cobraHook = x.option.CobraSetup(cmd)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		values := ParseParamFromCobra(flagRefs)
		if cobraHook != nil {
			nextValues := cobraHook(values)
			if nextValues != nil {
				values = nextValues
			}
		}
		return x.option.Run(values)
	}

	return cmd
}

func (x *ToolEntrypoint) RunCommand(args []string) error {
	cmd := x.NewCobraCommand()
	cmd.SilenceUsage = true
	cmd.SetArgs(NormalizeLegacyLongFlagArgs(args))
	return cmd.Execute()
}

// NormalizeLegacyLongFlagArgs 兼容历史参数风格：-db -> --db
// 仅处理“单横杠 + 多字符”参数，保留 -l/-h 这类短参数不变。
func NormalizeLegacyLongFlagArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	norm := make([]string, 0, len(args))
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") || len(arg) <= 2 {
			norm = append(norm, arg)
			continue
		}

		// 避免把负数字面量（如 -1、-0.5）误判成参数
		c := arg[1]
		if c >= '0' && c <= '9' {
			norm = append(norm, arg)
			continue
		}

		norm = append(norm, "-"+arg)
	}

	return norm
}
