/** Commander 帮助采用与 Python CLI 一致的圆角面板、颜色和选项分组。 */

import chalk from 'chalk';
import type { Argument, Command, Help, Option } from 'commander';
// 直接使用 Chalk，保持终端配色。

// 终端颜色定义。

const cyanBold = chalk.cyan.bold; // 参数占位符。
const yellow = chalk.yellow; // 用法标签。
const bold = chalk.bold; // 用法中的命令名。
const dim = chalk.dim; // 默认值和描述。
const dimBorder = chalk.dim; // 面板边框。

// 移除 ANSI 控制序列。

// biome-ignore lint/suspicious/noControlCharactersInRegex: 有意匹配 ANSI 转义序列。
const ANSI_RE = /\x1b\[[0-9;]*m/g; // eslint-disable-line no-control-regex -- 帮助排版需要匹配 ANSI 转义序列。

/**
 * 移除 ANSI 控制序列后计算文本长度。
 * @param str - 可能包含 ANSI 序列的文本。
 * @returns 去除 ANSI 序列后的字符串长度。
 */
function stripAnsi(str: string): number {
  return str.replace(ANSI_RE, '').length;
}

// 命令显示顺序与 Python CLI 一致。

/** 按现有 rich_help_panel 规则分组。 */
const COMMAND_GROUPS: { panel: string; commands: string[] }[] = [
  {
    panel: 'Memory',
    commands: ['add', 'search', 'get', 'list', 'update', 'delete'],
  },
  {
    panel: 'Management',
    commands: ['init', 'status', 'version', 'import', 'help', 'entity', 'event', 'config'],
  },
];

// 选项与面板的对应关系。

const OPTION_PANELS: Record<string, Record<string, string>> = {
  add: {
    '--user-id': 'Scope',
    '--agent-id': 'Scope',
    '--app-id': 'Scope',
    '--run-id': 'Scope',
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  search: {
    '--user-id': 'Scope',
    '--agent-id': 'Scope',
    '--app-id': 'Scope',
    '--run-id': 'Scope',
    '--top-k': 'Search',
    '--threshold': 'Search',
    '--rerank': 'Search',
    '--keyword': 'Search',
    '--filter': 'Search',
    '--fields': 'Search',
    '--graph': 'Search',
    '--no-graph': 'Search',
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  get: {
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  list: {
    '--user-id': 'Scope',
    '--agent-id': 'Scope',
    '--app-id': 'Scope',
    '--run-id': 'Scope',
    '--page': 'Pagination',
    '--page-size': 'Pagination',
    '--category': 'Filters',
    '--after': 'Filters',
    '--before': 'Filters',
    '--graph': 'Filters',
    '--no-graph': 'Filters',
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  update: {
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  delete: {
    '--user-id': 'Scope',
    '--agent-id': 'Scope',
    '--app-id': 'Scope',
    '--run-id': 'Scope',
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  status: {
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
  import: {
    '--user-id': 'Scope',
    '--agent-id': 'Scope',
    '--output': 'Output',
    '--api-key': 'Connection',
    '--base-url': 'Connection',
  },
};

const PANEL_ORDER: string[] = ['Scope', 'Search', 'Pagination', 'Filters', 'Output', 'Connection'];

// 面板渲染。

/**
 * 渲染圆角面板，按内容补齐边框宽度。
 * @param title - 展示标题。
 * @param rows - 面板内容行。
 * @param width - 包含边框的面板宽度。
 * @returns 包含边框的面板文本；没有内容时返回空字符串。
 */
function renderPanel(title: string, rows: string[], width: number): string {
  if (rows.length === 0) return '';

  // 内部宽度扣除两侧边框。
  const inner = width - 2;

  // 上边框包含标题。
  const titleStr = ` ${title} `;
  const fillLen = Math.max(0, inner - 1 - titleStr.length);
  const topLine =
    dimBorder('╭─') + dimBorder(titleStr) + dimBorder('─'.repeat(fillLen)) + dimBorder('╮');

  // 下边框闭合面板。
  const bottomLine = dimBorder('╰') + dimBorder('─'.repeat(inner)) + dimBorder('╯');

  // 渲染内容行。
  const contentLines = rows.map((row) => {
    const visLen = stripAnsi(row);
    const pad = Math.max(0, inner - 1 - visLen);
    return `${dimBorder('│')} ${row}${' '.repeat(pad)}${dimBorder('│')}`;
  });

  return [topLine, ...contentLines, bottomLine].join('\n');
}

// 组合短选项和长选项。

/**
 * 合并选项长短名称与参数占位符。
 * @param opt - Commander 选项定义。
 * @returns 选项名称及参数占位符。
 */
function formatOptionTerm(opt: Option): string {
  const parts: string[] = [];
  if (opt.short) parts.push(opt.short);
  if (opt.long) parts.push(opt.long);
  let term = parts.join(', ');

  // 非布尔选项追加参数占位符。
  if (opt.flags) {
    const match = opt.flags.match(/<[^>]+>|\[[^\]]+\]/);
    if (match) {
      term += ` ${match[0]}`;
    }
  }
  return term;
}

// 通过长选项查找面板。

/**
 * 取得长选项名称；仅有短名称时使用短名称。
 * @param opt - Commander 选项定义。
 * @returns 可用的选项名称，没有名称时为空字符串。
 */
function getLongFlag(opt: Option): string {
  if (opt.long) return opt.long;
  return opt.short || '';
}

// 格式化默认值。

/**
 * 生成可显示的选项默认值说明。
 * @param opt - Commander 选项定义。
 * @returns 默认值说明；未设置或值为假时为空字符串。
 */
function formatDefault(opt: Option): string {
  if (opt.defaultValue !== undefined && opt.defaultValue !== false) {
    return dim(` [default: ${opt.defaultValue}]`);
  }
  return '';
}

// 生成完整帮助内容。

/**
 * 按命令与选项分组生成终端帮助面板。
 * @param cmd - 待生成帮助的命令。
 * @param helper - Commander 帮助格式化器。
 * @returns 完整帮助文本。
 */
export function richFormatHelp(cmd: Command, helper: Help): string {
  const width = process.stdout.columns || 80;
  const lines: string[] = [];

  const isRoot = !cmd.parent;

  // 输出用法。
  const usage = helper.commandUsage(cmd);
  lines.push('');
  if (isRoot) {
    // 根命令的参数使用黄色，可选参数加粗。
    lines.push(
      ` ${yellow('Usage:')} ${bold(cmd.name())} ${yellow('<command>')} ${bold('[options]')}`
    );
  } else {
    // 子命令路径加粗，参数使用黄色。
    const usageParts = usage.split(' ');
    const cmdPath: string[] = [];
    const argParts: string[] = [];
    let pastCmd = false;
    for (const part of usageParts) {
      if (!pastCmd && !part.startsWith('[') && !part.startsWith('<')) {
        cmdPath.push(part);
      } else {
        pastCmd = true;
        argParts.push(part);
      }
    }
    lines.push(` ${yellow('Usage:')} ${bold(cmdPath.join(' '))} ${yellow(argParts.join(' '))}`);
  }
  lines.push('');

  // 输出描述。
  const desc = helper.commandDescription(cmd);
  if (desc) {
    // 拆分多行描述。
    const descLines = desc.split('\n');
    for (let i = 0; i < descLines.length; i++) {
      const dLine = descLines[i];
      // 首行为标题，其余非空行使用弱化颜色。
      if (i === 0 || dLine.trim() === '') {
        lines.push(` ${dLine}`);
      } else {
        lines.push(` ${dim(dLine)}`);
      }
    }
    lines.push('');
  }

  // 子命令参数面板。
  if (!isRoot) {
    const visibleArgs = helper.visibleArguments(cmd);
    if (visibleArgs.length > 0) {
      const maxLen = Math.max(...visibleArgs.map((a: Argument) => a.name().length));
      const argRows = visibleArgs.map((a: Argument) => {
        const name = cyanBold(a.name().padEnd(maxLen));
        const description = helper.argumentDescription(a);
        return ` ${name}  ${description}`;
      });
      const panel = renderPanel('Arguments', argRows, width);
      if (panel) lines.push(panel);
    }
  }

  // 按面板收集选项。
  const visibleOpts = helper.visibleOptions(cmd);
  const cmdName = cmd.name();
  const panelMap = !isRoot && OPTION_PANELS[cmdName] ? OPTION_PANELS[cmdName] : {};

  const grouped: Record<string, Option[]> = { Options: [] };
  for (const panelName of PANEL_ORDER) {
    grouped[panelName] = [];
  }

  for (const opt of visibleOpts) {
    const flag = getLongFlag(opt);
    const panel = panelMap[flag];
    if (panel && PANEL_ORDER.includes(panel)) {
      grouped[panel].push(opt);
    } else {
      grouped.Options.push(opt);
    }
  }

  // 收集子命令。
  const visibleCmds = helper.visibleCommands(cmd);

  if (isRoot) {
    // 根命令先显示选项，再显示命令分组。
    if (grouped.Options.length > 0) {
      const optRows = formatOptionRows(grouped.Options);
      const panel = renderPanel('Options', optRows, width);
      if (panel) lines.push(panel);
    }
    if (visibleCmds.length > 0) {
      const cmdMap = new Map(visibleCmds.map((c) => [c.name(), c]));
      for (const group of COMMAND_GROUPS) {
        const groupCmds = group.commands
          .map((name) => cmdMap.get(name))
          .filter((c): c is Command => c !== undefined);
        if (groupCmds.length === 0) continue;
        const maxLen = Math.max(...groupCmds.map((c) => c.name().length));
        const cmdRows = groupCmds.map((c) => {
          const name = cyanBold(c.name().padEnd(maxLen));
          const description = helper.subcommandDescription(c);
          return ` ${name}  ${description}`;
        });
        const panel = renderPanel(group.panel, cmdRows, width);
        if (panel) lines.push(panel);
      }
    }
  } else {
    // 子命令先显示选项，再显示下级命令。
    const panelSequence = ['Options', ...PANEL_ORDER];
    for (const panelName of panelSequence) {
      const opts = grouped[panelName];
      if (opts && opts.length > 0) {
        const optRows = formatOptionRows(opts);
        const panel = renderPanel(panelName, optRows, width);
        if (panel) lines.push(panel);
      }
    }
    // 渲染下级命令。
    if (visibleCmds.length > 0) {
      const maxLen = Math.max(...visibleCmds.map((c) => c.name().length));
      const cmdRows = visibleCmds.map((c) => {
        const name = cyanBold(c.name().padEnd(maxLen));
        const description = helper.subcommandDescription(c);
        return ` ${name}  ${description}`;
      });
      const panel = renderPanel('Commands', cmdRows, width);
      if (panel) lines.push(panel);
    }
  }

  lines.push('');
  return lines.join('\n');
}

// 对齐选项列。

/**
 * 对齐选项名称并拼接描述与默认值。
 * @param opts - 需要展示的 Commander 选项列表。
 * @returns 对齐后的选项行。
 */
function formatOptionRows(opts: Option[]): string[] {
  const terms = opts.map((o) => formatOptionTerm(o));
  const maxTermLen = Math.max(...terms.map((t) => t.length));

  return opts.map((opt, i) => {
    const term = cyanBold(terms[i].padEnd(maxTermLen));
    const desc = opt.description || '';
    const def = formatDefault(opt);
    return ` ${term}  ${desc}${def}`;
  });
}
