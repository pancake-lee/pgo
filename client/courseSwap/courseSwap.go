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
	mgr, err := CalculateSwapCandidates(config)
	if err != nil {
		return
	}

	courses := mgr.GetCourses()
	if len(courses) == 0 {
		plogger.Infof("No swap candidates found.")
		return
	}

	// Sort for consistent display
	sort.Slice(courses, func(i, j int) bool {
		return courses[i].ClassName < courses[j].ClassName
	})

	for i, c := range courses {
		fmt.Printf("[%d] ", i)
		logCourse(c)
	}

	fmt.Printf("Please enter the index of the course to swap (0-%d): ", len(courses)-1)
	var index int
	_, err = fmt.Scanf("%d", &index)
	if err != nil || index < 0 || index >= len(courses) {
		plogger.Errorf("Invalid input")
		return
	}

	selected := courses[index]
	plogger.Infof("Selected: ")
	logCourse(selected)

	// Confirm
	fmt.Printf("Confirm swap? (y/n): ")
	var confirm string
	fmt.Scanf("%s", &confirm)
	if confirm != "y" && confirm != "Y" {
		plogger.Infof("Cancelled")
		return
	}

	err = ExecuteSwap(config, selected)
	if err != nil {
		plogger.Errorf("Swap failed: %v", err)
	} else {
		plogger.Infof("Swap successful!")
	}
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

func CalculateSwapCandidates(config InputConfig) (*courseManager, error) {
	allCourseMgr, err := GetAllCourseList(config)
	if err != nil {
		return nil, err
	}

	dstCourseMgr, err := getSwapCandidates(allCourseMgr, config)
	if err != nil {
		return nil, err
	}
	return dstCourseMgr, nil
}

func ExecuteSwap(config InputConfig, target *CourseInfo) error {
	// AddCourseSwapRequest
	repo := getRepo(config.StorageType)
	err := repo.AddCourseSwapRequest(context.Background(), &swagger.ApiCourseSwapRequestInfo{
		SrcTeacher:   config.Teacher,
		SrcDate:      config.Date,
		SrcCourseNum: int32(config.CourseNum),
		DstTeacher:   target.Teacher,
		DstDate:      putil.TimeToStr(target.Date, "YYYYMMDD"),
		DstCourseNum: int32(target.ClassNum),
	})
	if err != nil {
		plogger.Debug("AddCourseSwap failed: ", err)
		return err
	}
	plogger.Debugf("new course swap added")
	return nil
}

// 实在词穷，courseInfo表示课程安排，如20240101一年1班的第1节课
// 而classInfo表示课程表中的一节课，如周三一年1班的第1节课
type CourseInfo struct {
	ClassName     string //课名
	ClassRoomName string //班级名
	ClassNum      int
	Date          time.Time
	Teacher       string
}

func (c *CourseInfo) String() string {
	teacher := c.Teacher
	if len([]rune(teacher)) == 2 {
		rs := []rune(teacher)
		teacher = string(rs[0]) + "  " + string(rs[1])
	}

	return fmt.Sprintf(
		"[%v] [%v] [第%v节] [%v]班 [%v] [%v]课",
		c.Date.Format("060102"),
		getWeekday(c.Date.Weekday()),
		c.ClassNum, c.ClassRoomName,
		teacher, c.ClassName)
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

	var allCourseList []*CourseInfo
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

				var c CourseInfo
				c.ClassName = classInfo.className
				c.ClassRoomName = classInfo.classRoomName
				c.ClassNum = classInfo.classNum
				c.Date = date
				c.Teacher = tInfo.teacher
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
			if srcCourse != nil && dstCourse != nil {
				tmpTeacher := srcCourse.Teacher
				tmpClassName := srcCourse.ClassName
				srcCourse.Teacher = dstCourse.Teacher
				srcCourse.ClassName = dstCourse.ClassName
				dstCourse.Teacher = tmpTeacher
				dstCourse.ClassName = tmpClassName
			}
		}
	}

	sort.Slice(allCourseList, func(i, j int) bool {
		return allCourseList[i].Date.Before(allCourseList[j].Date)
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
	}
	if srcCourse.ClassRoomName == "" {
		return nil, plogger.LogErrfMsg(
			"srcCourse[%v] ClassRoomName is empty", srcCourse)
	}

	logCourse(srcCourse)

	srcClassRoom := srcCourse.ClassRoomName
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
	var dstFreeCourseList []*CourseInfo //只用来记一下哪天第几节
	for date := time.Now(); date.Before(endTime); date = date.AddDate(0, 0, 1) {
		for courseNum := 1; courseNum <= courseNumMax; courseNum++ {
			c := mgr.getCourse(config.Teacher, date, courseNum)
			if c != nil {
				continue
			}
			dstFreeCourseList = append(dstFreeCourseList,
				&CourseInfo{Date: date, ClassNum: courseNum})
		}
	}
	plogger.Debugf("--------------------------------------------------")
	plogger.Debugf("找到[%v]未来有空上的目标课程，并且对应老师在[%v][第%v节]有空",
		config.Teacher, srcDateStr, config.CourseNum)
	var dstCourseList []*CourseInfo
	for _, dstFreeCourse := range dstFreeCourseList {
		for _, t := range srcFreeTeacherList {
			dstCourse := mgr.getCourse(t, dstFreeCourse.Date, dstFreeCourse.ClassNum)
			if dstCourse == nil ||
				dstCourse.ClassRoomName != srcClassRoom { //换同班的课
				continue
			}
			dstCourseList = append(dstCourseList, dstCourse)
		}
	}
	return newCourseManager(dstCourseList), nil
}

func handleSwapSelection(mgr *courseManager, config InputConfig) error {
	// Deprecated: Logic moved to CourseSwapCli and ExecuteSwap
	return nil
}

// --------------------------------------------------
func logCourse(course *CourseInfo) {
	putil.Interact.Infof("%v", course)
	// plogger.Debugf("%v", course)
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
