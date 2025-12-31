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
				key := fmt.Sprintf("%v-%v-%v", c.teacher, c.weekDay, c.classNum)
				if _, ok := keySet[key]; ok {
					return plogger.LogErrMsg("conflicting course schedules")
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
type classInfo struct {
	className     string //课名
	classRoomName string //班级名
	classNum      int
	weekDay       time.Weekday
	teacher       string
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
	for wDay := time.Monday; wDay <= time.Friday; wDay++ {
		for rowAdd := 1; rowAdd <= 8; rowAdd++ {
			//567是下午，表格中中午隔了一行
			classNum := rowAdd
			if rowAdd == 5 {
				continue
			} else if rowAdd > 5 {
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
			if classCol == "" {
				continue
			}
			var cInfo classInfo

			classColSplitList := strings.Split(classCol, "班")
			if len(classColSplitList) != 2 {
				cInfo.className = classCol
				cInfo.classRoomName = ""
			} else {
				cInfo.className = classCol
				cInfo.classRoomName = classColSplitList[0]
			}
			cInfo.classNum = classNum
			cInfo.teacher = tInfo.teacher
			cInfo.weekDay = wDay
			logStr += classCol + ","
			tInfo.classList = append(tInfo.classList, cInfo)
		}
	}
	// plogger.Debugf("找到一个老师 [%v] 课程有[%v]节", tInfo.teacher, len(tInfo.classList))
	// plogger.Debugf("找到一个老师 [%v] 课程有 : %v", tInfo.teacher, logStr)
	// plogger.Debugf("")
	return tInfo
}
