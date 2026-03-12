package sheet2mysql

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pancake-lee/pgo/cmd/pgo/common"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// --------------------------------------------------
const (
	paramNameConfig      = "config"
	paramNameBaseURL     = "baseUrl"
	paramNameSpaceID     = "spaceId"
	paramNameToken       = "token"
	paramNameDatasheetID = "datasheetId"
	paramNameTableName   = "tableName"
	paramNameOutFolder   = "outFolder"
	cacheKeyPrefix       = "tools.sheet2mysql."
)

var paramSettingList = []common.ParamItem{
	{
		Name:    paramNameConfig,
		Usage:   "配置文件路径，用于自动读取 APITable.token/baseUrl/spaceId（对应参数为空时生效）",
		Default: "./configs/pancake.yaml",
	}, {
		Name:    paramNameBaseURL,
		Usage:   "APITable base url（空则从 config 读取）",
		Default: "",
	}, {
		Name:    paramNameSpaceID,
		Usage:   "APITable space id（空则从 config 读取）",
		Default: "",
	}, {
		Name:    paramNameToken,
		Usage:   "APITable token（空则从 config 读取）",
		Default: "",
	}, // --------------------------------------------------
	{
		Name:    paramNameDatasheetID,
		Usage:   "APITable datasheet id",
		Default: "",
	}, {
		Name:    paramNameTableName,
		Usage:   "target mysql table name (建议填写 APITable 页面显示名，中文会自动转拼音)",
		Default: "",
	}, {
		Name:    paramNameOutFolder,
		Usage:   "output sql folder",
		Default: "./internal/pkg/db/",
	}}

var Entrypoint = common.NewToolEntrypoint(common.ToolEntrypointOption{
	ToolName:       "sheet2mysql",
	Use:            "sheet2mysql",
	Aliases:        []string{"genSheetDDL"},
	Short:          "读取 APITable 表结构并生成 MySQL 建表 SQL",
	CacheKeyPrefix: cacheKeyPrefix,
	ParamList:      paramSettingList,
	Run:            Run,
})

// --------------------------------------------------
// 运行参数，定义的参数列表最终转换成当前程序使用的运行选项

type RunOptions struct {
	Token       string
	BaseURL     string
	SpaceID     string
	DatasheetID string
	TableName   string
	OutFolder   string
}

// cobra参数值转换为“当前程序的”运行选项
func convParamToRunOpt(values common.ParamMap) (RunOptions, error) {
	datasheetID := strings.TrimSpace(values[paramNameDatasheetID])
	if true { // just for test
		datasheetID = "dstaBbsMatBLqc84Bh"
	}

	// --------------------------------------------------
	// 优先使用参数值，命令行参数默认为空
	// 未显式输入时，从配置文件读取
	// 都为空时返回失败
	baseURL := strings.TrimSpace(values[paramNameBaseURL])
	spaceID := strings.TrimSpace(values[paramNameSpaceID])
	token := strings.TrimSpace(values[paramNameToken])

	configPath := strings.TrimSpace(values[paramNameConfig])
	if configPath != "" && (baseURL == "" || spaceID == "" || token == "") {
		if loadErr := pconfig.InitConfig(configPath); loadErr == nil {
			if baseURL == "" {
				baseURL = strings.TrimSpace(pconfig.GetStringD("APITable.baseUrl", ""))
			}
			if spaceID == "" {
				spaceID = strings.TrimSpace(pconfig.GetStringD("APITable.spaceId", ""))
			}
			if token == "" {
				token = strings.TrimSpace(pconfig.GetStringD("APITable.token", ""))
			}
		}
	}
	// --------------------------------------------------

	if baseURL == "" || spaceID == "" || token == "" || datasheetID == "" {
		return RunOptions{}, errors.New("baseURL, spaceID, token, and datasheetID must be provided either through parameters or config file")
	}

	return RunOptions{
		Token:       token,
		BaseURL:     baseURL,
		SpaceID:     spaceID,
		DatasheetID: datasheetID,
		TableName:   strings.TrimSpace(values[paramNameTableName]),
		OutFolder:   strings.TrimSpace(values[paramNameOutFolder]),
	}, nil
}

// --------------------------------------------------
func Run(values common.ParamMap) error {
	options, err := convParamToRunOpt(values)
	if err != nil {
		return err
	}

	if options.DatasheetID == "" {
		return errors.New("datasheetId is empty")
	}
	if options.TableName == "" {
		return errors.New("tableName is empty: 请填写 APITable 页面显示名（表名无法自动获取）")
	}
	if options.Token == "" {
		return errors.New("token is empty: 请在 config 配置文件中填写 APITable.token 或通过工具参数填写")
	}

	// 提前解析最终表名，用于生成输出文件路径
	resolvedTableName, err := buildTableName(options.TableName)
	if err != nil {
		return fmt.Errorf("resolve table name failed: %w", err)
	}

	outFolder := options.OutFolder
	if outFolder == "" {
		outFolder = "./internal/pkg/db/"
	}
	outFile := filepath.Join(outFolder, resolvedTableName+".sql")

	err = papitable.InitAPITable(options.Token, options.BaseURL)
	if err != nil {
		return fmt.Errorf("init apitable failed: %w", err)
	}

	doc := papitable.NewMultiTableDoc(options.SpaceID, options.DatasheetID)
	fieldList, err := doc.GetCols()
	if err != nil {
		return fmt.Errorf("get sheet fields failed: %w", err)
	}

	builderOpt := BuildDDLOptions{
		DatasheetID:   options.DatasheetID,
		SourceComment: buildSourceComment(options.BaseURL, options.SpaceID, options.DatasheetID),
	}

	ddlSQL, warningList, err := BuildMysqlCreateTableSQL(options.TableName, fieldList, builderOpt)
	if err != nil {
		return err
	}

	err = os.MkdirAll(outFolder, 0755)
	if err != nil {
		return fmt.Errorf("create out dir failed: %w", err)
	}

	err = os.WriteFile(outFile, []byte(ddlSQL), 0644)
	if err != nil {
		return fmt.Errorf("write sql file failed: %w", err)
	}

	putil.Interact.Infof("已生成 MySQL 建表 SQL: %s", outFile)
	putil.Interact.Infof("字段总数: %d", len(fieldList))

	if len(warningList) > 0 {
		putil.Interact.Warnf("生成完成，但有 %d 条提示:", len(warningList))
		for _, warning := range warningList {
			putil.Interact.Warnf("- %s", warning)
		}
	}

	return nil
}

func buildSourceComment(baseURL string, spaceID string, datasheetID string) string {
	source := strings.TrimSpace(baseURL)
	source = strings.TrimPrefix(source, "http://")
	source = strings.TrimPrefix(source, "https://")
	source = strings.TrimRight(source, "/")

	if strings.TrimSpace(spaceID) != "" {
		return fmt.Sprintf("%s/%s/%s", source, strings.TrimSpace(spaceID), strings.TrimSpace(datasheetID))
	}

	if strings.TrimSpace(datasheetID) != "" {
		return fmt.Sprintf("%s/%s", source, strings.TrimSpace(datasheetID))
	}

	return source
}
