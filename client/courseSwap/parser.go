package courseSwap

import (
	"fmt"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/xuri/excelize/v2"
)

type courseParser struct {
	path           string
	teacherInfoMap map[string]*teacherInfo
}

func NewCourseParser(path string) *courseParser {
	return &courseParser{path: path, teacherInfoMap: make(map[string]*teacherInfo)}
}

func (parser *courseParser) ParseCourseExcel() (map[string]*teacherInfo, error) {
	f, err := excelize.OpenFile(parser.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sNameList := f.GetSheetList()
	for _, sheetName := range sNameList {
		rowList, err := f.GetRows(sheetName)
		if err != nil {
			return nil, err
		}
		parser.parseCourseSheet(rowList)
	}
	return parser.teacherInfoMap, nil
}

func (parser *courseParser) parseCourseSheet(rowList [][]string) (err error) {
	// 人+周几+节次 是唯一的
	keySet := make(map[string]bool)
	for row, colList := range rowList {
		// plogger.Debug("row[", row, "] col size : ", len(rowList))
		for col, cellStr := range colList {
			if !parser.isTeacherInfoStart(cellStr) {
				continue
			}
			//找到了一个老师的课表位置
			tInfo := parser.getTeacherInfo(row, col, rowList)
			for _, c := range tInfo.classList {
				key := fmt.Sprintf("%v-%v-%v-%v", c.teacher, c.weekDay, c.classNum, c.weekType)
				if _, ok := keySet[key]; ok {
					return plogger.LogErrfMsg("conflicting course schedules [%v]", key)
				}
				keySet[key] = true
			}

			_, ok := parser.teacherInfoMap[tInfo.teacher]
			if !ok {
				parser.teacherInfoMap[tInfo.teacher] = &tInfo
				continue
			}
			// 同一个老师课表分开写了，要合并
			parser.teacherInfoMap[tInfo.teacher].classList =
				append(parser.teacherInfoMap[tInfo.teacher].classList,
					tInfo.classList...)
		}
	}
	return
}

// --------------------------------------------------
const (
	WeekTypeAll = iota
	WeekTypeOdd
	WeekTypeEven
)

type classInfo struct {
	className     string //课名
	classRoomName string //班级名
	classNum      int
	weekDay       time.Weekday
	teacher       string
	weekType      int // 0:All, 1:Odd, 2:Even
}
type teacherInfo struct {
	teacher   string
	classList []classInfo
}

func (parser *courseParser) isTeacherInfoStart(cellStr string) bool {
	return cellStr == "节次"
}

// 老老实实的硬编码，根据excel的格式而定
func (parser *courseParser) getTeacherInfo(rowStart int, colStart int,
	rowList [][]string) (ret teacherInfo) {

	nameCell := rowList[rowStart-1][colStart]
	nameTmp := strings.Split(nameCell, " ")
	var tInfo teacherInfo
	tInfo.teacher = nameTmp[0]

	//循环每节课的cell
	logStr := ""
	rowAddMax := 8 + 1 //多加一行是因为中午有一行空行
	emptyRowAdd := 6   //第六行是空行
	for wDay := time.Monday; wDay <= time.Friday; wDay++ {
		for rowAdd := 1; rowAdd <= rowAddMax; rowAdd++ {
			classNum := rowAdd
			if rowAdd == emptyRowAdd {
				continue
			} else if rowAdd > emptyRowAdd {
				classNum = rowAdd - 1
			}
			rowTmp := rowStart + rowAdd
			colTmp := colStart + int(wDay)
			if rowTmp >= len(rowList) || colTmp >= len(rowList[rowTmp]) {
				continue
			}
			classCol := rowList[rowTmp][colTmp]
			classCol = strings.ReplaceAll(classCol, "\n", "")
			classCol = strings.ReplaceAll(classCol, "\r", "")
			classCol = strings.ReplaceAll(classCol, " ", "")
			if classCol == "" || classCol == "-" {
				continue
			}

			parseClassStr := func(str string) classInfo {
				var cInfo classInfo
				classColSplitList := strings.Split(str, "班")
				if len(classColSplitList) != 2 {
					cInfo.className = str
					cInfo.classRoomName = ""
				} else {
					cInfo.className = str
					cInfo.classRoomName = classColSplitList[0]
				}
				return cInfo
			}
			// plogger.Debugf("classCol : %v", classCol)
			var cInfoList []classInfo
			if strings.HasPrefix(classCol, "单") && strings.Contains(classCol, "双") {
				parts := strings.Split(classCol, "双")
				if len(parts) == 2 {
					oddStr := strings.TrimPrefix(parts[0], "单")
					evenStr := parts[1]
					// plogger.Debugf("oddStr : %v", oddStr)
					// plogger.Debugf("evenStr : %v", evenStr)

					evenInfo := parseClassStr(evenStr)
					evenInfo.weekType = WeekTypeEven
					// plogger.Debugf("evenInfo : %v", evenInfo)

					// Try to extract course name from evenStr to complete oddStr
					courseName := ""
					if evenInfo.classRoomName != "" {
						prefix := evenInfo.classRoomName + "班"
						if strings.HasPrefix(evenStr, prefix) {
							courseName = strings.TrimPrefix(evenStr, prefix)
						}
					}
					if courseName != "" && strings.HasSuffix(oddStr, "班") {
						oddStr += courseName
					}

					oddInfo := parseClassStr(oddStr)
					oddInfo.weekType = WeekTypeOdd
					// plogger.Debugf("oddInfo : %v", oddInfo)

					cInfoList = append(cInfoList, oddInfo, evenInfo)
				} else {
					info := parseClassStr(classCol)
					info.weekType = WeekTypeAll
					cInfoList = append(cInfoList, info)
				}
			} else {
				info := parseClassStr(classCol)
				info.weekType = WeekTypeAll
				cInfoList = append(cInfoList, info)
			}

			for _, cInfo := range cInfoList {
				cInfo.classNum = classNum
				cInfo.teacher = tInfo.teacher
				cInfo.weekDay = wDay
				logStr += cInfo.className + ","
				tInfo.classList = append(tInfo.classList, cInfo)
			}
		}
	}
	// plogger.Debugf("找到一个老师 [%v] 课程有[%v]节", tInfo.teacher, len(tInfo.classList))
	// plogger.Debugf("找到一个老师 [%v] 课程有 : %v", tInfo.teacher, logStr)
	// plogger.Debugf("")
	return tInfo
}
