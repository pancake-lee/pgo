package putil

import (
	"crypto/rand"
	"fmt"
	mrand "math/rand"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
)

// --------------------------------------------------
var mr *mrand.Rand

// [start, end)
func GetRand(start, end int) int {
	if mr == nil {
		mr = mrand.New(mrand.NewSource((time.Now().UnixNano())))
	}
	return mr.Intn(end-start) + start
}

func GetRandStr(n int) string {
	b := make([]byte, n/2+1)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	s := ""
	for _, v := range b {
		s += fmt.Sprintf("%02x", v)
	}
	s = s[:n]
	return s
}

// --------------------------------------------------
// 管理ID的分配和释放，当前不会在用的ID不会重复，释放了的ID可以重新分配

var DefaultIDMgr = NewIDManager(100000)

type IDManager struct {
	maxId    int32
	mu       sync.Mutex
	occupied map[int32]bool
}

func NewIDManager(initMaxId int32) *IDManager {
	return &IDManager{
		maxId:    initMaxId,
		occupied: make(map[int32]bool),
	}
}

// 获取最小但不重复的ID
func (m *IDManager) GetNewSmallestAndUniqueID() int32 {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i := int32(0); i < m.maxId; i++ {
		if !m.occupied[i] {
			m.occupied[i] = true
			return i
		}
	}
	return -1
}

func (m *IDManager) ReleaseID(id int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.occupied[id] {
		m.occupied[id] = false
	}
}

// --------------------------------------------------
func UUID() string {
	u, _ := uuid.NewV4()
	return u.String()
}

// 截断的短版UUID
func UUID_S() string {
	u, _ := uuid.NewV4()
	return u.String()[0:8]
}
