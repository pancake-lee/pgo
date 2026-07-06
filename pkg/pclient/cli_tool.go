package pclient

import (
	"strings"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/spf13/cobra"
)

// ToolOption contains configuration for a tool entrypoint
type ToolOption struct {
	ToolName       string
	Use            string
	Aliases        []string
	Short          string
	CacheKeyPrefix string
	ParamList      []ParamItem
	Run            func(values ParamMap) error

	// InteractiveHook allows additional interactive configuration beyond params
	InteractiveHook func(values ParamMap) ParamMap

	// CobraSetup allows additional cobra configuration
	CobraSetup func(cmd *cobra.Command) func(values ParamMap) ParamMap
}

// Tool provides unified CLI/Interactive/GUI tool execution
type Tool struct {
	option ToolOption
}

// NewTool creates a new tool
func NewTool(option ToolOption) *Tool {
	if option.ToolName == "" {
		option.ToolName = option.Use
	}
	return &Tool{option: option}
}

// RunInteractive runs the tool in interactive mode with cached parameters
func (x *Tool) RunInteractive() {
	cachePath := pconfig.GetDefaultCachePath()
	pthird.Interact.Infof("using cache file: %v", cachePath)
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

// NewCobraCommand creates a cobra command for CLI mode
func (x *Tool) NewCobraCommand() *cobra.Command {
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

// RunCommand runs the tool with command-line arguments
func (x *Tool) RunCommand(args []string) error {
	cmd := x.NewCobraCommand()
	cmd.SilenceUsage = true
	cmd.SetArgs(NormalizeLegacyLongFlagArgs(args))
	return cmd.Execute()
}

// NormalizeLegacyLongFlagArgs converts legacy args style: -db -> --db
// Only handles "single dash + multiple chars" args, preserves short args like -l/-h
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

		// Avoid misinterpreting negative literals like -1, -0.5
		c := arg[1]
		if c >= '0' && c <= '9' {
			norm = append(norm, arg)
			continue
		}

		norm = append(norm, "-"+arg)
	}

	return norm
}
