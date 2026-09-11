import { type Server, createServer } from 'node:http';
import type { AddressInfo } from 'node:net';
import { type JsonObject, jsonObject } from '../../src/backend/json.js';

export interface RecordedRequest {
  method: string;
  path: string;
  params: Record<string, string>;
  headers: import('node:http').IncomingHttpHeaders;
  json: JsonObject | undefined;
}
export interface TestResponse {
  status: number;
  body: string;
  headers?: Record<string, string>;
}
export interface HttpFixture {
  url: string;
  requests: RecordedRequest[];
  close(): Promise<void>;
}
/**
 * 使用真实本地 HTTP 服务验证协议，禁止依赖外部账号。
 * @param handler - 根据记录的请求生成响应的处理函数。
 * @returns 包含请求记录、服务地址与关闭方法的测试服务。
 */
export async function serve(
  handler: (request: RecordedRequest) => TestResponse
): Promise<HttpFixture> {
  const requests: RecordedRequest[] = [];
  const server: Server = createServer(async (req, res) => {
    try {
      const chunks: Buffer[] = [];
      for await (const chunk of req) chunks.push(Buffer.from(chunk));
      const text = Buffer.concat(chunks).toString();
      const url = new URL(req.url ?? '/', 'http://localhost');
      const request: RecordedRequest = {
        method: req.method ?? 'GET',
        path: url.pathname,
        params: Object.fromEntries(url.searchParams),
        headers: req.headers,
        json: text ? jsonObject(JSON.parse(text)) : undefined,
      };
      requests.push(request);
      const response = handler(request);
      res.writeHead(response.status, {
        'Content-Type': 'application/json',
        ...response.headers,
      });
      res.end(response.body);
    } catch (error) {
      res.writeHead(500);
      res.end(
        JSON.stringify({
          error: error instanceof Error ? error.message : String(error),
        })
      );
    }
  });
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  return {
    url: `http://127.0.0.1:${(server.address() as AddressInfo).port}`,
    requests,
    close: () =>
      new Promise<void>((resolve, reject) => {
        server.closeAllConnections();
        server.close((error) => (error ? reject(error) : resolve()));
      }),
  };
}
