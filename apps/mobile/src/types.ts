export type ThemeVariant = 'dark' | 'light';

export type ThreadMessageRole = 'assistant' | 'user' | 'tool' | 'system';
export type ThreadMessageKind =
  | 'plain'
  | 'plan'
  | 'reasoning'
  | 'command'
  | 'fileChange'
  | 'toolCall'
  | 'review'
  | 'image'
  | 'search';

export interface CodexMessage {
  id: string;
  itemId?: string;
  role: ThreadMessageRole;
  kind?: ThreadMessageKind;
  title: string;
  statusLabel?: string;
  auxiliaryLines?: string[];
  body: string;
}

export interface LiveStatusEntry {
  summary: string;
  detail?: string;
}

export interface PendingRequestOption {
  id: string;
  label: string;
  tone?: 'primary' | 'secondary' | 'danger';
}

export interface PendingRequestQuestion {
  id: string;
  header: string;
  prompt: string;
  options?: string[];
  allowsOther?: boolean;
  isSecret?: boolean;
}

export interface PendingServerRequest {
  requestId: string;
  method: string;
  kind: 'approval' | 'question' | 'permissions' | 'elicitation' | 'auth' | 'unsupported';
  title: string;
  detail: string;
  contextLines: string[];
  options: PendingRequestOption[];
  questions?: PendingRequestQuestion[];
}

export interface ThreadLiveState {
  diff: LiveStatusEntry | null;
  tokenUsage: LiveStatusEntry | null;
}

export interface ShellLiveStatus {
  account: LiveStatusEntry | null;
  rateLimit: LiveStatusEntry | null;
}

export interface CodexThread {
  id: string;
  title: string;
  subtitle: string;
  workspace: string;
  updatedAtLabel: string;
  backend: 'sample' | 'app-server';
  statusLabel?: string;
  isStreaming?: boolean;
  activeTurnId?: string | null;
  liveState?: ThreadLiveState | null;
  messages: CodexMessage[];
}
