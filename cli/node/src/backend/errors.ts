export class AuthError extends Error {
  /**
   * 构造外部接口错误并保留失败上下文。
   * @param message - 具体失败原因。
   */
  constructor(message: string) {
    super(message);
    this.name = 'AuthError';
  }
}

export class NotFoundError extends Error {
  /**
   * 构造外部接口错误并保留失败上下文。
   * @param path - 失败的资源或接口路径。
   */
  constructor(path: string) {
    super(`Resource not found: ${path}`);
    this.name = 'NotFoundError';
  }
}

export class APIError extends Error {
  /**
   * 构造外部接口错误并保留失败上下文。
   * @param path - 失败的资源或接口路径。
   * @param detail - 接口返回的具体错误信息。
   */
  constructor(path: string, detail: string) {
    super(`Bad request to ${path}: ${detail}`);
    this.name = 'APIError';
  }
}
