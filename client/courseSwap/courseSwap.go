package courseSwap

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pancake-lee/pgo/client/swagger"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func CourseSwapCli() {
	config, err := InputParams()
	if err != nil {
		return
	}
	CourseSwap(config)
}

func GetTeacherList(path string) ([]string, error) {
	courseMap, err := NewCourseParser(path).ParseCourseExcel()
	if err != nil {
		return nil, err
	}
	var teachers []string
	for t := range courseMap {
		teachers = append(teachers, t)
	}
	sort.Strings(teachers)
	return teachers, nil
}

func CourseSwap(config InputConfig) {
	allCourseMgr, err := GetAllCourseList(config)
	if err != nil {
		return
	}

	dstCourseMgr, err := getSwapCandidates(allCourseMgr, config)
	if err != nil {
		return
	}

	err = handleSwapSelection(dstCourseMgr, config)
	if err != nil {
		return
	}
}

// 实在词穷，courseInfo表示课程安排，如20240101一年1班的第1节课
// 而classInfo表示课程表中的一节课，如周三一年1班的第1节课
type courseInfo struct {
	className     string //课名
	classRoomName string //班级名
	classNum      int
	date          time.Time
	teacher       string
}

func GetAllCourseList(config InputConfig) (*courseManager, error) {
	courseMap, err := NewCourseParser(config.Path).ParseCourseExcel()
	if err != nil {
		plogger.Debug("parseCourseExcel failed: ", err)
		return nil, err
	}
	plogger.Debugf("从excel读取课程表[%v] 共[%v]个课程安排",
		config.Path, len(courseMap))

	tNow := time.Now()
	wDiff := tNow.Weekday() - time.Monday
	endWeek := 3
	endTime := tNow.AddDate(0, 0, 7*endWeek-int(wDiff))

	plogger.Debugf("用课程表，计算未来[%v]周内的课程安排[%v]-[%v]",
		endWeek,
		putil.TimeToStr(tNow, "YYYYMMDD"),
		putil.TimeToStr(endTime, "YYYYMMDD"))

	var allCourseList []*courseInfo
	for _, tInfo := range courseMap {
		// plogger.Debugf("teacher[%v] class cnt[%v]", tInfo.teacher, len(tInfo.classList))
		for _, classInfo := range tInfo.classList {
			// 一节课向后推3周
			date := tNow.AddDate(0, 0, int(classInfo.weekDay-tNow.Weekday()))
			for date.Before(endTime) {
				if date.Before(tNow) {
					date = date.AddDate(0, 0, 7)
					continue
				}

				var c courseInfo
				c.className = classInfo.className
				c.classRoomName = classInfo.classRoomName
				c.classNum = classInfo.classNum
				c.date = date
				c.teacher = tInfo.teacher
				allCourseList = append(allCourseList, &c)

				date = date.AddDate(0, 0, 7)
			}
		}
	}

	plogger.Debug("查询当前换课记录，结合换课记录来计算")
	repo := getRepo(config.StorageType)

	{ // GetCourseSwapRequestList
		reqList, err := repo.GetCourseSwapRequestList(context.Background())
		if err != nil {
			plogger.Debug("GetCourseSwapRequestList failed: ", err)
			return nil, err
		}

		sort.Slice(reqList, func(i, j int) bool {
			return reqList[i].CreateTime < reqList[j].CreateTime
		})

		for _, req := range reqList {
			// allCourseList 中找到src课程
			mgr := newCourseManager(allCourseList)
			srcTime, _ := putil.TimeFromStr("YYYYMMDD", req.SrcDate)
			srcCourse := mgr.getCourse(req.SrcTeacher, srcTime, int(req.SrcCourseNum))
			// allCourseList 中找到dst课程
			dstTime, _ := putil.TimeFromStr("YYYYMMDD", req.DstDate)
			dstCourse := mgr.getCourse(req.DstTeacher, dstTime, int(req.DstCourseNum))
			// 交换老师和课名
			tmpTeacher := srcCourse.teacher
			tmpClassName := srcCourse.className
			srcCourse.teacher = dstCourse.teacher
			srcCourse.className = dstCourse.className
			dstCourse.teacher = tmpTeacher
			dstCourse.className = tmpClassName
		}
	}

	sort.Slice(allCourseList, func(i, j int) bool {
		return allCourseList[i].date.Before(allCourseList[j].date)
	})
	return newCourseManager(allCourseList), nil
}

var courseNumMax = 7

