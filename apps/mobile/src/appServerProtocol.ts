export interface AppServerInitializeResponse {
  userAgent: string;
  platformFamily: string;
  platformOs: string;
}

export type AppServerRequestId = number | string;

export interface AppServerThreadStatus {
  type: 'notLoaded' | 'idle' | 'systemError' | 'active';
  activeFlags?: string[];
}

export interface AppServerTextInput {
  type: 'text';
  text: string;
  text_elements: Array<unknown>;
}

export type AppServerUserInput =
  | AppServerTextInput
  | { type: 'image'; url: string }
  | { type: 'localImage'; path: string }
  | { type: 'skill'; name: string; path: string }
  | { type: 'mention'; name: string; path: string };

export type AppServerThreadItem =
  | {
      type: 'userMessage';
      id: string;
      content: AppServerUserInput[];
    }
  | {
      type: 'agentMessage';
      id: string;
      text: string;
      phase: string | null;
      memoryCitation: unknown | null;
    }
  | {
      type: 'plan';
      id: string;
      text: string;
    }
  | {
      type: 'reasoning';
      id: string;
      summary: string[];
      content: string[];
    }
  | {
      type: 'commandExecution';
      id: string;
      command: string;
      cwd: string;
      processId: string | null;
      status: string;
      aggregatedOutput: string | null;
      exitCode: number | null;
      durationMs: number | null;
    }
  | {
      type: 'fileChange';
      id: string;
      changes: Array<unknown>;
      status: string;
    }
  | {
      type: 'mcpToolCall';
      id: string;
      server: string;
      tool: string;
      status: string;
      result: unknown | null;
      error: unknown | null;
      durationMs: number | null;
    }
  | {
      type: 'dynamicToolCall';
      id: string;
      tool: string;
      status: string;
      contentItems: Array<unknown> | null;
      success: boolean | null;
      durationMs: number | null;
    }
  | {
      type: 'collabAgentToolCall';
      id: string;
      tool: string;
      status: string;
      senderThreadId: string;
      receiverThreadIds: string[];
      prompt: string | null;
      model: string | null;
    }
  | {
      type: 'webSearch';
      id: string;
      query: string;
      action: unknown | null;
    }
  | {
      type: 'imageView';
      id: string;
      path: string;
    }
  | {
      type: 'imageGeneration';
      id: string;
      status: string;
      revisedPrompt: string | null;
      result: string;
    }
  | {
      type: 'enteredReviewMode';
      id: string;
      review: string;
    }
  | {
      type: 'exitedReviewMode';
      id: string;
      review: string;
    }
  | {
      type: 'contextCompaction';
      id: string;
    };

export interface AppServerTurn {
  id: string;
  items: AppServerThreadItem[];
  status: 'completed' | 'interrupted' | 'failed' | 'inProgress';
  error: { message?: string } | null;
}

export interface AppServerThread {
  id: string;
  preview: string;
  ephemeral: boolean;
  modelProvider: string;
  createdAt: number;
  updatedAt: number;
  status: AppServerThreadStatus;
  path: string | null;
  cwd: string;
  cliVersion: string;
  source: string | { custom: string } | { subAgent: unknown };
  agentNickname: string | null;
  agentRole: string | null;
  gitInfo: { branch?: string | null } | null;
  name: string | null;
  turns: AppServerTurn[];
}

export interface AppServerThreadListResponse {
  data: AppServerThread[];
  nextCursor: string | null;
}

export interface AppServerThreadReadResponse {
  thread: AppServerThread;
}

export interface AppServerThreadResumeResponse {
  thread: AppServerThread;
  model: string;
  modelProvider: string;
  cwd: string;
}

export interface AppServerThreadStartResponse {
  thread: AppServerThread;
  model: string;
  modelProvider: string;
  cwd: string;
}

export interface AppServerTurnStartResponse {
  turn: AppServerTurn;
}

