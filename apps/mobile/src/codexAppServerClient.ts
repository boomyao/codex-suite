import type {
  AppServerInitializeResponse,
  AppServerServerRequest,
  AppServerServerNotification,
  AppServerThreadListResponse,
  AppServerThreadReadResponse,
  AppServerThreadResumeResponse,
  AppServerThreadStartResponse,
  AppServerTurnStartResponse,
} from './appServerProtocol';

type PendingRequest = {
  resolve: (value: unknown) => void;
  reject: (error: Error) => void;
};

type JsonRpcResponse = {
  id: number | string;
  result?: unknown;
  error?: {
    code?: number;
    message?: string;
    data?: unknown;
  };
};

export interface CodexAppServerClientOptions {
  endpoint: string;
  headers?: Record<string, string>;
  onNotification?: (notification: AppServerServerNotification) => void;
  onServerRequest?: (request: AppServerServerRequest) => Promise<unknown> | unknown;
  onClose?: () => void;
  onError?: (error: Error) => void;
}

export class CodexAppServerClient {
  private endpoint: string;
  private headers?: Record<string, string>;
  private onNotification?: (notification: AppServerServerNotification) => void;
  private onServerRequest?: (request: AppServerServerRequest) => Promise<unknown> | unknown;
  private onClose?: () => void;
  private onError?: (error: Error) => void;
  private socket: WebSocket | null = null;
  private nextId = 1;
  private pending = new Map<number | string, PendingRequest>();

  constructor(options: CodexAppServerClientOptions) {
    this.endpoint = options.endpoint;
    this.headers = options.headers;
    this.onNotification = options.onNotification;
    this.onServerRequest = options.onServerRequest;
    this.onClose = options.onClose;
    this.onError = options.onError;
  }

  async connect(): Promise<AppServerInitializeResponse> {
    if (this.socket && this.socket.readyState === WebSocket.OPEN) {
      throw new Error('App-server socket is already connected.');
    }

    if (this.socket && this.socket.readyState === WebSocket.CONNECTING) {
      throw new Error('App-server socket connection is already in progress.');
    }

    const socket =
      this.headers && Object.keys(this.headers).length > 0
        ? new (WebSocket as unknown as {
            new (
              url: string,
              protocols: string[] | undefined,
              options: { headers: Record<string, string> },
            ): WebSocket;
          })(this.endpoint, undefined, {
            headers: this.headers,
          })
        : new WebSocket(this.endpoint);
    this.socket = socket;
    let opened = false;

    const waitForOpen = new Promise<void>((resolve, reject) => {
      socket.onopen = () => {
        if (this.socket !== socket) {
          return;
        }
        opened = true;
        resolve();
      };
      socket.onerror = () => {
        if (this.socket !== socket) {
          return;
        }
        if (!opened) {
          reject(new Error('WebSocket connection failed.'));
          return;
        }
        this.onError?.(new Error('WebSocket transport error.'));
      };
      socket.onclose = () => {
        if (this.socket !== socket) {
          return;
        }
        this.socket = null;
        this.rejectAllPending(new Error('Connection closed.'));
        if (!opened) {
          reject(new Error('WebSocket connection closed before initialization.'));
          return;
        }
        this.onClose?.();
      };
    });

    try {
      await waitForOpen;
    } catch (error) {
      if (this.socket === socket) {
        this.socket = null;
      }
      try {
        socket.close();
      } catch {
        // Ignore close failures during failed connection cleanup.
      }
      throw error;
    }

    if (this.socket !== socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error('WebSocket connection is no longer open.');
    }

    socket.onmessage = (event) => {
      if (this.socket !== socket) {
        return;
      }
      this.handleMessage(event.data);
    };

    const initializeResponse = await this.request<AppServerInitializeResponse>(
      'initialize',
      {
        clientInfo: {
          name: 'codex_mobile_mvp',
          title: 'Codex Mobile MVP',
          version: '0.1.0',
        },
        capabilities: {
          experimentalApi: true,
        },
      },
    );
    this.notify('initialized');
    return initializeResponse;
  }

  disconnect() {
    this.socket?.close();
  }

  isConnected() {
    return this.socket?.readyState === WebSocket.OPEN;
  }

  listThreads() {
    return this.request<AppServerThreadListResponse>('thread/list', {
      limit: 50,
      sortKey: 'updated_at',
    });
  }

