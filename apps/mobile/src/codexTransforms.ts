import type {
  AppServerThread,
  AppServerThreadItem,
  AppServerThreadStatus,
  AppServerTurn,
  AppServerUserInput,
} from './appServerProtocol';
import type { CodexMessage, CodexThread, ThreadMessageKind, ThreadMessageRole } from './types';

function formatRelativeTime(unixSeconds: number): string {
  const diffSeconds = Math.max(0, Math.floor(Date.now() / 1000) - unixSeconds);
  if (diffSeconds < 60) {
    return `${diffSeconds}s`;
  }
  if (diffSeconds < 3600) {
    return `${Math.floor(diffSeconds / 60)}m`;
  }
  if (diffSeconds < 86400) {
    return `${Math.floor(diffSeconds / 3600)}h`;
  }
  return `${Math.floor(diffSeconds / 86400)}d`;
}

function statusLabel(status: AppServerThreadStatus): string {
  switch (status.type) {
    case 'active':
      return status.activeFlags?.length
        ? `active · ${status.activeFlags.join(', ')}`
        : 'active';
    case 'systemError':
      return 'system error';
    case 'notLoaded':
      return 'not loaded';
    default:
      return 'idle';
  }
}

function userInputToText(input: AppServerUserInput): string {
  switch (input.type) {
    case 'text':
      return input.text;
    case 'mention':
      return `@${input.name}`;
    case 'skill':
      return `$${input.name}`;
    case 'image':
      return input.url;
    case 'localImage':
      return input.path;
    default:
      return '';
  }
}

function compactJson(value: unknown): string {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function firstString(value: unknown, keys: string[]): string | null {
  if (!isRecord(value)) {
    return null;
  }

  for (const key of keys) {
    const entry = value[key];
    if (typeof entry === 'string' && entry.trim()) {
      return entry.trim();
    }
  }

  return null;
}

function summarizeTokenUsage(value: unknown): string | null {
  const candidates = [value, isRecord(value) ? value.usage : null, isRecord(value) ? value.tokenUsage : null];

  for (const candidate of candidates) {
    if (!isRecord(candidate)) {
      continue;
    }

    const promptTokens = candidate.promptTokens ?? candidate.inputTokens;
    const completionTokens = candidate.completionTokens ?? candidate.outputTokens;
    const totalTokens = candidate.totalTokens ?? candidate.tokens;

    if (
      typeof promptTokens === 'number' ||
      typeof completionTokens === 'number' ||
      typeof totalTokens === 'number'
    ) {
      const pieces = [
        typeof promptTokens === 'number' ? `prompt ${promptTokens}` : null,
        typeof completionTokens === 'number' ? `completion ${completionTokens}` : null,
        typeof totalTokens === 'number' ? `total ${totalTokens}` : null,
      ].filter(Boolean);

      return `tokens: ${pieces.join(' · ')}`;
    }
  }

  return null;
}

function summarizeStructuredValue(value: unknown): string[] {
  if (value == null) {
    return [];
  }

  if (typeof value === 'string') {
    return value.trim() ? [value.trim()] : [];
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return [String(value)];
  }

  if (Array.isArray(value)) {
    return value.slice(0, 3).flatMap((entry) => summarizeStructuredValue(entry)).slice(0, 4);
  }

  if (!isRecord(value)) {
    return [compactJson(value)];
  }

  const lines: string[] = [];
  const path = firstString(value, ['path', 'filePath', 'filename', 'file', 'cwd']);
  const label = firstString(value, ['name', 'kind', 'type', 'status', 'action', 'operation']);
  const diffText = firstString(value, ['diff', 'patch']);
  const tokens = summarizeTokenUsage(value);

  if (path) {
    lines.push(`file: ${path}`);
  }

  if (label) {
    lines.push(label);
  }

  if (tokens) {
    lines.push(tokens);
  }

  if (diffText) {
    const diffLines = diffText
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean);

    if (diffLines.length > 0) {
      lines.push(`diff: ${diffLines[0]}`);
      if (diffLines.length > 1) {
        lines.push(diffLines[1]);
      }
    }
  }

  if (lines.length > 0) {
    return lines.slice(0, 4);
  }

  return [compactJson(value)];
}

function summarizeChange(change: unknown): string {
  const lines = summarizeStructuredValue(change);
  if (lines.length === 0) {
    return '';
  }

  return lines.join(' · ');
}