func getSwapCandidates(mgr *courseManager, config InputConfig) (*courseManager, error) {
	srcDate, _ := putil.TimeFromStr(config.Date, "YYYYMMDD")
	tNow := time.Now()
	wDiff := tNow.Weekday() - time.Monday
	endTime := tNow.AddDate(0, 0, 21-int(wDiff))

	// 获取当前需要调课的课程，则某老师某天的某节课
	srcCourse := mgr.getCourse(config.Teacher, srcDate, config.CourseNum)
	if srcCourse == nil {
		plogger.Debug("srcCourse not found")
		return nil, fmt.Errorf("srcCourse not found")
	} else {
		logCourse(srcCourse)
	}
	srcClassRoom := srcCourse.classRoomName
	srcDateStr := putil.TimeToStr(srcDate, "YYYYMMDD")

	plogger.Debugf("--------------------------------------------------")
	plogger.Debugf("当前输入为[%v][%v][第%v节]，班级为[%v]",
		config.Teacher, srcDateStr, config.CourseNum, srcClassRoom)
	plogger.Debugf("下面找到这个时间不用上课的，候选列表如下:")
	var srcFreeTeacherList []string
	teacherList := mgr.getTeacherListByClassRoom(srcClassRoom)
	for _, t := range teacherList {
		c := mgr.getCourse(t, srcDate, config.CourseNum)
		if c == nil {
			plogger.Debugf("teacher[%v] is free", t)
			srcFreeTeacherList = append(srcFreeTeacherList, t)
		}
	}

	// 当前老师未来有空的时间
	// 直接遍历时间
	var dstFreeCourseList []*courseInfo //只用来记一下哪天第几节
	for date := time.Now(); date.Before(endTime); date = date.AddDate(0, 0, 1) {
		for courseNum := 1; courseNum <= courseNumMax; courseNum++ {
			c := mgr.getCourse(config.Teacher, date, courseNum)
			if c != nil {
				continue
			}
			dstFreeCourseList = append(dstFreeCourseList,
				&courseInfo{date: date, classNum: courseNum})
		}
	}
	plogger.Debugf("--------------------------------------------------")
	plogger.Debugf("找到[%v]未来有空上的目标课程，并且对应老师在[%v][第%v节]有空",
		config.Teacher, srcDateStr, config.CourseNum)
	var dstCourseList []*courseInfo
	for _, dstFreeCourse := range dstFreeCourseList {
		for _, t := range srcFreeTeacherList {
			dstCourse := mgr.getCourse(t, dstFreeCourse.date, dstFreeCourse.classNum)
			if dstCourse == nil ||
				dstCourse.classRoomName != srcClassRoom { //换同班的课
				continue
			}
			dstCourseList = append(dstCourseList, dstCourse)
		}
	}
	return newCourseManager(dstCourseList), nil
}

func handleSwapSelection(mgr *courseManager, config InputConfig) error {
	mgr.logCourseList()

	if true {
		return nil
	}
	// AddCourseSwapRequest
	repo := getRepo(config.StorageType)
	err := repo.AddCourseSwapRequest(context.Background(), &swagger.ApiCourseSwapRequestInfo{
		SrcTeacher: config.Teacher,
	})
	if err != nil {
		plogger.Debug("AddCourseSwap failed: ", err)
		return err
	}
	plogger.Debugf("new course swap added")
	return nil
}

// --------------------------------------------------
func logCourse(course *courseInfo) {
	teacher := course.teacher
	if len([]rune(teacher)) == 2 {
		rs := []rune(teacher)
		teacher = string(rs[0]) + "  " + string(rs[1])
	}
	putil.Interact.Infof(
		// plogger.Debugf(
		"course [%v] [%v] [第%v节] [%v]班 [%v] [%v]课",
		course.date.Format("060102"),
		getWeekday(course.date.Weekday()),
		course.classNum, course.classRoomName,
		teacher, course.className)
}

var weekDayMap = map[time.Weekday]string{
	time.Monday:    "周一",
	time.Tuesday:   "周二",
	time.Wednesday: "周三",
	time.Thursday:  "周四",
	time.Friday:    "周五",
	time.Saturday:  "周六",
	time.Sunday:    "周日",
}

func getWeekday(w time.Weekday) string {
	return weekDayMap[w]
}

// --------------------------------------------------
func getRepo(storageType string) CourseSwapRepo {
	if storageType == "Cloud" {
		return NewCloudRepo()
	}
	return NewLocalRepo()
}
