package courseSwap

import (
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pthird"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func CourseSwapCli() {
	config, err := InputParams()
	if err != nil {
		pthird.Interact.Error(err)
		return
	}
	mgr, err := CalculateSwapCandidates(config)
	if err != nil {
		pthird.Interact.Error(err)
		return
	}

	courses := mgr.GetCourses()
	if len(courses) == 0 {
		pthird.Interact.Errorf("没有找到可换的课程")
		return
	}

	for i, c := range courses {
		pthird.Interact.Infof("[%d] %v", i, c)
	}

	idxStr := pthird.Interact.MustInput(
		fmt.Sprintf("请输入需要换课的目标序号(0-%d): ", len(courses)-1))

	index, err := putil.StrToInt(idxStr)
	if err != nil {
		pthird.Interact.Errorf("检查输入是否范围内的序号，错误：%v", err)
		return
	}
	if index < 0 || index >= len(courses) {
		pthird.Interact.Errorf("序号[%v]超出范围(0-%d)", index, len(courses)-1)
		return
	}

	selected := courses[index]
	pthird.Interact.Infof("选择的序号为: %v", selected)

	pthird.Interact.MustConfirm("确认换课?")

	err = ExecuteSwap(config, selected)
	if err != nil {
		pthird.Interact.Error(err)
	} else {
		pthird.Interact.Infof("换课成功!")
	}
}
