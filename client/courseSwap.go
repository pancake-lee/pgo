package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"pgo/client/swagger"
	"pgo/pkg/config"
	"pgo/pkg/util"
	"sort"
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

var courseNumMax = 7
var cli *swagger.APIClient

func getCli() *swagger.APIClient {
	if cli == nil {
		cfg := swagger.NewConfiguration()
		cfg.Host = ""
		cfg.Scheme = ""
		cfg.BasePath = "http://127.0.0.1:8000"
		cfg.HTTPClient = http.DefaultClient
		cli = swagger.NewAPIClient(cfg)
	}
	return cli
}

func handleErr(err error, httpResp *http.Response) error {
	if err != nil {
		log.Println("GetCourseSwapRequestList failed: ", err)
		return err
	}
	if httpResp.StatusCode != http.StatusOK {
		log.Println("GetCourseSwapRequestList failed: ", httpResp.Status)
		return fmt.Errorf("http status code: %v", httpResp.StatusCode)
	}
	return nil
}

func inputStrIfEmpty(str *string, msg string) {
	if *str == "" {
		inputStr(str, msg)
	}
}
func inputIntIfZero(i *int, msg string) {
	if *i == 0 {
		inputInt(i, msg)
	}
}

func inputStr(str *string, msg string) {
	log.Print(msg)
	var tmpStr string
	_, _ = fmt.Scanln(&tmpStr)
	if tmpStr != "" {
		*str = tmpStr
	}
}
func inputInt(i *int, msg string) {
	log.Print(msg)
	var tmpInt int
	_, _ = fmt.Scanln(&tmpInt)
	if tmpInt != 0 {
		*i = tmpInt
	}
}

var useConfig bool = true

func CourseSwap() {
	var configPath string = "./configs/courseSwap.ini"
	inputStr(&configPath, "配置文件，默认为./configs/courseSwap.ini，输入NO将不使用配置文件")
	if configPath == "NO" {
		useConfig = false
	}
	var path string
	var srcTeacher string
	var srcDateStr string
	var srcCourseNum int

	if useConfig {
		config.MustInitConfig(configPath)
		path = config.GetStringD("filePath", "")
		srcTeacher = config.GetStringD("teacher", "")
		srcDateStr = config.GetStringD("date", "")
		srcCourseNum = int(config.GetInt64D("courseNum", 0))
	}
	inputStrIfEmpty(&path, "请输入需要导入的课程表文件(excel)，以回车结束")
	inputStrIfEmpty(&srcTeacher, "请输入老师名字，不要输入空格等额外内容，以回车结束")
	inputStrIfEmpty(&srcDateStr, "请输入日期，如20240101，以回车结束")
	inputIntIfZero(&srcCourseNum, "请输入第几节课，1~7，以回车结束")

	if srcTeacher == "" || srcDateStr == "" || srcCourseNum == 0 {
		log.Println("input error")
		return
	}

	srcDate, err := util.TimeFromStr(srcDateStr, "YYYYMMDD")
	if err != nil {
		log.Println("time.Parse failed: ", err)
		return
	}

	log.Println("从excel读取课程表: " + path)
	courseMap, err := NewCourseParser(path).ParseCourseExcel()
	if err != nil {
		log.Println("parseCourseExcel failed: ", err)
		return
	}

	tNow := time.Now()
	wDiff := tNow.Weekday() - time.Monday
	endTime := tNow.AddDate(0, 0, 21-int(wDiff))

	log.Printf("用课程表，计算未来3周内的课程安排[%v]-[%v]\n",
		util.TimeToStr(tNow, "YYYYMMDD"), util.TimeToStr(endTime, "YYYYMMDD"))

	var allCourseList []*courseInfo
	log.Printf("teacher cnt[%v]\n", len(courseMap))
	for _, tInfo := range courseMap {
		// log.Printf("teacher[%v] class cnt[%v]\n", tInfo.teacher, len(tInfo.classList))
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

	log.Println("查询当前换课记录，结合换课记录来计算")
	{ // GetCourseSwapRequestList
		resp, httpResp, err := getCli().SchoolCURDApi.SchoolCURDGetCourseSwapRequestList(
			context.Background(), &swagger.SchoolCURDApiSchoolCURDGetCourseSwapRequestListOpts{
				// IDList: optional.NewInterface([]int64{0}),
			})
		err = handleErr(err, httpResp)
		if err != nil {
			log.Println("GetCourseSwapRequestList failed: ", err)
			return
		}

		sort.Slice(resp.CourseSwapRequestList, func(i, j int) bool {
			return resp.CourseSwapRequestList[i].CreateTime < resp.CourseSwapRequestList[j].CreateTime
		})

		for _, req := range resp.CourseSwapRequestList {
			// allCourseList 中找到src课程
			srcTime, _ := util.TimeFromStr("YYYYMMDD", req.SrcDate)
			srcCourse := getCourse(allCourseList, req.SrcTeacher, srcTime, int(req.SrcCourseNum))
			// allCourseList 中找到dst课程
			dstTime, _ := util.TimeFromStr("YYYYMMDD", req.DstDate)
			dstCourse := getCourse(allCourseList, req.DstTeacher, dstTime, int(req.DstCourseNum))
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

	// 获取当前需要调课的课程，则某老师某天的某节课
	srcCourse := getCourse(allCourseList, srcTeacher, srcDate, srcCourseNum)
	if srcCourse == nil {
		log.Println("srcCourse not found")
		return
	} else {
		logCourse(srcCourse)
	}
	srcClassRoom := srcCourse.classRoomName

	log.Printf("找到[%v][第%v节]，不用上课的，[%v]同班老师\n",
		srcDateStr, srcCourseNum, srcCourse.classRoomName)
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
			c := getCourse(allCourseList, srcTeacher, date, courseNum)
			if c != nil {
				continue
			}
			dstFreeCourseList = append(dstFreeCourseList,
				&courseInfo{date: date, classNum: courseNum})
		}
	}

	log.Printf("找到[%v]未来有空上的目标课程，并且对应老师在[%v][第%v节]有空\n",
		srcTeacher, srcDateStr, srcCourseNum)
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

	if true {
		return
	}
	// AddCourseSwapRequest
	resp, httpResp, err := getCli().SchoolCURDApi.SchoolCURDAddCourseSwapRequest(
		context.Background(), swagger.ApiAddCourseSwapRequestRequest{
			CourseSwapRequest: &swagger.ApiCourseSwapRequestInfo{
				SrcTeacher: srcTeacher,
			}})
	err = handleErr(err, httpResp)
	if err != nil {
		log.Println("AddCourseSwap failed: ", err)
		return
	}
	log.Printf("new course swap : %v", resp)
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
	log.Printf("course[%v][%v][第%v节][%v][%v][%v]",
		course.date.Format("060102"),
		getWeekday(course.date.Weekday()),
		course.classNum, course.classRoomName,
		course.teacher, course.className)
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
