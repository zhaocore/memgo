import { createServer } from 'node:http';
import type { AddressInfo } from 'node:net';
import { expect, it } from 'vitest';
import { requestOnce } from '../../src/backend/http.js';

it('真实未响应服务在超时后取消连接，并保留请求上下文', async () => {
  const server = createServer(() => {
    /* 保持连接不返回，验证客户端取消。 */
  });
  await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
  try {
    const address = server.address() as AddressInfo;
    await expect(
      requestOnce({
        url: `http://127.0.0.1:${address.port}/waiting`,
        path: '/waiting',
        method: 'GET',
        headers: {},
        body: undefined,
        timeoutMs: 30,
      })
    ).rejects.toThrow('GET failed after at most 30ms');
  } finally {
    server.closeAllConnections();
    await new Promise<void>((resolve, reject) =>
      server.close((error) => (error ? reject(error) : resolve()))
    );
  }
});
