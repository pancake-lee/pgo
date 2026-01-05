package courseSwap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

type InputConfig struct {
	Path        string `json:"path"`
	Teacher     string `json:"teacher"`
	Date        string `json:"date"`
	CourseNum   int    `json:"courseNum"`
	StorageType string `json:"storageType"`
	IsOddWeek   bool   `json:"isOddWeek"`
}

func getCachePath() string {
	path := "./"
	if false {
		home, err := os.UserHomeDir()
		if err != nil {
			return "course_swap_cache.json"
		}
		path := filepath.Join(home, "pgo")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_ = os.MkdirAll(path, 0755)
		}
	}

	return filepath.Join(path, "course_swap_cache.json")
}

func LoadCache() InputConfig {
	var cache InputConfig
	data, err := os.ReadFile(getCachePath())
	if err == nil {
		_ = json.Unmarshal(data, &cache)
	}
	return cache
}

func SaveCache(cache InputConfig) {
	data, err := json.MarshalIndent(cache, "", "  ")
	if err == nil {
		_ = os.WriteFile(getCachePath(), data, 0644)
	}
}

// --------------------------------------------------

func InputParams() (config InputConfig, err error) {
	cache := LoadCache()

	// Path
	prompt := "请输入需要导入的课程表文件(excel)"
	if cache.Path != "" {
		prompt += fmt.Sprintf(" (默认: %s)", cache.Path)
	}
	prompt += "，以回车结束"
	config.Path = putil.Interact.Input(prompt)
	if config.Path == "" {
		config.Path = cache.Path
	}
	for config.Path == "" {
		config.Path = putil.Interact.MustInput("请输入需要导入的课程表文件(excel)，以回车结束")
	}

	// Teacher
	prompt = "请输入老师名字"
	if cache.Teacher != "" {
		prompt += fmt.Sprintf(" (默认: %s)", cache.Teacher)
	}
	prompt += "，不要输入空格等额外内容，以回车结束"
	config.Teacher = putil.Interact.Input(prompt)
	if config.Teacher == "" {
		config.Teacher = cache.Teacher
	}
	for config.Teacher == "" {
		config.Teacher = putil.Interact.MustInput("请输入老师名字，不要输入空格等额外内容，以回车结束")
	}

	// Date
	prompt = "请输入日期，如20240101"
	if cache.Date != "" {
		prompt += fmt.Sprintf(" (默认: %s)", cache.Date)
	}
	prompt += "，以回车结束"
	config.Date = putil.Interact.Input(prompt)
	if config.Date == "" {
		config.Date = cache.Date
	}
	for config.Date == "" {
		config.Date = putil.Interact.MustInput("请输入日期，如20240101，以回车结束")
	}

	// CourseNum
	prompt = "请输入第几节课，1~7"
	if cache.CourseNum != 0 {
		prompt += fmt.Sprintf(" (默认: %d)", cache.CourseNum)
	}
	prompt += "，以回车结束"
	val := putil.Interact.Input(prompt)
	if val == "" && cache.CourseNum != 0 {
		config.CourseNum = cache.CourseNum
	} else if val != "" {
		config.CourseNum, _ = putil.StrToInt(val)
	}
	for config.CourseNum == 0 {
		val := putil.Interact.MustInput("请输入第几节课，1~7，以回车结束")
		config.CourseNum, _ = putil.StrToInt(val)
	}

	// StorageType
	prompt = "请输入存储类型(Cloud/Local)"
	defaultStorage := "Local"
	if cache.StorageType != "" {
		defaultStorage = cache.StorageType
	}
	prompt += fmt.Sprintf(" (默认: %s)", defaultStorage)
	config.StorageType = putil.Interact.Input(prompt)
	if config.StorageType == "" {
		config.StorageType = defaultStorage
	}

	// IsOddWeek
	prompt = "本周是否为单周(y/n)"
	defaultOdd := "n"
	if cache.IsOddWeek {
		defaultOdd = "y"
	}
	prompt += fmt.Sprintf(" (默认: %s)", defaultOdd)

	oddStr := putil.Interact.Input(prompt)
	if oddStr == "" {
		config.IsOddWeek = cache.IsOddWeek
	} else {
		if oddStr == "y" || oddStr == "Y" || oddStr == "yes" {
			config.IsOddWeek = true
		} else {
			config.IsOddWeek = false
		}
	}

	// Save cache
	SaveCache(config)

	if config.Teacher == "" || config.Date == "" || config.CourseNum == 0 {
		plogger.Debug("input error")
		err = fmt.Errorf("input error")
		return
	}

	_, err = putil.TimeFromStr(config.Date, "YYYYMMDD")
	if err != nil {
		plogger.Debug("time.Parse failed: ", err)
		return
	}
	return
}
