import { beforeEach } from 'vitest';
import { createInvocationState, enterInvocation } from '../src/runtime/state.js';
/** 每个测试显式装配独立的调用上下文。 */
beforeEach(() => enterInvocation(createInvocationState()));
