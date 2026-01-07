package courseSwap

import (
	"fmt"

	"github.com/pancake-lee/pgo/pkg/putil"
)

func CourseSwapCli() {
	config, err := InputParams()
	if err != nil {
		putil.Interact.Errorf(err.Error())
		return
	}
	mgr, err := CalculateSwapCandidates(config)
	if err != nil {
		putil.Interact.Errorf(err.Error())
		return
	}

	courses := mgr.GetCourses()
	if len(courses) == 0 {
		putil.Interact.Errorf("没有找到可换的课程")
		return
	}

	for i, c := range courses {
		putil.Interact.Infof("[%d] %v", i, c)
	}

	idxStr := putil.Interact.MustInput(
		fmt.Sprintf("请输入需要换课的目标序号(0-%d): ", len(courses)-1))

	index, err := putil.StrToInt(idxStr)
	if err != nil {
		putil.Interact.Errorf("检查输入是否范围内的序号，错误：%v", err)
		return
	}
	if index < 0 || index >= len(courses) {
		putil.Interact.Errorf("序号[%v]超出范围(0-%d)", index, len(courses)-1)
		return
	}

	selected := courses[index]
	putil.Interact.Infof("选择的序号为: %v", selected)

	putil.Interact.MustConfirm("确认换课?")

	err = ExecuteSwap(config, selected)
	if err != nil {
		putil.Interact.Errorf(err.Error())
	} else {
		putil.Interact.Infof("换课成功!")
	}
}
