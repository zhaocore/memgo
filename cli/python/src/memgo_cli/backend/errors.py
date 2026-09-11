"""可分类的后端失败类型。"""


class AuthError(Exception):
    """服务拒绝鉴权凭据。"""


class NotFoundError(Exception):
    """目标资源不存在。"""


class APIError(Exception):
    """请求、响应或协议不符合约定。"""