export function appServerThreadToSummary(thread: AppServerThread): CodexThread {
  return {
    id: thread.id,
    title:
      thread.name ??
      thread.preview ??
      thread.cwd.split('/').pop() ??
      'Untitled thread',
    subtitle: thread.preview || statusLabel(thread.status),
    workspace: thread.cwd,
    updatedAtLabel: formatRelativeTime(thread.updatedAt),
    backend: 'app-server',
    statusLabel: statusLabel(thread.status),
    isStreaming: thread.status.type === 'active',
    activeTurnId: null,
    messages: [],
  };
}

export function appServerThreadToCodexThread(thread: AppServerThread): CodexThread {
  const summary = appServerThreadToSummary(thread);
  const turns = [...thread.turns];
  const latestTurn = turns.at(-1) ?? null;

  return {
    ...summary,
    isStreaming: latestTurn?.status === 'inProgress' || thread.status.type === 'active',
    activeTurnId: latestTurn?.status === 'inProgress' ? latestTurn.id : null,
    messages: turns.flatMap((turn) => turn.items.map(threadItemToCodexMessage)).filter(Boolean) as CodexMessage[],
  };
}

export function threadItemToCodexMessage(item: AppServerThreadItem): CodexMessage | null {
  switch (item.type) {
    case 'userMessage':
      return {
        id: item.id,
        itemId: item.id,
        role: 'user',
        kind: 'plain',
        title: 'You',
        body: item.content.map(userInputToText).filter(Boolean).join('\n'),
      };
    case 'agentMessage':
      return {
        id: item.id,
        itemId: item.id,
        role: 'assistant',
        kind: 'plain',
        title: 'Codex',
        statusLabel: item.phase ?? undefined,
        body: item.text,
      };
    case 'plan':
      return {
        id: item.id,
        itemId: item.id,
        role: 'assistant',
        kind: 'plan',
        title: 'Plan',
        body: item.text,
      };
    case 'reasoning':
      return {
        id: item.id,
        itemId: item.id,
        role: 'system',
        kind: 'reasoning',
        title: 'Reasoning',
        auxiliaryLines: item.summary,
        body: [...item.summary, ...item.content].join('\n'),
      };
    case 'commandExecution':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'command',
        title: 'Command',
        statusLabel: item.status,
        auxiliaryLines: [
          item.cwd,
          item.durationMs != null ? `${item.durationMs} ms` : '',
          item.exitCode != null ? `exit ${item.exitCode}` : '',
        ].filter(Boolean),
        body: [item.command, item.aggregatedOutput ?? '']
          .filter(Boolean)
          .join('\n\n'),
      };
    case 'fileChange':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'fileChange',
        title: 'File change',
        statusLabel: item.status,
        auxiliaryLines: item.changes.slice(0, 4).map((change) => summarizeChange(change)).filter(Boolean),
        body: `${item.changes.length} change(s) staged in this item.`,
      };
    case 'mcpToolCall':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'toolCall',
        title: `MCP · ${item.server}/${item.tool}`,
        statusLabel: item.status,
        auxiliaryLines: [
          item.durationMs != null ? `${item.durationMs} ms` : '',
          ...summarizeStructuredValue(item.error ?? item.result),
        ].filter(Boolean),
        body: item.error ? compactJson(item.error) : compactJson(item.result ?? item.status),
      };
    case 'dynamicToolCall':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'toolCall',
        title: `Tool · ${item.tool}`,
        statusLabel: item.status,
        auxiliaryLines: [
          item.durationMs != null ? `${item.durationMs} ms` : '',
          ...summarizeStructuredValue(item.contentItems ?? item.status),
        ].filter(Boolean),
        body: compactJson(item.contentItems ?? item.status),
      };
    case 'collabAgentToolCall':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'toolCall',
        title: `Agent tool · ${item.tool}`,
        statusLabel: item.status,
        auxiliaryLines: item.receiverThreadIds,
        body: [item.prompt ?? '', item.model ?? '', item.receiverThreadIds.join(', ')].filter(Boolean).join('\n'),
      };
    case 'webSearch':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'search',
        title: 'Web search',
        body: item.query,
      };
    case 'imageView':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'image',
        title: 'Image view',
        body: item.path,
      };
    case 'imageGeneration':
      return {
        id: item.id,
        itemId: item.id,
        role: 'tool',
        kind: 'image',
        title: 'Image generation',
        statusLabel: item.status,
        body: item.revisedPrompt ?? item.result,
      };
    case 'enteredReviewMode':
      return {
        id: item.id,
        itemId: item.id,
        role: 'system',
        kind: 'review',
        title: 'Review mode',
        body: item.review,
      };
    case 'exitedReviewMode':
      return {
        id: item.id,
        itemId: item.id,
        role: 'system',
        kind: 'review',
        title: 'Exited review mode',
        body: item.review,
      };
    case 'contextCompaction':
      return {
        id: item.id,
        itemId: item.id,
        role: 'system',
        kind: 'plain',
        title: 'Context compaction',
        body: 'Thread context was compacted.',
      };
    default:
      return null;
  }
}

