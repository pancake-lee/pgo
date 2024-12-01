package service

import (
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

func parseCourseExcel(path string,
) (ret map[string]*teacherInfo, err error) {
	ret = make(map[string]*teacherInfo)

	f, err := excelize.OpenFile(path)
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
		parseCourseSheet(rowList, ret)
	}
	return
}
func parseCourseSheet(rowList [][]string, ret map[string]*teacherInfo,
) (err error) {

	for row, colList := range rowList {
		// log.Print("row[", row, "] col size : ", len(rowList))
		for col, cellStr := range colList {
			if !isTecherInfoStart(cellStr) {
				continue
			}
			//找到了一个老师的课表位置
			tInfo := getTecherInfo(row, col, rowList)
			_, ok := ret[tInfo.teacher]
			if !ok {
				ret[tInfo.teacher] = &tInfo
			}
			// 同一个老师课表分开写了，要合并
			ret[tInfo.teacher].classList =
				append(ret[tInfo.teacher].classList,
					tInfo.classList...)
		}
	}
	return
}

// ------------------------------------------------------------
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

func isTecherInfoStart(cellStr string) bool {
	return cellStr == "节次"
}

// 老老实实的硬编码，根据excel的格式而定
func getTecherInfo(rowStart int, colStart int, rowList [][]string) (ret teacherInfo) {
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
	// log.Printf("找到一个老师 [%v] 课程有 : %v", tInfo.teacher, logStr)
	// log.Printf("")
	return tInfo
}
