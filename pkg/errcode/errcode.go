package errcode

import (
	"fmt"
	"net/http"
)

// Error 表示包含 HTTP 状态码与业务错误码的企业级通用错误。
type Error struct {
	HTTPStatus int    `json:"-"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// 常见系统级与业务错误码定义
var (
	// 成功
	Success = &Error{HTTPStatus: http.StatusOK, Code: 0, Message: "success"}

	// 客户端通用错误 (400xx)
	ErrInvalidParam   = &Error{HTTPStatus: http.StatusBadRequest, Code: 40001, Message: "请求参数错误"}
	ErrPayloadTooLarge = &Error{HTTPStatus: http.StatusRequestEntityTooLarge, Code: 40002, Message: "请求载荷过大"}
	ErrBadRequest     = &Error{HTTPStatus: http.StatusBadRequest, Code: 40003, Message: "无效的请求"}

	// 认证与权限 (401xx, 403xx)
	ErrUnauthorized = &Error{HTTPStatus: http.StatusUnauthorized, Code: 40101, Message: "认证未通过或口令无效"}
	ErrForbidden    = &Error{HTTPStatus: http.StatusForbidden, Code: 40301, Message: "权限拒绝"}

	// 资源状态 (404xx, 409xx)
	ErrNotFound = &Error{HTTPStatus: http.StatusNotFound, Code: 40401, Message: "目标资源不存在"}
	ErrConflict = &Error{HTTPStatus: http.StatusConflict, Code: 40901, Message: "资源唯一键冲突或已存在"}

	// 微信协议特有错误 (420xx)
	ErrWeChatBadSignature = &Error{HTTPStatus: http.StatusUnauthorized, Code: 42001, Message: "微信消息签名校验失败"}
	ErrWeChatDecryptFail  = &Error{HTTPStatus: http.StatusBadRequest, Code: 42002, Message: "微信密文解密失败"}
	ErrWeChatMissingKey   = &Error{HTTPStatus: http.StatusBadRequest, Code: 42003, Message: "未配置微信 EncodingAESKey"}
	ErrWeChatBadXML       = &Error{HTTPStatus: http.StatusBadRequest, Code: 42004, Message: "微信消息 XML 格式非法"}

	// 服务端内部错误 (500xx)
	ErrInternalServer = &Error{HTTPStatus: http.StatusInternalServerError, Code: 50001, Message: "系统内部错误"}
	ErrDatabase       = &Error{HTTPStatus: http.StatusInternalServerError, Code: 50002, Message: "数据库存储异常"}
	ErrServiceUnavailable = &Error{HTTPStatus: http.StatusServiceUnavailable, Code: 50301, Message: "服务暂时不可用"}
)

// WithMessage 派生具有特定自定义提示信息的同类型错误。
func (e *Error) WithMessage(msg string) *Error {
	return &Error{
		HTTPStatus: e.HTTPStatus,
		Code:       e.Code,
		Message:    msg,
	}
}