export function upsertThread(threads: CodexThread[], next: CodexThread): CodexThread[] {
  const existingIndex = threads.findIndex((thread) => thread.id === next.id);
  if (existingIndex === -1) {
    return [next, ...threads];
  }
  const updated = [...threads];
  updated[existingIndex] = {
    ...updated[existingIndex],
    ...next,
    liveState: next.liveState ?? updated[existingIndex].liveState ?? null,
    messages: next.messages.length > 0 ? next.messages : updated[existingIndex].messages,
  };
  return updated;
}

function joinDelta(current: string, delta: string): string {
  if (!current) {
    return delta;
  }
  if (!delta) {
    return current;
  }
  return `${current}${delta}`;
}

function upsertSyntheticMessage(
  thread: CodexThread,
  itemId: string,
  seed: {
    role: ThreadMessageRole;
    kind?: ThreadMessageKind;
    title: string;
    body?: string;
    statusLabel?: string;
    auxiliaryLines?: string[];
  },
  updater: (current: CodexMessage) => CodexMessage,
): CodexThread {
  const nextMessages = [...thread.messages];
  const existingIndex = nextMessages.findIndex(
    (message) => message.itemId === itemId || message.id === itemId,
  );

  const currentMessage: CodexMessage =
    existingIndex === -1
      ? {
          id: itemId,
          itemId,
          role: seed.role,
          kind: seed.kind,
          title: seed.title,
          statusLabel: seed.statusLabel,
          auxiliaryLines: seed.auxiliaryLines,
          body: seed.body ?? '',
        }
      : nextMessages[existingIndex];

  const nextMessage = updater(currentMessage);
  if (existingIndex === -1) {
    nextMessages.push(nextMessage);
  } else {
    nextMessages[existingIndex] = nextMessage;
  }

  return {
    ...thread,
    updatedAtLabel: 'now',
    messages: nextMessages,
  };
}

export function appendItemBodyDelta(
  thread: CodexThread,
  itemId: string,
  delta: string,
  seed: {
    role: ThreadMessageRole;
    kind?: ThreadMessageKind;
    title: string;
    statusLabel?: string;
    auxiliaryLines?: string[];
  },
): CodexThread {
  return upsertSyntheticMessage(thread, itemId, seed, (current) => ({
    ...current,
    role: seed.role,
    kind: seed.kind ?? current.kind,
    title: seed.title,
    statusLabel: seed.statusLabel ?? current.statusLabel,
    auxiliaryLines: seed.auxiliaryLines ?? current.auxiliaryLines,
    body: joinDelta(current.body, delta),
  }));
}

export function appendReasoningSummaryDelta(
  thread: CodexThread,
  itemId: string,
  summaryIndex: number,
  delta: string,
): CodexThread {
  return upsertSyntheticMessage(
    thread,
    itemId,
    {
      role: 'system',
      kind: 'reasoning',
      title: 'Reasoning',
      auxiliaryLines: [],
    },
    (current) => {
      const nextAuxiliaryLines = [...(current.auxiliaryLines ?? [])];
      while (nextAuxiliaryLines.length <= summaryIndex) {
        nextAuxiliaryLines.push('');
      }
      nextAuxiliaryLines[summaryIndex] = joinDelta(
        nextAuxiliaryLines[summaryIndex] ?? '',
        delta,
      );
      const summaryBlock = nextAuxiliaryLines.filter(Boolean).join('\n');
      const detailBlock = current.body
        .split('\n')
        .slice(nextAuxiliaryLines.length)
        .join('\n')
        .trim();
      return {
        ...current,
        auxiliaryLines: nextAuxiliaryLines,
        body: [summaryBlock, detailBlock].filter(Boolean).join('\n'),
      };
    },
  );
}

