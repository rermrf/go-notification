package client

import (
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v5/client"
	"github.com/alibabacloud-go/tea/tea"
	"strings"
)

var (
	platformTemplateType2Aliyun = map[TemplateType]TemplateType{
		TemplateTypeVerification:  TemplateTypeVerification,
		TemplateTypeNotification:  TemplateTypeMarketing,
		TemplateTypeMarketing:     TemplateTypeNotification,
		TemplateTypeInternational: TemplateTypeInternational,
	}
	_ Client = (*AliyunSMS)(nil)
)

// AliyunSMS 阿里云短信实现
type AliyunSMS struct {
	client *dysmsapi.Client
}

func NewAliyunSMS(regionId, accessKeyID, accessKeySecret string) (*AliyunSMS, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		RegionId:        tea.String(regionId),
		Endpoint:        tea.String("dysmsapi.aliyuncs.com"),
	}
	client, err := dysmsapi.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &AliyunSMS{
		client: client,
	}, nil
}

func (a *AliyunSMS) CreateTemplate(req CreateTemplateReq) (CreateTemplateResp, error) {
	// https://help.aliyun.com/zh/sms/developer-reference/api-dysmsapi-2017-05-25-createsmstemplate?spm=a2c4g.11186623.help-menu-44282.d_4_2_4_2_0.18706b6bAOg39L&scm=20140722.H_2807431._.OR_help-T_cn~zh-V_1
	templateType, ok := platformTemplateType2Aliyun[req.TemplateType]
	if !ok {
		return CreateTemplateResp{}, fmt.Errorf("%w: 模版类型非法", ErrInvalidParameter)
	}

	request := &dysmsapi.CreateSmsTemplateRequest{
		TemplateName:    tea.String(req.TemplateName),
		TemplateContent: tea.String(req.TemplateContent),
		TemplateType:    tea.Int32(int32(templateType)),
		Remark:          tea.String(req.Remark),
	}

	response, err := a.client.CreateSmsTemplate(request)
	if err != nil {
		return CreateTemplateResp{}, fmt.Errorf("%w: %w", ErrCreateTemplateFailed, err)
	}

	if response.Body == nil || response.Body.Code == nil || !strings.EqualFold(*response.Body.Code, "OK") {
		return CreateTemplateResp{}, fmt.Errorf("%w: %v", ErrCreateTemplateFailed, "响应异常")
	}

	return CreateTemplateResp{
		RequestID:  *response.Body.RequestId,
		TemplateID: *response.Body.TemplateCode,
	}, nil
}

func (a *AliyunSMS) BatchQueryTemplateStatus(req BatchQueryTemplateStatusReq) (BatchQueryTemplateStatusResp, error) {
	// https://help.aliyun.com/zh/sms/developer-reference/api-dysmsapi-2017-05-25-querysmstemplatelist?spm=a2c4g.11186623.help-menu-44282.d_4_2_4_2_2.13686e8bNLlSVA&scm=20140722.H_419288._.OR_help-T_cn~zh-V_1

	// 如果没有模版ID，返回空结果
	if len(req.TemplateIDs) == 0 {
		return BatchQueryTemplateStatusResp{
			Results: make(map[string]QueryTemplateStatusResp),
		}, nil
	}

	// 构建结果map
	results := make(map[string]QueryTemplateStatusResp)

	// 创建模版ID的map，提高查找效率
	requestedIDMap := make(map[string]bool)
	for _, templateID := range req.TemplateIDs {
		requestedIDMap[templateID] = true
	}

	// 阿里云不支持直接通过模版ID列表查询，需要遍历PageIndex来获取所有模版
	// 为了效率，先分页获取所有模版，然后筛选出我们需要的
	pageSize := 50
	pageIndex := 1

	for {
		request := &dysmsapi.QuerySmsTemplateListRequest{
			PageSize:  tea.Int32(int32(pageSize)),
			PageIndex: tea.Int32(int32(pageIndex)),
		}

		response, err := a.client.QuerySmsTemplateList(request)
		if err != nil {
			return BatchQueryTemplateStatusResp{}, fmt.Errorf()
		}
	}
}

func (a *AliyunSMS) Send(req SendReq) (SendResp, error) {

}
