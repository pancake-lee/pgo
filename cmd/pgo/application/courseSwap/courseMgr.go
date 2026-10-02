package courseSwap

import (
	"sort"
	"time"

	"github.com/pancake-lee/pgo/pkg/plogger"
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

func (m *courseManager) getCourse(teacher string, t time.Time, courseNum int) *CourseInfo {
	for _, c := range m.courses {
		if c.ClassNum == courseNum &&
			c.Teacher == teacher {
			if putil.DateToStrDefault(c.Date) == putil.DateToStrDefault(t) {
				return c
			}
			// plogger.Debugf("date not match, c.Date[%v] t[%v]",
			// 	putil.DateToStrDefault(c.Date),
			// 	putil.DateToStrDefault(t))
		}
	}
	// plogger.Errorf("teacher[%v] date[%v] courseNum[%v] not found",
	// teacher, putil.DateToStrDefault(t), courseNum)
	return nil
}

func (m *courseManager) logCourseList(teacher string, date time.Time) {
	sort.Slice(m.courses, func(i, j int) bool {
		return m.courses[i].ClassName < m.courses[j].ClassName
	})
	for _, c := range m.courses {
		if c.Teacher != teacher {
			continue
		}
		if putil.DateToStrDefault(c.Date) != putil.DateToStrDefault(date) {
			continue
		}
		plogger.Debug(c)
	}
}
