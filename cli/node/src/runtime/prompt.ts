import readline from 'node:readline';

/**
 * 在提示前关闭终端回显，读取密钥并恢复终端状态。
 * @param label - 终端提示文本。
 * @returns 用户输入的密钥字符串。
 */
export function promptSecret(label: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const wasRaw = Boolean(process.stdin.isRaw);

    if (process.stdin.isTTY) {
      process.stdin.setRawMode(true);
    }
    process.stdin.resume();
    process.stdin.setEncoding('utf-8');

    const chars: string[] = [];

    const onData = (key: string) => {
      for (const ch of key) {
        if (ch === '\r' || ch === '\n') {
          cleanup();
          process.stdout.write('\n');
          resolve(chars.join(''));
          return;
        }
        if (ch === '\x03') {
          cleanup();
          reject(new Error('Interrupted'));
          return;
        }
        if (ch === '\x7f' || ch === '\x08') {
          // 删除前一个字符。
          if (chars.length > 0) {
            chars.pop();
            process.stdout.write('\b \b');
          }
        } else if (ch === '\x15') {
          // Ctrl+U 清空当前输入行。
          process.stdout.write('\b \b'.repeat(chars.length));
          chars.length = 0;
        } else if (ch >= ' ') {
          chars.push(ch);
          process.stdout.write('*');
        }
      }
    };

    const cleanup = () => {
      process.stdin.removeListener('data', onData);
      process.stdin.removeListener('end', onEnd);
      process.stdin.removeListener('error', onError);
      if (process.stdin.isTTY) {
        process.stdin.setRawMode(wasRaw);
      }
      process.stdin.pause();
    };

    const onEnd = () => {
      cleanup();
      reject(new Error('Secret input ended before confirmation'));
    };
    const onError = (error: Error) => {
      cleanup();
      reject(error);
    };
    process.stdin.on('data', onData);
    process.stdin.once('end', onEnd);
    process.stdin.once('error', onError);
    // 先关闭终端回显并注册监听，再显示提示，避免快速输入泄露密钥。
    process.stdout.write(label);
  });
}

/**
 * 读取一行终端输入并清除首尾空白。
 * @param label - 终端提示文本。
 * @param defaultValue - 空输入时使用的值。
 * @returns 清除首尾空白后的输入；支持默认值的入口在空输入时使用默认值。
 */
export function promptLine(label: string, defaultValue?: string): Promise<string> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  const prompt = defaultValue ? `${label} [${defaultValue}]: ` : `${label}: `;
  return new Promise((resolve) => {
    rl.question(prompt, (answer) => {
      rl.close();
      resolve(answer.trim() || defaultValue || '');
    });
  });
}