  readThread(threadId: string) {
    return this.request<AppServerThreadReadResponse>('thread/read', {
      threadId,
      includeTurns: true,
    });
  }

  resumeThread(threadId: string) {
    return this.request<AppServerThreadResumeResponse>('thread/resume', {
      threadId,
    });
  }

  startThread(cwd?: string | null) {
    return this.request<AppServerThreadStartResponse>('thread/start', {
      cwd: cwd ?? null,
      personality: 'pragmatic',
      experimentalRawEvents: false,
    });
  }

  startTurn(threadId: string, text: string) {
    return this.request<AppServerTurnStartResponse>('turn/start', {
      threadId,
      input: [
        {
          type: 'text',
          text,
          text_elements: [],
        },
      ],
      personality: 'pragmatic',
    });
  }

  interruptTurn(threadId: string, turnId: string) {
    return this.request<{}>('turn/interrupt', {
      threadId,
      turnId,
    });
  }

  readAccount() {
    return this.request<unknown>('account/read', {});
  }

  readRateLimits() {
    return this.request<unknown>('account/rateLimits/read', {});
  }

  sendRequest<T = unknown>(method: string, params: unknown) {
    return this.request<T>(method, params);
  }

  private request<T>(method: string, params: unknown): Promise<T> {
    const socket = this.requireConnectedSocket();
    const id = this.nextId++;
    const payload = { id, method, params };
    return new Promise<T>((resolve, reject) => {
      this.pending.set(id, { resolve: (value) => resolve(value as T), reject });
      try {
        this.send(socket, payload);
      } catch (error) {
        this.pending.delete(id);
        reject(error instanceof Error ? error : new Error('Failed to send request.'));
      }
    });
  }

  private notify(method: string, params?: unknown) {
    const payload = params === undefined ? { method } : { method, params };
    const socket = this.requireConnectedSocket();
    this.send(socket, payload);
  }

  private send(socket: WebSocket, payload: unknown) {
    if (this.socket !== socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error('App-server socket is not connected.');
    }
    socket.send(JSON.stringify(payload));
  }

  private handleMessage(raw: unknown) {
    if (typeof raw !== 'string') {
      return;
    }

    let message:
      | JsonRpcResponse
      | AppServerServerNotification
      | AppServerServerRequest;
    try {
      message = JSON.parse(raw) as
        | JsonRpcResponse
        | AppServerServerNotification
        | AppServerServerRequest;
    } catch {
      return;
    }

    if ('id' in message && 'method' in message) {
      void this.handleServerRequest(message);
      return;
    }

    if ('id' in message) {
      const pending = this.pending.get(message.id);
      if (!pending) {
        return;
      }
      this.pending.delete(message.id);
      if (message.error) {
        pending.reject(
          new Error(message.error.message ?? 'Unknown app-server error.'),
        );
        return;
      }
      pending.resolve(message.result);
      return;
    }

    this.onNotification?.(message);
  }

  private async handleServerRequest(request: AppServerServerRequest) {
    const socket = this.socket;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      return;
    }

    if (!this.onServerRequest) {
      this.respondWithError(
        socket,
        request.id,
        'Mobile client does not handle app-server server requests yet.',
      );
      return;
    }

    try {
      const result = await this.onServerRequest(request);
      if (this.socket !== socket || socket.readyState !== WebSocket.OPEN) {
        return;
      }
      this.respondWithResult(socket, request.id, result ?? {});
    } catch (error) {
      if (this.socket !== socket || socket.readyState !== WebSocket.OPEN) {
        return;
      }
      this.respondWithError(
        socket,
        request.id,
        error instanceof Error ? error.message : 'Server request failed.',
      );
    }
  }

  private requireConnectedSocket() {
    const socket = this.socket;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      throw new Error('App-server socket is not connected.');
    }
    return socket;
  }

  private respondWithResult(
    socket: WebSocket,
    id: number | string,
    result: unknown,
  ) {
    this.send(socket, { id, result });
  }

  private respondWithError(
    socket: WebSocket,
    id: number | string,
    message: string,
  ) {
    this.send(socket, {
      id,
      error: {
        message,
      },
    });
  }

  private rejectAllPending(error: Error) {
    for (const pending of this.pending.values()) {
      pending.reject(error);
    }
    this.pending.clear();
  }
}
