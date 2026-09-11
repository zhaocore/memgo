#!/usr/bin/env node
/** CLI 进程入口；命令和领域模块不直接结束进程。 */
import { runCli } from './cli/run.js';
process.exitCode = await runCli(process.argv);