export type AppServerServerRequest =
  | {
      id: AppServerRequestId;
      method: 'item/commandExecution/requestApproval';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        approvalId?: string | null;
        reason?: string | null;
        command?: string | null;
        cwd?: string | null;
        availableDecisions?: unknown[] | null;
        [key: string]: unknown;
      };
    }
  | {
      id: AppServerRequestId;
      method: 'item/fileChange/requestApproval';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        reason?: string | null;
        grantRoot?: string | null;
        [key: string]: unknown;
      };
    }
  | {
      id: AppServerRequestId;
      method: 'item/tool/requestUserInput';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        questions: Array<{
          id: string;
          header: string;
          question: string;
          isOther?: boolean;
          isSecret?: boolean;
          options?: Array<{ label: string; description?: string }> | null;
        }>;
      };
    }
  | {
      id: AppServerRequestId;
      method: 'mcpServer/elicitation/request';
      params: {
        threadId: string;
        turnId: string | null;
        serverName: string;
        mode: 'form' | 'url';
        message: string;
        requestedSchema?: unknown;
        url?: string;
        elicitationId?: string;
        [key: string]: unknown;
      };
    }
  | {
      id: AppServerRequestId;
      method: 'item/permissions/requestApproval';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        reason?: string | null;
        permissions?: unknown;
        [key: string]: unknown;
      };
    }
  | {
      id: AppServerRequestId;
      method: 'item/tool/call';
      params: Record<string, unknown>;
    }
  | {
      id: AppServerRequestId;
      method: 'account/chatgptAuthTokens/refresh';
      params: Record<string, unknown>;
    }
  | {
      id: AppServerRequestId;
      method: 'applyPatchApproval';
      params: {
        conversationId: string;
        callId: string;
        reason?: string | null;
        grantRoot?: string | null;
        [key: string]: unknown;
      };
    }
  | {
      id: AppServerRequestId;
      method: 'execCommandApproval';
      params: {
        conversationId: string;
        callId: string;
        approvalId?: string | null;
        command?: string[];
        cwd?: string;
        reason?: string | null;
        [key: string]: unknown;
      };
    };

export type AppServerServerNotification =
  | { method: 'thread/started'; params: { thread: AppServerThread } }
  | { method: 'thread/status/changed'; params: { threadId: string; status: AppServerThreadStatus } }
  | { method: 'turn/started'; params: { threadId: string; turn: AppServerTurn } }
  | { method: 'turn/completed'; params: { threadId: string; turn: AppServerTurn } }
  | { method: 'turn/diff/updated'; params: { threadId: string; turnId: string; diff: unknown } }
  | {
      method: 'turn/plan/updated';
      params: {
        threadId: string;
        turnId: string;
        explanation: string | null;
        plan: Array<{ step: string; status: string }>;
      };
    }
  | { method: 'turn/diff/changed'; params: { threadId: string; turnId: string; diff: unknown } }
  | { method: 'account/updated'; params: { account: unknown } }
  | { method: 'account/changed'; params: { account: unknown } }
  | { method: 'account/rateLimits/updated'; params: { rateLimit: unknown } }
  | { method: 'rate-limit/updated'; params: { rateLimit: unknown } }
  | { method: 'rate-limit/changed'; params: { rateLimit: unknown } }
  | { method: 'rateLimit/updated'; params: { rateLimit: unknown } }
  | { method: 'rateLimit/changed'; params: { rateLimit: unknown } }
  | { method: 'thread/token-usage/updated'; params: { threadId: string; tokenUsage: unknown } }
  | { method: 'thread/token-usage/changed'; params: { threadId: string; tokenUsage: unknown } }
  | { method: 'thread/tokenUsage/updated'; params: { threadId: string; tokenUsage: unknown } }
  | { method: 'thread/tokenUsage/changed'; params: { threadId: string; tokenUsage: unknown } }
  | {
      method: 'item/started';
      params: { threadId: string; turnId: string; item: AppServerThreadItem };
    }
  | {
      method: 'item/completed';
      params: { threadId: string; turnId: string; item: AppServerThreadItem };
    }
  | {
      method: 'item/agentMessage/delta';
      params: { threadId: string; turnId: string; itemId: string; delta: string };
    }
  | {
      method: 'item/plan/delta';
      params: { threadId: string; turnId: string; itemId: string; delta: string };
    }
  | {
      method: 'item/reasoning/summaryTextDelta';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        summaryIndex: number;
        delta: string;
      };
    }
  | {
      method: 'item/reasoning/summaryPartAdded';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        summaryIndex: number;
      };
    }
  | {
      method: 'item/reasoning/textDelta';
      params: {
        threadId: string;
        turnId: string;
        itemId: string;
        contentIndex: number;
        delta: string;
      };
    }
  | {
      method: 'item/commandExecution/outputDelta';
      params: { threadId: string; turnId: string; itemId: string; delta: string };
    }
  | {
      method: 'item/fileChange/outputDelta';
      params: { threadId: string; turnId: string; itemId: string; delta: string };
    }
  | {
      method: 'serverRequest/resolved';
      params: { threadId: string; requestId: AppServerRequestId };
    }
  | { method: 'error'; params: { message?: string } };
