/** 后端端口与工厂的公开入口。 */

export { APIError, AuthError, NotFoundError } from './errors.js';
export { getBackend } from './factory.js';
export { OSSBackend } from './oss.js';
export type {
  AddOptions,
  Backend,
  DeleteOptions,
  EntityIds,
  ListOptions,
  SearchOptions,
} from './types.js';
