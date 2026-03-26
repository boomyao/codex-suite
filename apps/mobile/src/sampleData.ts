import type { CodexThread } from './types';

export const SAMPLE_THREADS: CodexThread[] = [
  {
    id: 'thr_mobile_shell',
    title: 'Mobile shell MVP',
    subtitle: 'Bring Codex desktop-style UI into a phone shell',
    workspace: '/Users/boomyao/lab/codex-app-fork/mobile',
    updatedAtLabel: '2m',
    backend: 'sample',
    statusLabel: 'sample',
    isStreaming: false,
    activeTurnId: null,
    messages: [
      {
        id: 'msg-1',
        role: 'system',
        title: 'System',
        body:
          'This prototype keeps the Codex desktop information density, but reorganizes it for a narrow viewport with drawers instead of permanent side rails.',
      },
      {
        id: 'msg-2',
        role: 'assistant',
        title: 'Codex Mobile',
        body:
          'The shell already models thread navigation, inspector cards, settings, and connection state. The next layer is wiring `thread/list`, `thread/read`, and `turn/start` to app-server.',
      },
      {
        id: 'msg-3',
        role: 'tool',
        title: 'Endpoint plan',
        body:
          '1. Probe the configured Codex endpoint.\n2. Initialize JSON-RPC transport.\n3. Map app-server notifications into the shell event stream.\n4. Replace sample threads with real thread state.',
      },
    ],
  },
  {
    id: 'thr_ui_parity',
    title: 'UI parity audit',
    subtitle: 'Track which Codex desktop affordances to retain',
    workspace: '/Applications/Codex.app',
    updatedAtLabel: '9m',
    backend: 'sample',
    statusLabel: 'sample',
    isStreaming: false,
    activeTurnId: null,
    messages: [
      {
        id: 'msg-4',
        role: 'assistant',
        title: 'Codex Mobile',
        body:
          'Priority parity features: thread switching, streaming output, diff review, approvals, and quick route changes. Native menus, window chrome, and local PTY are phase-two items.',
      },
    ],
  },
  {
    id: 'thr_remote_mode',
    title: 'Remote-first architecture',
    subtitle: 'Use a remote Codex endpoint before attempting full local execution',
    workspace: '/Users/boomyao/.codex',
    updatedAtLabel: '27m',
    backend: 'sample',
    statusLabel: 'sample',
    isStreaming: false,
    activeTurnId: null,
    messages: [
      {
        id: 'msg-5',
        role: 'assistant',
        title: 'Codex Mobile',
        body:
          'Mobile should initially supervise Codex, not host it. That keeps the UX close to desktop while avoiding iOS sandbox limits and Android PTY edge cases.',
      },
    ],
  },
];
