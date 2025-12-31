package courseSwap

import (
	"context"
	"fmt"
	"net/http"

	"github.com/antihax/optional"
	"github.com/pancake-lee/pgo/client/swagger"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

type CourseSwapRepo interface {
	GetCourseSwapRequestList(ctx context.Context) ([]swagger.ApiCourseSwapRequestInfo, error)
	AddCourseSwapRequest(ctx context.Context, req *swagger.ApiCourseSwapRequestInfo) error
	DeleteCourseSwapRequest(ctx context.Context, id int32) error
}

type CloudRepo struct {
	cli *swagger.APIClient
}

func NewCloudRepo() *CloudRepo {
	cfg := swagger.NewConfiguration()
	cfg.Host = ""
	cfg.Scheme = ""
	cfg.BasePath = "http://127.0.0.1:8000"
	cfg.HTTPClient = http.DefaultClient
	return &CloudRepo{
		cli: swagger.NewAPIClient(cfg),
	}
}

func (r *CloudRepo) GetCourseSwapRequestList(ctx context.Context) ([]swagger.ApiCourseSwapRequestInfo, error) {
	resp, httpResp, err := r.cli.SchoolCURDApi.SchoolCURDGetCourseSwapRequestList(
		ctx, &swagger.SchoolCURDApiSchoolCURDGetCourseSwapRequestListOpts{
			// IDList: optional.NewInterface([]int64{0}),
		})
	err = handleErr(err, httpResp)
	if err != nil {
		return nil, err
	}
	return resp.CourseSwapRequestList, nil
}

func (r *CloudRepo) AddCourseSwapRequest(ctx context.Context, req *swagger.ApiCourseSwapRequestInfo) error {
	_, httpResp, err := r.cli.SchoolCURDApi.SchoolCURDAddCourseSwapRequest(
		ctx, swagger.ApiAddCourseSwapRequestRequest{
			CourseSwapRequest: req,
		})
	return handleErr(err, httpResp)
}

func (r *CloudRepo) DeleteCourseSwapRequest(ctx context.Context, id int32) error {
	_, httpResp, err := r.cli.SchoolCURDApi.SchoolCURDDelCourseSwapRequestByIDList(
		ctx, &swagger.SchoolCURDApiSchoolCURDDelCourseSwapRequestByIDListOpts{
			IDList: optional.NewInterface([]int32{id}),
		})
	return handleErr(err, httpResp)
}

func handleErr(err error, httpResp *http.Response) error {
	if err != nil {
		plogger.Debug("Request failed: ", err)
		return err
	}
	if httpResp.StatusCode != http.StatusOK {
		plogger.Debug("Request failed: ", httpResp.Status)
		return fmt.Errorf("http status code: %v", httpResp.StatusCode)
	}
	return nil
}
