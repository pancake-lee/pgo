package courseSwap

import (
	"sort"
	"time"

	"github.com/pancake-lee/pgo/pkg/putil"
)

type courseManager struct {
	courses []*CourseInfo
}

func newCourseManager(courses []*CourseInfo) *courseManager {
	return &courseManager{courses: courses}
}

func (m *courseManager) GetCourses() []*CourseInfo {
	return m.courses
}

func (m *courseManager) getTeacherListByClassRoom(classRoom string) []string {
	var retList []string
	for _, c := range m.courses {
		if c.ClassRoomName == classRoom {
			retList = append(retList, c.Teacher)
		}
	}
	retList = putil.StrListUnique(retList)
	return retList
}

func (m *courseManager) GetAllTeacherList() []string {
	var retList []string
	for _, c := range m.courses {
		retList = append(retList, c.Teacher)
	}
	retList = putil.StrListUnique(retList)
	return retList
}

func (m *courseManager) getCourseByDateAndNum(t time.Time, courseNum int) *CourseInfo {
	for _, c := range m.courses {
		if c.Date.Format("20160102") == t.Format("20160102") &&
			c.ClassNum == courseNum {
			return c
		}
	}
	return nil
}

func (m *courseManager) getCourseByTeacher(teacher string) []*CourseInfo {
	var retList []*CourseInfo
	for _, c := range m.courses {
		if c.Teacher == teacher {
			retList = append(retList, c)
		}
	}
	return retList
}

func (m *courseManager) getCourse(teacher string, t time.Time, courseNum int) *CourseInfo {
	for _, c := range m.courses {
		if c.Date.Format("20160102") == t.Format("20160102") &&
			c.ClassNum == courseNum &&
			c.Teacher == teacher {
			return c
		}
	}
	return nil
}

func (m *courseManager) logCourseList() {
	sort.Slice(m.courses, func(i, j int) bool {
		return m.courses[i].ClassName < m.courses[j].ClassName
	})
	for _, c := range m.courses {
		logCourse(c)
	}
}
