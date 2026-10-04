package core

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	klog "github.com/go-kratos/kratos/v2/log"
)

// fakeDataLifecycle 记录生命周期调用，验证复用与重建的执行顺序。
type fakeDataLifecycle struct {
	state       DataState
	checkErr    error
	cleanupErr  error
	callList    []string
	cleanupPath string
}

// CheckData 返回可控状态而不访问真实业务。
func (data *fakeDataLifecycle) CheckData(context.Context, string,
) (DataState, error) {
	data.callList = append(data.callList, "check")
	return data.state, data.checkErr
}

// CreateData 记录新批次创建请求。
func (data *fakeDataLifecycle) CreateData(context.Context, string) error {
	data.callList = append(data.callList, "create")
	return nil
}

// CleanupData 记录精确清理请求和清单目录。
func (data *fakeDataLifecycle) CleanupData(_ context.Context, path string,
) error {
	data.callList = append(data.callList, "cleanup")
	data.cleanupPath = path
	return data.cleanupErr
}

// Prepare 满足场景入口契约，清理入口不应调用负载准备。
func (data *fakeDataLifecycle) Prepare(context.Context, string,
) (string, func() error, error) {
	panic("cleanup must not prepare a load")
}

// TestEnsureDataLifecycle 验证先检查、按需清理再创建且错误时保留数据。
func TestEnsureDataLifecycle(t *testing.T) {
	failure := errors.New("HTTP unavailable")
	testList := []struct {
		name       string
		state      DataState
		checkErr   error
		cleanupErr error
		wantList   []string
		wantErr    bool
	}{
		{"missing", DataMissing, nil, nil, []string{"check", "create"}, false},
		{"reuse", DataReusable, nil, nil, []string{"check"}, false},
		{"rebuild", DataNeedsRebuild, nil, nil, []string{"check", "cleanup", "create"}, false},
		{"check-failed", DataReusable, failure, nil, []string{"check"}, true},
		{"cleanup-failed", DataNeedsRebuild, nil, failure, []string{"check", "cleanup"}, true},
	}
	for _, item := range testList {
		t.Run(item.name, func(t *testing.T) {
			data := &fakeDataLifecycle{
				state: item.state, checkErr: item.checkErr,
				cleanupErr: item.cleanupErr,
			}
			err := EnsureData(t.Context(), t.TempDir(), data,
				klog.NewStdLogger(io.Discard))
			if (err != nil) != item.wantErr || !reflect.DeepEqual(data.callList, item.wantList) {
				t.Fatalf("calls=%v error=%v", data.callList, err)
			}
		})
	}
}

// TestCleanupCommandNeedsOnlyDirectory 验证独立清理不发现服务或检查负载工具。
func TestCleanupCommandNeedsOnlyDirectory(t *testing.T) {
	data := &fakeDataLifecycle{}
	entrypoint := NewEntrypoint(Scenario{
		Name: "sample", Short: "sample",
		NewPreparer: func(Config, klog.Logger) Preparer { return data },
	})
	path := t.TempDir()
	command := entrypoint.CleanupEntrypoint().NewCobraCommand()
	command.SetOut(io.Discard)
	command.SetArgs([]string{path})
	err := command.ExecuteContext(t.Context())
	if err != nil || data.cleanupPath != path || len(data.callList) != 1 {
		t.Fatalf("cleanup calls=%v path=%s error=%v", data.callList, data.cleanupPath, err)
	}
	command = entrypoint.NewCobraCommand()
	if command.Flag("output-dir") == nil {
		t.Fatal("test cannot select the shared data directory")
	}
	command.SetOut(io.Discard)
	command.SetArgs([]string{"cleanup", path})
	err = command.ExecuteContext(t.Context())
	if err != nil || len(data.callList) != 2 {
		t.Fatalf("nested cleanup error=%v calls=%v", err, data.callList)
	}
}
