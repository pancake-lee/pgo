package service

import (
	"fmt"
	"gogogo/pkg/util"
	"log"
	"sort"
	"testing"
	"time"
)

// 实在词穷，courseInfo表示课程安排，如20240101一年1班的第1节课
// 而classInfo表示课程表中的一节课，如周三一年1班的第1节课
type courseInfo struct {
	className     string //课名
	classRoomName string //班级名
	classNum      int
	date          time.Time
	teacher       string
}

func TestCourseSwap(t *testing.T) {
	courseNumMax := 7

	log.Print("请输入老师名字，不要输入空格等额外内容，以回车结束")
	var srcTecher string
	_, _ = fmt.Scanln(&srcTecher)

	log.Print("请输入日期，如20240101")
	var srcDateStr string
	_, _ = fmt.Scanln(&srcDateStr)

	log.Print("请输入第几节课，1~7")
	var srcCourseNum int
	_, _ = fmt.Scanln(&srcCourseNum)

	if srcTecher == "" || srcDateStr == "" || srcCourseNum == 0 {
		t.Error("input error")
		return
	}

	srcDate, err := time.Parse("20060102", srcDateStr)
	if err != nil {
		t.Error("time.Parse failed: ", err)
		return
	}

	// 1：从excel读取课程表
	fPath := `/root/workspace/class_schedule.xlsx`
	courseMap, err := parseCourseExcel(fPath)
	if err != nil {
		t.Error("parseCourseExcel failed: ", err)
		return
	}

	tNow := time.Now()
	wDiff := tNow.Weekday() - time.Monday
	endTime := tNow.AddDate(0, 0, 21-int(wDiff))

	var allCourseList []*courseInfo

	for _, techerInfo := range courseMap {
		for _, classInfo := range techerInfo.classList {
			// 用课程表的一节课，计算未来3周内的课程安排
			date := tNow.AddDate(0, 0, int(classInfo.weekDay-tNow.Weekday()))

			for date.Before(endTime) {
				if date.Before(tNow) {
					continue
				}

				var c courseInfo
				c.className = classInfo.className
				c.classRoomName = classInfo.classRoomName
				c.classNum = classInfo.classNum
				c.date = date
				c.teacher = techerInfo.teacher
				allCourseList = append(allCourseList, &c)

				date = date.AddDate(0, 0, 7)
			}
		}
	}
	sort.Slice(allCourseList, func(i, j int) bool {
		return allCourseList[i].date.Before(allCourseList[j].date)
	})

	// 3：获取当前需要调课的课程，则某老师某天的某节课
	srcCourse := getCourse(allCourseList,
		srcTecher, srcDate, srcCourseNum)
	if srcCourse == nil {
		t.Error("srcCourse not found")
		return
	} else {
		logCourse(srcCourse)
	}
	srcClassRoom := srcCourse.classRoomName

	// 这节课的时间，不用上课的，同班老师
	var srcFreeTeacherList []string
	teacherList := getTeacherListByClassRoom(allCourseList, srcClassRoom)
	for _, t := range teacherList {
		c := getCourse(allCourseList, t, srcDate, srcCourseNum)
		if c == nil {
			log.Printf("teacher[%v] is free", t)
			srcFreeTeacherList = append(srcFreeTeacherList, t)
		}
	}

	// 当前老师未来有空的时间
	// 直接遍历时间
	var dstFreeCourseList []*courseInfo //只用来记一下哪天第几节
	for date := time.Now(); date.Before(endTime); date = date.AddDate(0, 0, 1) {
		for courseNum := 1; courseNum <= courseNumMax; courseNum++ {
			c := getCourse(allCourseList, srcTecher, date, courseNum)
			if c != nil {
				continue
			}
			dstFreeCourseList = append(dstFreeCourseList,
				&courseInfo{
					date:     date,
					classNum: courseNum,
				})
		}
	}

	// 当前课堂有空的老师，在当前老师未来有空的时间，对应的课程
	var dstCourseList []*courseInfo
	for _, dstFreeCourse := range dstFreeCourseList {
		for _, t := range srcFreeTeacherList {
			dstCourse := getCourse(allCourseList, t, dstFreeCourse.date, dstFreeCourse.classNum)
			if dstCourse == nil ||
				dstCourse.classRoomName != srcClassRoom { //换同班的课
				continue
			}
			dstCourseList = append(dstCourseList, dstCourse)
		}
	}

	logCourseList(dstCourseList)

	// TODO 查询当前换课记录，结合换课记录来计算
	// GetCourseSwapRequestList

	// TODO 写入换课记录到DB
	// AddCourseSwapRequest

}
func getTeacherListByClassRoom(courseList []*courseInfo,
	classRoom string) []string {
	var retList []string
	for _, c := range courseList {
		if c.classRoomName == classRoom {
			retList = append(retList, c.teacher)
		}
	}
	retList = util.StrListUnique(retList)
	return retList
}

func getAllTeacherList(courseList []*courseInfo) []string {
	var retList []string
	for _, c := range courseList {
		retList = append(retList, c.teacher)
	}
	retList = util.StrListUnique(retList)
	return retList
}

func getCourseByDateAndNum(courseList []*courseInfo,
	t time.Time, courseNum int) *courseInfo {
	for _, c := range courseList {
		if c.date.Format("20160102") == t.Format("20160102") &&
			c.classNum == courseNum {
			return c
		}
	}
	return nil
}

func getCourseByTeacher(courseList []*courseInfo,
	teacher string) []*courseInfo {
	var retList []*courseInfo
	for _, c := range courseList {
		if c.teacher == teacher {
			retList = append(retList, c)
		}
	}
	return retList
}

func getCourse(courseList []*courseInfo,
	teacher string, t time.Time, courseNum int) *courseInfo {
	for _, c := range courseList {
		if c.date.Format("20160102") == t.Format("20160102") &&
			c.classNum == courseNum &&
			c.teacher == teacher {
			return c
		}
	}
	return nil
}

func logCourseList(courseList []*courseInfo) {
	for _, c := range courseList {
		logCourse(c)
	}
}
func logCourse(course *courseInfo) {
	log.Printf("course[%v][%v][周%v][%v][%v][%v]",
		course.teacher,
		course.date.Format("060102"), course.date.Weekday(),
		course.classNum, course.classRoomName, course.className)
}