export function appendReasoningContentDelta(
  thread: CodexThread,
  itemId: string,
  contentIndex: number,
  delta: string,
): CodexThread {
  return upsertSyntheticMessage(
    thread,
    itemId,
    {
      role: 'system',
      kind: 'reasoning',
      title: 'Reasoning',
    },
    (current) => {
      const summaryLines = current.auxiliaryLines ?? [];
      const detailLines = current.body
        .split('\n')
        .slice(summaryLines.length)
        .filter((line) => line.length > 0);
      while (detailLines.length <= contentIndex) {
        detailLines.push('');
      }
      detailLines[contentIndex] = joinDelta(detailLines[contentIndex] ?? '', delta);
      return {
        ...current,
        body: [...summaryLines, ...detailLines].filter(Boolean).join('\n'),
      };
    },
  );
}

export function addReasoningSummaryPlaceholder(
  thread: CodexThread,
  itemId: string,
  summaryIndex: number,
): CodexThread {
  return upsertSyntheticMessage(
    thread,
    itemId,
    {
      role: 'system',
      kind: 'reasoning',
      title: 'Reasoning',
      auxiliaryLines: [],
    },
    (current) => {
      const nextAuxiliaryLines = [...(current.auxiliaryLines ?? [])];
      while (nextAuxiliaryLines.length <= summaryIndex) {
        nextAuxiliaryLines.push('');
      }
      return {
        ...current,
        auxiliaryLines: nextAuxiliaryLines,
      };
    },
  );
}

export function upsertTurnPlan(
  thread: CodexThread,
  turnId: string,
  explanation: string | null,
  plan: Array<{ step: string; status: string }>,
): CodexThread {
  const body = [
    explanation ?? '',
    ...plan.map((entry) => `${entry.status}: ${entry.step}`),
  ]
    .filter(Boolean)
    .join('\n');

  return upsertSyntheticMessage(
    thread,
    `turn-plan-${turnId}`,
    {
      role: 'assistant',
      kind: 'plan',
      title: 'Plan',
      body,
    },
    () => ({
      id: `turn-plan-${turnId}`,
      itemId: `turn-plan-${turnId}`,
      role: 'assistant',
      kind: 'plan',
      title: 'Plan',
      body,
    }),
  );
}

export function applyThreadItemToThread(
  thread: CodexThread,
  item: AppServerThreadItem,
): CodexThread {
  const message = threadItemToCodexMessage(item);
  if (!message) {
    return thread;
  }
  const existingIndex = thread.messages.findIndex(
    (entry) => entry.itemId === item.id || entry.id === item.id,
  );
  const nextMessages = [...thread.messages];
  if (existingIndex === -1) {
    nextMessages.push(message);
  } else {
    nextMessages[existingIndex] = message;
  }
  return {
    ...thread,
    updatedAtLabel: 'now',
    messages: nextMessages,
  };
}

export function appendAgentDelta(
  thread: CodexThread,
  itemId: string,
  delta: string,
  turnId: string,
): CodexThread {
  const existingIndex = thread.messages.findIndex(
    (message) => message.itemId === itemId || message.id === itemId,
  );
  const nextMessages = [...thread.messages];
  if (existingIndex === -1) {
    nextMessages.push({
      id: itemId,
      itemId,
      role: 'assistant',
      kind: 'plain',
      title: 'Codex',
      body: delta,
    });
  } else {
    nextMessages[existingIndex] = {
      ...nextMessages[existingIndex],
      body: `${nextMessages[existingIndex].body}${delta}`,
    };
  }
  return {
    ...thread,
    updatedAtLabel: 'now',
    isStreaming: true,
    activeTurnId: turnId,
    messages: nextMessages,
  };
}

export function markThreadTurnState(
  thread: CodexThread,
  turn: AppServerTurn,
): CodexThread {
  return {
    ...thread,
    updatedAtLabel: 'now',
    isStreaming: turn.status === 'inProgress',
    activeTurnId: turn.status === 'inProgress' ? turn.id : null,
  };
}
