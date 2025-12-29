package courseSwap

import (
	"time"

	"github.com/pancake-lee/pgo/pkg/putil"
)

type courseManager struct {
	courses []*courseInfo
}

func newCourseManager(courses []*courseInfo) *courseManager {
	return &courseManager{courses: courses}
}

func (m *courseManager) getTeacherListByClassRoom(classRoom string) []string {
	var retList []string
	for _, c := range m.courses {
		if c.classRoomName == classRoom {
			retList = append(retList, c.teacher)
		}
	}
	retList = putil.StrListUnique(retList)
	return retList
}

func (m *courseManager) getAllTeacherList() []string {
	var retList []string
	for _, c := range m.courses {
		retList = append(retList, c.teacher)
	}
	retList = putil.StrListUnique(retList)
	return retList
}

func (m *courseManager) getCourseByDateAndNum(t time.Time, courseNum int) *courseInfo {
	for _, c := range m.courses {
		if c.date.Format("20160102") == t.Format("20160102") &&
			c.classNum == courseNum {
			return c
		}
	}
	return nil
}

func (m *courseManager) getCourseByTeacher(teacher string) []*courseInfo {
	var retList []*courseInfo
	for _, c := range m.courses {
		if c.teacher == teacher {
			retList = append(retList, c)
		}
	}
	return retList
}

func (m *courseManager) getCourse(teacher string, t time.Time, courseNum int) *courseInfo {
	for _, c := range m.courses {
		if c.date.Format("20160102") == t.Format("20160102") &&
			c.classNum == courseNum &&
			c.teacher == teacher {
			return c
		}
	}
	return nil
}

func (m *courseManager) logCourseList() {
	for _, c := range m.courses {
		logCourse(c)
	}
}
