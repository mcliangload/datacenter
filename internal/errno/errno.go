package errno

import "net/http"

// Error 业务错误：包含业务码、HTTP 状态码与用户可读信息。
// Cause 仅用于日志记录，不返回给客户端。
type Error struct {
	Code       int
	HTTPStatus int
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// New 构造业务错误
func New(code, httpStatus int, message string) *Error {
	return &Error{Code: code, HTTPStatus: httpStatus, Message: message}
}

// WithCause 附加底层错误（供日志排查，不泄露给客户端）
func (e *Error) WithCause(err error) *Error {
	cp := *e
	cp.Cause = err
	return &cp
}

// 预定义错误码
// 业务码段：0 成功；1xxx 通用；2xxx 用户/权限；3xxx 集合；4xxx 数据项；5xxx 刮削。
var (
	OK              = &Error{Code: 0, HTTPStatus: http.StatusOK, Message: "ok"}
	ErrParam        = &Error{Code: 1001, HTTPStatus: http.StatusBadRequest, Message: "参数错误"}
	ErrUnauthorized = &Error{Code: 1002, HTTPStatus: http.StatusUnauthorized, Message: "未认证或登录已过期"}
	ErrForbidden    = &Error{Code: 1003, HTTPStatus: http.StatusForbidden, Message: "无权限执行该操作"}
	ErrNotFound     = &Error{Code: 1004, HTTPStatus: http.StatusNotFound, Message: "资源不存在"}
	ErrConflict     = &Error{Code: 1005, HTTPStatus: http.StatusConflict, Message: "资源冲突"}
	ErrInternal     = &Error{Code: 1006, HTTPStatus: http.StatusInternalServerError, Message: "服务内部错误"}
	ErrBodyTooLarge = &Error{Code: 1007, HTTPStatus: http.StatusRequestEntityTooLarge, Message: "请求体过大（上限 4MB）"}

	// 2xxx 用户/权限
	ErrUserNotFound   = &Error{Code: 2001, HTTPStatus: http.StatusNotFound, Message: "用户不存在"}
	ErrUsernameExists = &Error{Code: 2002, HTTPStatus: http.StatusConflict, Message: "用户名已存在"}
	ErrUserSelfOp     = &Error{Code: 2003, HTTPStatus: http.StatusBadRequest, Message: "不能对自己执行该操作"}
	ErrLastAdmin      = &Error{Code: 2004, HTTPStatus: http.StatusBadRequest, Message: "不能删除或禁用最后一个管理员"}

	// 3xxx 集合
	ErrCollectionNotFound   = &Error{Code: 3001, HTTPStatus: http.StatusNotFound, Message: "集合不存在"}
	ErrCollectionNameExists = &Error{Code: 3002, HTTPStatus: http.StatusConflict, Message: "集合名称已存在"}
	ErrNotMember            = &Error{Code: 3003, HTTPStatus: http.StatusForbidden, Message: "您不是该集合成员"}
	ErrNoPermission         = &Error{Code: 3004, HTTPStatus: http.StatusForbidden, Message: "当前角色无权限执行该操作"}
	ErrTagSchemaInvalid     = &Error{Code: 3005, HTTPStatus: http.StatusBadRequest, Message: "标签定义不合法"}
	ErrNoScrapeScript       = &Error{Code: 3006, HTTPStatus: http.StatusBadRequest, Message: "集合尚未配置刮削脚本"}
	ErrCannotRemoveAdmin    = &Error{Code: 3007, HTTPStatus: http.StatusBadRequest, Message: "不能通过移除成员接口移除集合管理员，请使用更换集合管理员接口"}
	ErrMemberExists         = &Error{Code: 3008, HTTPStatus: http.StatusConflict, Message: "该用户已是集合成员"}
	ErrNotMemberOfCol       = &Error{Code: 3009, HTTPStatus: http.StatusBadRequest, Message: "该用户不是集合成员"}

	// 4xxx 数据项
	ErrItemNotFound    = &Error{Code: 4001, HTTPStatus: http.StatusNotFound, Message: "数据项不存在"}
	ErrPathNotExist    = &Error{Code: 4002, HTTPStatus: http.StatusBadRequest, Message: "路径不存在或不可访问"}
	ErrPathOutsideRoot = &Error{Code: 4003, HTTPStatus: http.StatusBadRequest, Message: "路径不在数据根目录内"}
	ErrTagValueInvalid = &Error{Code: 4004, HTTPStatus: http.StatusBadRequest, Message: "标签值不合法"}
	ErrItemPathExists  = &Error{Code: 4005, HTTPStatus: http.StatusConflict, Message: "该路径已在集合中注册"}
	ErrDQLInvalid      = &Error{Code: 4006, HTTPStatus: http.StatusBadRequest, Message: "DQL 语句不合法"}

	// 5xxx 刮削
	ErrTaskNotFound = &Error{Code: 5001, HTTPStatus: http.StatusNotFound, Message: "刮削任务不存在"}
)
