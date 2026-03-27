import AsyncStorage from '@react-native-async-storage/async-storage';
import { StatusBar } from 'expo-status-bar';
import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Linking,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  useColorScheme,
  useWindowDimensions,
  View,
} from 'react-native';
import { WebView, type WebViewMessageEvent } from 'react-native-webview';

import { DEFAULT_SERVER_ENDPOINT } from './src/appServer';
import type {
  AppServerRequestId,
  AppServerServerNotification,
  AppServerThreadReadResponse,
  AppServerThreadStatus,
  AppServerTurn,
} from './src/appServerProtocol';
import { CodexAppServerClient } from './src/codexAppServerClient';

const STORAGE_KEY = 'codex-mobile.remote-shell.preferences.v5';
const LEGACY_STORAGE_KEYS = [
  'codex-mobile.remote-shell.preferences.v4',
  'codex-mobile.remote-shell.preferences.v3',
  'codex-mobile.desktop-shell.preferences.v2',
];
const LOCAL_SERVER_DEBUG_ENDPOINT = DEFAULT_SERVER_ENDPOINT;
const ENABLE_SERVER_DEBUG_LOGS = typeof __DEV__ !== 'undefined' && __DEV__;

interface BridgeProfile {
  id: string;
  name: string;
  serverEndpoint: string;
  authToken: string | null;
}

interface MobilePreferences {
  prefersDarkMode: boolean;
  activeBridgeId: string;
  bridges: BridgeProfile[];
}

interface StoredBridgeProfile {
  id?: unknown;
  name?: unknown;
  serverEndpoint?: unknown;
  authToken?: unknown;
}

interface StoredMobilePreferences {
  activeBridgeId?: unknown;
  bridges?: unknown;
  endpoint?: unknown;
  serverEndpoint?: unknown;
  bridgeEndpoint?: unknown;
  uiEndpoint?: unknown;
  appServerEndpoint?: unknown;
  prefersDarkMode?: unknown;
  authToken?: unknown;
}

function generateBridgeId() {
  return `bridge_${Math.random().toString(36).slice(2, 10)}_${Date.now().toString(36)}`;
}

function deriveBridgeName(endpoint: string, fallback = 'Bridge') {
  const normalized = normalizeEndpoint(endpoint);
  if (!normalized) {
    return fallback;
  }
  try {
    const url = new URL(deriveServerHttpBaseUrl(normalized));
    return url.host || fallback;
  } catch {
    return normalized;
  }
}

function createBridgeProfile(
  input: Partial<BridgeProfile> & Pick<BridgeProfile, 'serverEndpoint'>,
): BridgeProfile {
  const serverEndpoint = normalizeEndpoint(input.serverEndpoint);
  const name =
    typeof input.name === 'string' && input.name.trim().length > 0
      ? input.name.trim()
      : deriveBridgeName(serverEndpoint);
  const authToken =
    typeof input.authToken === 'string' && input.authToken.trim().length > 0
      ? input.authToken.trim()
      : null;

  return {
    id:
      typeof input.id === 'string' && input.id.trim().length > 0
        ? input.id.trim()
        : generateBridgeId(),
    name,
    serverEndpoint,
    authToken,
  };
}

function createDefaultBridgeProfile(): BridgeProfile {
  return createBridgeProfile({
    id: 'bridge_local',
    name: 'Local',
    serverEndpoint: LOCAL_SERVER_DEBUG_ENDPOINT,
    authToken: null,
  });
}

function ensureBridgeProfiles(
  input: unknown,
  fallbackEndpoint: string | null,
  fallbackAuthToken: string | null,
): BridgeProfile[] {
  const profiles: BridgeProfile[] = [];
  const seenIds = new Set<string>();

  if (Array.isArray(input)) {
    for (const value of input) {
      const profile = value as StoredBridgeProfile | null;
      if (!profile || typeof profile !== 'object') {
        continue;
      }
      const endpoint =
        typeof profile.serverEndpoint === 'string' && profile.serverEndpoint.trim().length > 0
          ? normalizeEndpoint(profile.serverEndpoint)
          : '';
      if (!endpoint) {
        continue;
      }

      const nextProfile = createBridgeProfile({
        id: typeof profile.id === 'string' ? profile.id : undefined,
        name: typeof profile.name === 'string' ? profile.name : undefined,
        serverEndpoint: endpoint,
        authToken: typeof profile.authToken === 'string' ? profile.authToken : null,
      });
      if (seenIds.has(nextProfile.id)) {
        nextProfile.id = generateBridgeId();
      }
      seenIds.add(nextProfile.id);
      profiles.push(nextProfile);
    }
  }

  if (profiles.length > 0) {
    return profiles;
  }

  if (fallbackEndpoint) {
    return [
      createBridgeProfile({
        id: 'bridge_primary',
        serverEndpoint: fallbackEndpoint,
        authToken: fallbackAuthToken,
      }),
    ];
  }

  return [createDefaultBridgeProfile()];
}

function deriveLegacyEndpoint(parsed: StoredMobilePreferences): string | null {
  const legacyEndpoint =
    typeof parsed.endpoint === 'string' && parsed.endpoint.trim().length > 0
      ? parsed.endpoint.trim()
      : null;

  return typeof parsed.serverEndpoint === 'string' && parsed.serverEndpoint.trim().length > 0
    ? parsed.serverEndpoint.trim()
    : typeof parsed.bridgeEndpoint === 'string' && parsed.bridgeEndpoint.trim().length > 0
      ? parsed.bridgeEndpoint.trim()
      : typeof parsed.uiEndpoint === 'string' && parsed.uiEndpoint.trim().length > 0
        ? parsed.uiEndpoint.trim()
        : typeof parsed.appServerEndpoint === 'string' && parsed.appServerEndpoint.trim().length > 0
          ? parsed.appServerEndpoint.trim()
          : legacyEndpoint;
}

function migrateStoredPreferences(parsed: StoredMobilePreferences): MobilePreferences {
  const fallbackEndpoint = deriveLegacyEndpoint(parsed);
  const fallbackAuthToken =
    typeof parsed.authToken === 'string' && parsed.authToken.trim().length > 0
      ? parsed.authToken.trim()
      : null;
  const bridges = ensureBridgeProfiles(parsed.bridges, fallbackEndpoint, fallbackAuthToken);
  const activeBridgeId =
    typeof parsed.activeBridgeId === 'string' &&
    bridges.some((bridge) => bridge.id === parsed.activeBridgeId)
      ? parsed.activeBridgeId
      : bridges[0].id;

  return {
    prefersDarkMode:
      typeof parsed.prefersDarkMode === 'boolean' ? parsed.prefersDarkMode : true,
    activeBridgeId,
    bridges,
  };
}

function resolveActiveBridge(preferences: MobilePreferences): BridgeProfile {
  return (
    preferences.bridges.find((bridge) => bridge.id === preferences.activeBridgeId) ??
    preferences.bridges[0] ??
    createDefaultBridgeProfile()
  );
}

const DEFAULT_PREFERENCES: MobilePreferences = {
  prefersDarkMode: true,
  activeBridgeId: 'bridge_local',
  bridges: [createDefaultBridgeProfile()],
};

interface ConnectionTargetResponse {
  recommendedServerEndpoint: string;
  authMode: 'none' | 'device-token';
  localAuthPage: string | null;
}

interface BridgeAuthorizationResponse {
  authorized: boolean;
  authMode: 'none' | 'device-token';
  localAuthPage: string | null;
  reason: string | null;
  deviceName: string | null;
}

const LOCAL_HOST_ID = 'local';
const DESKTOP_EXTENSION_INFO = {
  version: '26.323.20928',
  buildFlavor: 'prod',
  buildNumber: '1173',
};
const LOCAL_HOST_CONFIG = {
  id: LOCAL_HOST_ID,
  hostId: LOCAL_HOST_ID,
  display_name: 'Local',
  displayName: 'Local',
  kind: 'local',
};
const UNHANDLED_LOCAL_METHOD = Symbol('unhandled-local-method');
const APP_SERVER_REQUEST_LOG_THRESHOLD_MS = 250;
const DEFAULT_THREAD_LIST_PARAMS = {
  limit: 50,
  cursor: null,
  sortKey: 'updated_at',
  modelProviders: null,
  archived: false,
  sourceKinds: [],
} as const;
const DEFAULT_APP_LIST_PARAMS = {
  cursor: null,
  limit: 1000,
  forceRefetch: true,
} as const;
const DEFAULT_MCP_SERVER_STATUS_LIST_PARAMS = {
  cursor: null,
  limit: 100,
} as const;
const DEFAULT_MODEL_LIST_PARAMS = {
  includeHidden: true,
  cursor: null,
  limit: 100,
} as const;
const DEFAULT_PERSISTED_ATOM_STATE: Record<string, unknown> = {
  statsig_default_enable_features: {
    fast_mode: true,
  },
};

interface NativeEnvelope {
  __codexMobile?: boolean;
  kind?: string;
  payload?: unknown;
}

type BridgeMenuKind = 'application' | 'context';

interface BridgeMenuItem {
  id?: string;
  type?: string;
  label: string;
  enabled: boolean;
  submenu?: BridgeMenuItem[];
}

interface BridgeMenuLevel {
  title: string | null;
  items: BridgeMenuItem[];
}

interface ActiveBridgeMenu {
  kind: BridgeMenuKind;
  menuId: string | null;
  requestId: string;
  stack: BridgeMenuLevel[];
}

interface PendingServerRequestResolver {
  resolve: (value: unknown) => void;
  reject: (error: Error) => void;
  method: string;
}

interface TerminalSnapshot {
  cwd: string;
  shell: string;
  buffer: string;
  truncated: boolean;
}

interface HostStateSnapshot {
  account: unknown | null;
  rateLimit: unknown | null;
  workspaceRootOptions: string[];
  activeWorkspaceRoots: string[];
  workspaceRootLabels: Record<string, string>;
  pinnedThreadIds: string[];
}

interface CachedAppServerResult {
  expiresAt: number;
  result: unknown;
}

function shouldTraceAppServerMethod(method: string) {
  return (
    method === 'turn/start' ||
    method === 'turn/interrupt' ||
    method === 'thread/list' ||
    method === 'thread/read' ||
    method === 'thread/resume' ||
    method === 'config/read' ||
    method === 'app/list' ||
    method === 'mcpServerStatus/list' ||
    method === 'model/list'
  );
}

function shouldTraceAppServerNotification(method: string) {
  return (
    method === 'thread/started' ||
    method === 'thread/status/changed' ||
    method === 'turn/started' ||
    method === 'turn/completed' ||
    method === 'item/started' ||
    method === 'item/completed' ||
    method === 'item/agentMessage/delta' ||
    method === 'item/plan/delta' ||
    method === 'item/reasoning/summaryTextDelta' ||
    method === 'item/reasoning/textDelta' ||
    method === 'item/commandExecution/outputDelta' ||
    method === 'serverRequest/resolved' ||
    method === 'error'
  );
}

function summarizeAppServerNotification(notification: AppServerServerNotification) {
  const params = asObject(notification.params);
  return {
    method: notification.method,
    threadId: typeof params?.threadId === 'string' ? params.threadId : null,
    turnId: typeof params?.turnId === 'string' ? params.turnId : null,
    itemId: typeof params?.itemId === 'string' ? params.itemId : null,
    itemType:
      params && typeof params.item === 'object' && params.item && 'type' in params.item
        ? typeof params.item.type === 'string'
          ? params.item.type
          : null
        : null,
    deltaLength: typeof params?.delta === 'string' ? params.delta.length : null,
  };
}

function summarizeAppServerResult(method: string, result: unknown) {
  const payload = asObject(result);
  const thread = asObject(payload?.thread);
  const turn = asObject(payload?.turn);
  const lastThreadTurn =
    Array.isArray(thread?.turns) && thread.turns.length > 0
      ? asObject(thread.turns.at(-1))
      : null;
  const items = Array.isArray(turn?.items)
    ? turn.items
    : Array.isArray(lastThreadTurn?.items)
      ? lastThreadTurn.items
      : [];

  return {
    method,
    threadId:
      typeof thread?.id === 'string'
        ? thread.id
        : typeof payload?.threadId === 'string'
          ? payload.threadId
          : null,
    turnId:
      typeof turn?.id === 'string'
        ? turn.id
        : typeof payload?.turnId === 'string'
          ? payload.turnId
          : typeof lastThreadTurn?.id === 'string'
            ? lastThreadTurn.id
            : null,
    turnStatus:
      typeof turn?.status === 'string'
        ? turn.status
        : typeof payload?.status === 'string'
          ? payload.status
          : typeof lastThreadTurn?.status === 'string'
            ? lastThreadTurn.status
            : null,
    itemCount: items.length,
    itemTypes: items
      .map((item) => {
        const nextItem = asObject(item);
        return typeof nextItem?.type === 'string' ? nextItem.type : null;
      })
      .filter((itemType): itemType is string => itemType !== null)
      .slice(0, 8),
  };
}

function summarizeHostBridgeMessage(message: Record<string, unknown>) {
  const response = asObject(message.response);
  const request = asObject(message.request);
  const notification = asObject(message.notification);
  const result = asObject(message.result);

  return {
    type: typeof message.type === 'string' ? message.type : null,
    id:
      normalizeRequestId(message.id) ??
      normalizeRequestId(response?.id) ??
      normalizeRequestId(request?.id),
    method:
      typeof message.method === 'string'
        ? message.method
        : typeof request?.method === 'string'
          ? request.method
          : typeof notification?.method === 'string'
            ? notification.method
            : null,
    hostId: typeof message.hostId === 'string' ? message.hostId : null,
    hasError: asObject(message.error) != null || asObject(response?.error) != null,
    result:
      typeof message.method === 'string'
        ? summarizeAppServerResult(message.method, result ?? message.result)
        : null,
  };
}

function summarizeAppServerParams(params: unknown) {
  try {
    return JSON.stringify(params ?? null);
  } catch {
    return '[unserializable-params]';
  }
}

function normalizeAppServerRequestParams(method: string, params: unknown) {
  if (method === 'thread/list') {
    const payload = asObject(params);
    return {
      limit: typeof payload?.limit === 'number' ? payload.limit : DEFAULT_THREAD_LIST_PARAMS.limit,
      cursor:
        typeof payload?.cursor === 'string' || payload?.cursor === null
          ? payload.cursor
          : DEFAULT_THREAD_LIST_PARAMS.cursor,
      sortKey:
        typeof payload?.sortKey === 'string'
          ? payload.sortKey
          : DEFAULT_THREAD_LIST_PARAMS.sortKey,
      modelProviders: Array.isArray(payload?.modelProviders)
        ? payload.modelProviders
        : DEFAULT_THREAD_LIST_PARAMS.modelProviders,
      archived:
        typeof payload?.archived === 'boolean'
          ? payload.archived
          : DEFAULT_THREAD_LIST_PARAMS.archived,
      sourceKinds: Array.isArray(payload?.sourceKinds)
        ? payload.sourceKinds
        : DEFAULT_THREAD_LIST_PARAMS.sourceKinds,
    };
  }

  if (method === 'app/list') {
    const payload = asObject(params);
    return {
      cursor:
        typeof payload?.cursor === 'string' || payload?.cursor === null
          ? payload.cursor
          : DEFAULT_APP_LIST_PARAMS.cursor,
      limit: typeof payload?.limit === 'number' ? payload.limit : DEFAULT_APP_LIST_PARAMS.limit,
      forceRefetch:
        typeof payload?.forceRefetch === 'boolean'
          ? payload.forceRefetch
          : DEFAULT_APP_LIST_PARAMS.forceRefetch,
    };
  }

  if (method === 'mcpServerStatus/list') {
    const payload = asObject(params);
    return {
      cursor:
        typeof payload?.cursor === 'string' || payload?.cursor === null
          ? payload.cursor
          : DEFAULT_MCP_SERVER_STATUS_LIST_PARAMS.cursor,
      limit:
        typeof payload?.limit === 'number'
          ? payload.limit
          : DEFAULT_MCP_SERVER_STATUS_LIST_PARAMS.limit,
    };
  }

  if (method === 'model/list') {
    const payload = asObject(params);
    return {
      includeHidden:
        typeof payload?.includeHidden === 'boolean'
          ? payload.includeHidden
          : DEFAULT_MODEL_LIST_PARAMS.includeHidden,
      cursor:
        typeof payload?.cursor === 'string' || payload?.cursor === null
          ? payload.cursor
          : DEFAULT_MODEL_LIST_PARAMS.cursor,
      limit:
        typeof payload?.limit === 'number' ? payload.limit : DEFAULT_MODEL_LIST_PARAMS.limit,
    };
  }

  if (method === 'config/read') {
    const payload = asObject(params);
    return {
      includeLayers:
        typeof payload?.includeLayers === 'boolean' ? payload.includeLayers : undefined,
      cwd:
        typeof payload?.cwd === 'string' || payload?.cwd === null
          ? payload.cwd
          : undefined,
    };
  }

  return params ?? null;
}

function normalizeEndpoint(value: string): string {
  return value.trim().replace(/\/+$/, '');
}

function deriveServerHttpBaseUrl(endpoint: string): string {
  const normalized = normalizeEndpoint(endpoint);
  if (normalized.startsWith('ws://')) {
    return `http://${normalized.slice('ws://'.length)}`;
  }
  if (normalized.startsWith('wss://')) {
    return `https://${normalized.slice('wss://'.length)}`;
  }
  if (normalized.startsWith('http://') || normalized.startsWith('https://')) {
    return normalized;
  }
  return `http://${normalized}`;
}

function buildAuthHeaders(authToken: string | null | undefined): Record<string, string> {
  const token = typeof authToken === 'string' ? authToken.trim() : '';
  if (!token) {
    return {};
  }
  return {
    Authorization: `Bearer ${token}`,
  };
}

async function fetchConnectionTarget(
  endpoint: string,
  authToken: string | null | undefined,
): Promise<ConnectionTargetResponse> {
  const response = await fetch(`${deriveServerHttpBaseUrl(endpoint)}/codex-mobile/connect`, {
    headers: buildAuthHeaders(authToken),
  });
  if (!response.ok) {
    throw new Error(`Connection discovery failed with HTTP ${response.status}.`);
  }

  const payload = (await response.json()) as {
    connection?: {
      recommendedServerEndpoint?: unknown;
    };
    auth?: {
      mode?: unknown;
    };
    localAuthPage?: unknown;
  };

  const recommendedServerEndpoint =
    typeof payload.connection?.recommendedServerEndpoint === 'string' &&
    payload.connection.recommendedServerEndpoint.trim().length > 0
      ? normalizeEndpoint(payload.connection.recommendedServerEndpoint)
      : normalizeEndpoint(endpoint);

  const authMode = payload.auth?.mode === 'device-token' ? 'device-token' : 'none';
  const localAuthPage =
    typeof payload.localAuthPage === 'string' && payload.localAuthPage.trim().length > 0
      ? payload.localAuthPage.trim()
      : null;

  return {
    recommendedServerEndpoint,
    authMode,
    localAuthPage,
  };
}

async function completeDevicePairing(
  endpoint: string,
  pairingCode: string,
  authToken: string | null | undefined,
): Promise<{ accessToken: string; approved: boolean }> {
  const response = await fetch(`${deriveServerHttpBaseUrl(endpoint)}/auth/pair/complete`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...buildAuthHeaders(authToken),
    },
    body: JSON.stringify({
      code: pairingCode.trim(),
      deviceName: `Codex Mobile (${Platform.OS})`,
    }),
  });
  const payload = (await response.json().catch(() => null)) as
    | {
        accessToken?: unknown;
        approved?: unknown;
        error?: unknown;
      }
    | null;

  if (!response.ok) {
    const errorMessage =
      typeof payload?.error === 'string' && payload.error.trim().length > 0
        ? payload.error
        : `Pairing failed with HTTP ${response.status}.`;
    throw new Error(errorMessage);
  }

  if (typeof payload?.accessToken !== 'string' || payload.accessToken.trim().length === 0) {
    throw new Error('Pairing response did not include an access token.');
  }

  return {
    accessToken: payload.accessToken.trim(),
    approved: payload?.approved === true,
  };
}

async function probeBridgeAuthorization(
  endpoint: string,
  authToken: string | null | undefined,
): Promise<BridgeAuthorizationResponse> {
  const response = await fetch(`${deriveServerHttpBaseUrl(endpoint)}/auth/session`, {
    headers: buildAuthHeaders(authToken),
  });
  const payload = (await response.json().catch(() => null)) as
    | {
        authorized?: unknown;
        auth?: {
          mode?: unknown;
        };
        localAuthPage?: unknown;
        reason?: unknown;
        deviceName?: unknown;
        error?: unknown;
      }
    | null;

  const authMode = payload?.auth?.mode === 'device-token' ? 'device-token' : 'none';
  const localAuthPage =
    typeof payload?.localAuthPage === 'string' && payload.localAuthPage.trim().length > 0
      ? payload.localAuthPage.trim()
      : null;
  const reason =
    typeof payload?.reason === 'string' && payload.reason.trim().length > 0
      ? payload.reason.trim()
      : null;
  const deviceName =
    typeof payload?.deviceName === 'string' && payload.deviceName.trim().length > 0
      ? payload.deviceName.trim()
      : null;

  if (response.status === 401) {
    return {
      authorized: false,
      authMode,
      localAuthPage,
      reason,
      deviceName,
    };
  }
  if (!response.ok) {
    const errorMessage =
      typeof payload?.error === 'string' && payload.error.trim().length > 0
        ? payload.error
        : `Authorization probe failed with HTTP ${response.status}.`;
    throw new Error(errorMessage);
  }

  return {
    authorized: payload?.authorized !== false,
    authMode,
    localAuthPage,
    reason,
    deviceName,
  };
}

function buildDeviceTokenHelpMessage(
  localAuthPage: string | null,
  reason: string,
): string {
  if (localAuthPage) {
    return `${reason} On the bridge host, open ${localAuthPage} to generate a pairing code and approve devices.`;
  }
  return `${reason} Open the bridge host locally and generate a pairing code before reconnecting.`;
}

function buildUnauthorizedReasonMessage(
  probe: BridgeAuthorizationResponse,
  fallbackReason: string,
): string {
  const deviceName = probe.deviceName ? ` for ${probe.deviceName}` : '';
  switch (probe.reason) {
    case 'pending_approval':
      return buildDeviceTokenHelpMessage(
        probe.localAuthPage,
        `This device${deviceName} is paired but still waiting for approval.`,
      );
    case 'revoked':
      return buildDeviceTokenHelpMessage(
        probe.localAuthPage,
        `This device token${deviceName} has been revoked. Re-pair this device to continue.`,
      );
    case 'unknown_token':
      return buildDeviceTokenHelpMessage(
        probe.localAuthPage,
        `This device token is no longer recognized by the bridge.`,
      );
    case 'missing_token':
      return buildDeviceTokenHelpMessage(
        probe.localAuthPage,
        `This bridge requires device pairing before it will accept requests.`,
      );
    default:
      return buildDeviceTokenHelpMessage(probe.localAuthPage, fallbackReason);
  }
}

function buildRemoteShellUrl(endpoint: string, themeVariant: 'light' | 'dark'): string {
  const baseUrl = deriveServerHttpBaseUrl(endpoint);
  return `${baseUrl}/ui/index.html?codexTheme=${encodeURIComponent(themeVariant)}`;
}

function resolveServerFetchUrl(rawUrl: string, endpoint: string): string | null {
  const trimmed = rawUrl.trim();
  if (trimmed.length === 0) {
    return null;
  }
  if (trimmed.startsWith('http://') || trimmed.startsWith('https://')) {
    return trimmed;
  }
  try {
    return new URL(trimmed, `${deriveServerHttpBaseUrl(endpoint)}/`).toString();
  } catch {
    return null;
  }
}

function asObject(value: unknown): Record<string, unknown> | null {
  return value != null && typeof value === 'object' ? (value as Record<string, unknown>) : null;
}

function normalizeBridgeMenuItems(value: unknown): BridgeMenuItem[] {
  if (!Array.isArray(value)) {
    return [];
  }

  const nextItems: BridgeMenuItem[] = [];
  for (const entry of value) {
    const item = asObject(entry);
    if (!item) {
      continue;
    }

    const type = typeof item.type === 'string' ? item.type : 'normal';
    const submenu = normalizeBridgeMenuItems(item.submenu);
    const label = typeof item.label === 'string' ? item.label : '';
    if (type !== 'separator' && !label && submenu.length === 0) {
      continue;
    }

    nextItems.push({
      id: typeof item.id === 'string' ? item.id : undefined,
      type,
      label,
      enabled: item.enabled !== false,
      submenu: submenu.length > 0 ? submenu : undefined,
    });
  }

  return nextItems;
}

function normalizeErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message.trim().length > 0) {
    return error.message;
  }
  if (typeof error === 'string' && error.trim().length > 0) {
    return error;
  }
  try {
    const serialized = JSON.stringify(error);
    return serialized && serialized !== 'null' ? serialized : 'Unknown error.';
  } catch {
    return 'Unknown error.';
  }
}

function normalizeRequestId(value: unknown): AppServerRequestId | null {
  if (typeof value === 'string' || typeof value === 'number') {
    return value;
  }
  return null;
}

function parseJsonBody(value: unknown): unknown {
  if (typeof value !== 'string' || value.trim().length === 0) {
    return undefined;
  }
  try {
    return JSON.parse(value);
  } catch {
    return undefined;
  }
}

function normalizePathString(value: string): string {
  const normalized = value.replace(/\\/g, '/').replace(/\/+/g, '/');
  if (normalized.length > 1 && normalized.endsWith('/')) {
    return normalized.slice(0, -1);
  }
  return normalized;
}

function uniqueStringValues(values: unknown[]): string[] {
  const nextValues: string[] = [];
  const seen = new Set<string>();
  for (const value of values) {
    if (typeof value !== 'string') {
      continue;
    }
    const normalized = value.trim();
    if (!normalized || seen.has(normalized)) {
      continue;
    }
    seen.add(normalized);
    nextValues.push(normalized);
  }
  return nextValues;
}

function shallowStringArrayEquals(left: string[], right: string[]) {
  if (left.length !== right.length) {
    return false;
  }
  for (let index = 0; index < left.length; index += 1) {
    if (left[index] !== right[index]) {
      return false;
    }
  }
  return true;
}

function isSessionArtifactPath(value: string, codexHome: string) {
  const normalizedPath = normalizePathString(value);
  const normalizedCodexHome = normalizePathString(codexHome);
  return (
    normalizedPath.endsWith('.jsonl') ||
    normalizedPath.startsWith(`${normalizedCodexHome}/sessions/`)
  );
}

function resolveRequestMethodName(rawUrl: string): string | null {
  try {
    const url = new URL(rawUrl);
    if (url.protocol !== 'vscode:' || url.hostname !== 'codex') {
      return null;
    }
    return url.pathname.replace(/^\/+/, '') || null;
  } catch {
    return null;
  }
}

function buildFetchSuccessResponse(
  requestId: string,
  body: unknown,
  status = 200,
  headers: Record<string, string> = {},
) {
  return {
    type: 'fetch-response',
    requestId,
    responseType: 'success',
    status,
    headers,
    bodyJsonString: JSON.stringify(body ?? null),
  };
}

function buildFetchErrorResponse(
  requestId: string,
  error: string,
  status = 500,
) {
  return {
    type: 'fetch-response',
    requestId,
    responseType: 'error',
    status,
    error,
  };
}

export default function App() {
  const webViewRef = useRef<WebView>(null);
  const systemTheme = useColorScheme();
  const { width: viewportWidth } = useWindowDimensions();
  const persistedAtomStateRef = useRef<Record<string, unknown>>(DEFAULT_PERSISTED_ATOM_STATE);
  const globalStateRef = useRef<Record<string, unknown>>({});
  const configurationStateRef = useRef<Record<string, unknown>>({});
  const terminalSnapshotsRef = useRef<Record<string, TerminalSnapshot>>({});
  const sharedObjectsRef = useRef<Record<string, unknown>>({
    pending_worktrees: [],
    remote_connections: [],
    host_config: LOCAL_HOST_CONFIG,
  });
  const sharedObjectSubscribersRef = useRef(new Map<string, number>());
  const queuedHostMessagesRef = useRef<Record<string, unknown>[]>([]);
  const queuedHostMessagesFlushTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const appServerClientRef = useRef<CodexAppServerClient | null>(null);
  const appServerConnectPromiseRef = useRef<Promise<CodexAppServerClient> | null>(null);
  const appServerConnectionVersionRef = useRef(0);
  const appServerPendingRequestsRef = useRef(new Map<string, Promise<unknown>>());
  const appServerResultCacheRef = useRef(new Map<string, CachedAppServerResult>());
  const pendingTurnCompletionRef = useRef(new Map<string, string>());
  const pendingTurnReconciliationsRef = useRef(new Map<string, Promise<void>>());
  const lastAuthPromptAtRef = useRef(0);
  const pendingServerRequestResolversRef = useRef(
    new Map<AppServerRequestId, PendingServerRequestResolver>(),
  );
  const hostStateRef = useRef<HostStateSnapshot>({
    account: null,
    rateLimit: null,
    workspaceRootOptions: [],
    activeWorkspaceRoots: [],
    workspaceRootLabels: {},
    pinnedThreadIds: [],
  });

  const [preferences, setPreferences] = useState<MobilePreferences>(DEFAULT_PREFERENCES);
  const activeBridge = resolveActiveBridge(preferences);
  const [resolvedServerEndpoint, setResolvedServerEndpoint] = useState(
    activeBridge.serverEndpoint,
  );
  const [resolvedAuthMode, setResolvedAuthMode] = useState<'none' | 'device-token'>('none');
  const [resolvedLocalAuthPage, setResolvedLocalAuthPage] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [editingBridgeId, setEditingBridgeId] = useState(activeBridge.id);
  const [bridgeNameDraft, setBridgeNameDraft] = useState(activeBridge.name);
  const [serverEndpointDraft, setServerEndpointDraft] = useState(activeBridge.serverEndpoint);
  const [pairingCodeDraft, setPairingCodeDraft] = useState('');
  const [settingsBusy, setSettingsBusy] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [activeBridgeMenu, setActiveBridgeMenu] = useState<ActiveBridgeMenu | null>(null);
  const [preferencesHydrated, setPreferencesHydrated] = useState(false);

  const editingBridge =
    preferences.bridges.find((bridge) => bridge.id === editingBridgeId) ?? null;
  const activeAuthToken = activeBridge.authToken;

  useEffect(() => {
    void (async () => {
      try {
        let stored = await AsyncStorage.getItem(STORAGE_KEY);
        if (!stored) {
          for (const legacyKey of LEGACY_STORAGE_KEYS) {
            stored = await AsyncStorage.getItem(legacyKey);
            if (stored) {
              break;
            }
          }
        }
        if (!stored) {
          setPreferencesHydrated(true);
          return;
        }
        const parsed = JSON.parse(stored) as StoredMobilePreferences;
        const nextPreferences = migrateStoredPreferences(parsed);
        const nextActiveBridge = resolveActiveBridge(nextPreferences);
        setPreferences(nextPreferences);
        setResolvedServerEndpoint(nextActiveBridge.serverEndpoint);
        setEditingBridgeId(nextActiveBridge.id);
        setBridgeNameDraft(nextActiveBridge.name);
        setServerEndpointDraft(nextActiveBridge.serverEndpoint);
        await Promise.all(
          LEGACY_STORAGE_KEYS.map((legacyKey) => AsyncStorage.removeItem(legacyKey)),
        );
      } catch (error) {
        console.warn('Failed to load mobile preferences', error);
      } finally {
        setPreferencesHydrated(true);
      }
    })();
  }, []);

  useEffect(() => {
    if (!preferencesHydrated) {
      return;
    }
    void AsyncStorage.setItem(STORAGE_KEY, JSON.stringify(preferences)).catch((error) => {
      console.warn('Failed to persist mobile preferences', error);
    });
  }, [preferences, preferencesHydrated]);

  useEffect(() => {
    let cancelled = false;

    setResolvedServerEndpoint(activeBridge.serverEndpoint);
    setResolvedAuthMode('none');
    setResolvedLocalAuthPage(null);

    void fetchConnectionTarget(activeBridge.serverEndpoint, activeAuthToken)
      .then((connection) => {
        if (cancelled) {
          return;
        }
        setResolvedServerEndpoint(connection.recommendedServerEndpoint);
        setResolvedAuthMode(connection.authMode);
        setResolvedLocalAuthPage(connection.localAuthPage);
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        console.warn('Failed to resolve server connection target', error);
      });

    return () => {
      cancelled = true;
    };
  }, [activeAuthToken, activeBridge.serverEndpoint]);

  useEffect(() => {
    let snapshotTimeout: ReturnType<typeof setTimeout> | null = null;
    let appListWarmTimeout: ReturnType<typeof setTimeout> | null = null;
    let mcpStatusWarmTimeout: ReturnType<typeof setTimeout> | null = null;
    let modelListWarmTimeout: ReturnType<typeof setTimeout> | null = null;

    appServerConnectionVersionRef.current += 1;
    appServerClientRef.current?.disconnect();
    appServerClientRef.current = null;
    appServerConnectPromiseRef.current = null;
    appServerPendingRequestsRef.current.clear();
    appServerResultCacheRef.current.clear();
    pendingServerRequestResolversRef.current.clear();

    void ensureAppServerClient()
      .then(async (client) => {
        const result = await sendAppServerRequest('thread/list', DEFAULT_THREAD_LIST_PARAMS);
        appListWarmTimeout = setTimeout(() => {
          void sendAppServerRequest('app/list', DEFAULT_APP_LIST_PARAMS).catch((error) => {
            console.warn('Failed to warm app list', error);
          });
        }, 250);
        mcpStatusWarmTimeout = setTimeout(() => {
          void sendAppServerRequest(
            'mcpServerStatus/list',
            DEFAULT_MCP_SERVER_STATUS_LIST_PARAMS,
          ).catch((error) => {
            console.warn('Failed to warm MCP server status list', error);
          });
        }, 500);
        modelListWarmTimeout = setTimeout(() => {
          void sendAppServerRequest('model/list', DEFAULT_MODEL_LIST_PARAMS).catch((error) => {
            console.warn('Failed to warm model list', error);
          });
        }, 750);
        snapshotTimeout = setTimeout(() => {
          refreshAppServerSnapshots(client, appServerConnectionVersionRef.current);
        }, 2_500);
        return result;
      })
      .then((result) => {
        integrateAppServerResult('thread/list', result);
      })
      .catch((error) => {
        console.warn('Failed to warm app-server', error);
      });

    return () => {
      appServerConnectionVersionRef.current += 1;
      appServerClientRef.current?.disconnect();
      appServerClientRef.current = null;
      appServerConnectPromiseRef.current = null;
      appServerPendingRequestsRef.current.clear();
      appServerResultCacheRef.current.clear();
      pendingServerRequestResolversRef.current.clear();
      if (snapshotTimeout) {
        clearTimeout(snapshotTimeout);
      }
      if (appListWarmTimeout) {
        clearTimeout(appListWarmTimeout);
      }
      if (mcpStatusWarmTimeout) {
        clearTimeout(mcpStatusWarmTimeout);
      }
      if (modelListWarmTimeout) {
        clearTimeout(modelListWarmTimeout);
      }
    };
  }, [activeAuthToken, resolvedServerEndpoint]);

  const themeVariant: 'light' | 'dark' =
    preferences.prefersDarkMode || systemTheme !== 'light' ? 'dark' : 'light';
  const viewportScale =
    viewportWidth < 900
      ? Math.max(Math.min(viewportWidth / 760, 0.56), 0.46)
      : 1;
  const remoteShellUrl = useMemo(
    () => buildRemoteShellUrl(resolvedServerEndpoint, themeVariant),
    [resolvedServerEndpoint, themeVariant],
  );

  useEffect(() => {
    syncThemeToRenderer(themeVariant);
  }, [themeVariant]);

  useEffect(() => {
    syncViewportToRenderer(viewportScale);
  }, [viewportScale]);

  async function handleUnauthorizedBridgeAccess(
    reason: string,
    options: { showBanner?: boolean } = {},
  ) {
    try {
      const probe = await probeBridgeAuthorization(
        activeBridge.serverEndpoint,
        activeAuthToken,
      );
      if (probe.authorized || probe.authMode !== 'device-token') {
        return false;
      }

      const message = buildUnauthorizedReasonMessage(probe, reason);
      setResolvedAuthMode(probe.authMode);
      setResolvedLocalAuthPage(probe.localAuthPage);
      setEditingBridgeId(activeBridge.id);
      setBridgeNameDraft(activeBridge.name);
      setServerEndpointDraft(activeBridge.serverEndpoint);
      setSettingsOpen(true);
      if (options.showBanner) {
        setLoadError(message);
      }

      const now = Date.now();
      if (now - lastAuthPromptAtRef.current > 5000) {
        lastAuthPromptAtRef.current = now;
        Alert.alert('Authentication required', message);
      }
      return true;
    } catch (error) {
      console.warn('Failed to probe bridge authorization state', error);
      return false;
    }
  }

  function injectHostMessages(messages: Record<string, unknown>[]) {
    if (messages.length === 0) {
      return;
    }

    webViewRef.current?.injectJavaScript(
      [
        '(function () {',
        '  var host = window.__codexMobileHost;',
        '  if (host && typeof host.dispatchHostMessage === "function") {',
        `    var messages = ${JSON.stringify(messages)};`,
        '    for (var i = 0; i < messages.length; i += 1) {',
        '      host.dispatchHostMessage(messages[i]);',
        '    }',
        '  }',
        '})();',
        'true;',
      ].join('\n'),
    );
  }

  function flushQueuedHostMessages() {
    if (queuedHostMessagesFlushTimeoutRef.current) {
      clearTimeout(queuedHostMessagesFlushTimeoutRef.current);
      queuedHostMessagesFlushTimeoutRef.current = null;
    }

    if (queuedHostMessagesRef.current.length === 0) {
      return;
    }

    const messages = queuedHostMessagesRef.current.splice(0);
    injectHostMessages(messages);
  }

  function queueHostMessage(message: Record<string, unknown>) {
    queuedHostMessagesRef.current.push(message);
    if (queuedHostMessagesRef.current.length >= 24) {
      flushQueuedHostMessages();
      return;
    }
    if (queuedHostMessagesFlushTimeoutRef.current) {
      return;
    }
    queuedHostMessagesFlushTimeoutRef.current = setTimeout(() => {
      queuedHostMessagesFlushTimeoutRef.current = null;
      flushQueuedHostMessages();
    }, 16);
  }

  function sendHostMessage(message: Record<string, unknown>) {
    const messageType = typeof message.type === 'string' ? message.type : null;
    if (
      messageType === 'mcp-notification' ||
      messageType === 'mcp-request' ||
      messageType === 'mcp-response'
    ) {
      if (ENABLE_SERVER_DEBUG_LOGS) {
        console.log('[host->renderer]', summarizeHostBridgeMessage(message));
      }
    }

    if (messageType === 'mcp-notification') {
      queueHostMessage(message);
      return;
    }

    flushQueuedHostMessages();
    injectHostMessages([message]);
  }

  function sendWorkerMessage(workerId: string, payload: Record<string, unknown>) {
    webViewRef.current?.injectJavaScript(
      [
        '(function () {',
        '  var host = window.__codexMobileHost;',
        '  if (host && typeof host.dispatchWorkerMessage === "function") {',
        `    host.dispatchWorkerMessage(${JSON.stringify(workerId)}, ${JSON.stringify(payload)});`,
        '  }',
        '})();',
        'true;',
      ].join('\n'),
    );
  }

  function syncThemeToRenderer(nextTheme: 'light' | 'dark') {
    webViewRef.current?.injectJavaScript(
      [
        '(function () {',
        '  window.codexWindowType = "electron";',
        '  try {',
        '    if (document && document.documentElement) {',
        '      document.documentElement.setAttribute("data-codex-window-type", "electron");',
        '    }',
        '  } catch (error) {}',
        '  var host = window.__codexMobileHost;',
        '  if (host && typeof host.updateTheme === "function") {',
        `    host.updateTheme(${JSON.stringify(nextTheme)});`,
        '  }',
        '})();',
        'true;',
      ].join('\n'),
    );
  }

  function syncViewportToRenderer(nextScale: number) {
    const normalizedScale = Math.max(Math.min(nextScale, 1), 0.28);
    const widthPercent = `${(100 / normalizedScale).toFixed(3)}%`;
    const minHeight = `${(100 / normalizedScale).toFixed(3)}vh`;
    webViewRef.current?.injectJavaScript(
      [
        '(function () {',
        '  var doc = document;',
        '  if (!doc || !doc.head || !doc.body) { return; }',
        '  var root = doc.getElementById("root");',
        '  if (!root) { return; }',
        '  var styleId = "codex-mobile-viewport-style";',
        '  var style = doc.getElementById(styleId);',
        '  if (!style) {',
        '    style = doc.createElement("style");',
        '    style.id = styleId;',
        '    doc.head.appendChild(style);',
        '  }',
        `  var scale = ${JSON.stringify(normalizedScale)};`,
        '  if (scale >= 0.999) {',
        '    style.textContent = "";',
        '    root.style.removeProperty("transform");',
        '    root.style.removeProperty("transform-origin");',
        '    root.style.removeProperty("width");',
        '    root.style.removeProperty("min-height");',
        '    return;',
        '  }',
        '  style.textContent = [',
        '    "html, body { overflow: auto !important; overscroll-behavior: auto !important; }",',
        '    "body { margin: 0 !important; }",',
        `    "#root { transform: scale(${normalizedScale}) !important; transform-origin: top left !important; width: ${widthPercent} !important; min-height: ${minHeight} !important; }"`,
        '  ].join("\\n");',
        '})();',
        'true;',
      ].join('\n'),
    );
  }

  function broadcastSharedObjectUpdate(key: string, value: unknown) {
    sendHostMessage({
      type: 'shared-object-updated',
      key,
      value,
    });
  }

  function sendPersistedAtomSync() {
    sendHostMessage({
      type: 'persisted-atom-sync',
      state: persistedAtomStateRef.current,
    });
  }

  function updateWorkspaceRoots(nextRoots: string[], preferredRoot?: string | null) {
    const normalizedRoots = uniqueStringValues(
      nextRoots
        .map((root) => normalizeWorkspaceRootCandidate(root))
        .filter((root): root is string => root !== null),
    );
    const currentActiveRoot =
      normalizeWorkspaceRootCandidate(preferredRoot) ??
      hostStateRef.current.activeWorkspaceRoots[0] ??
      null;
    const nextActiveRoots =
      currentActiveRoot && normalizedRoots.includes(currentActiveRoot)
        ? [currentActiveRoot]
        : normalizedRoots.length > 0
          ? [normalizedRoots[0]]
          : [];
    if (
      shallowStringArrayEquals(hostStateRef.current.workspaceRootOptions, normalizedRoots) &&
      shallowStringArrayEquals(hostStateRef.current.activeWorkspaceRoots, nextActiveRoots)
    ) {
      return;
    }

    hostStateRef.current.workspaceRootOptions = normalizedRoots;
    hostStateRef.current.activeWorkspaceRoots = nextActiveRoots;
    sendHostMessage({ type: 'workspace-root-options-updated' });
    sendHostMessage({ type: 'active-workspace-roots-updated' });
  }

  function mergeWorkspaceRoots(nextRoots: string[], preferredRoot?: string | null) {
    const mergedRoots = [
      ...hostStateRef.current.workspaceRootOptions,
      ...nextRoots,
    ];
    updateWorkspaceRoots(mergedRoots, preferredRoot);
  }

  function deriveCodexHome() {
    return '/Users/boomyao/.codex';
  }

  function deriveLocaleInfo() {
    const locale =
      Intl.DateTimeFormat().resolvedOptions().locale || 'en-US';
    return {
      ideLocale: locale,
      systemLocale: locale,
    };
  }

  function normalizeWorkspaceRootCandidate(value: unknown) {
    if (typeof value !== 'string') {
      return null;
    }

    const normalized = normalizePathString(value.trim());
    if (!normalized || isSessionArtifactPath(normalized, deriveCodexHome())) {
      return null;
    }

    return normalized;
  }

  function setActiveWorkspaceRoot(root: string) {
    const normalizedRoot = normalizeWorkspaceRootCandidate(root);
    if (!normalizedRoot) {
      return;
    }

    const nextOptions = hostStateRef.current.workspaceRootOptions.includes(normalizedRoot)
      ? hostStateRef.current.workspaceRootOptions
      : [normalizedRoot, ...hostStateRef.current.workspaceRootOptions];
    const nextActiveRoots = [normalizedRoot];
    const optionsChanged = !shallowStringArrayEquals(
      hostStateRef.current.workspaceRootOptions,
      nextOptions,
    );
    const activeChanged = !shallowStringArrayEquals(
      hostStateRef.current.activeWorkspaceRoots,
      nextActiveRoots,
    );
    if (!optionsChanged && !activeChanged) {
      return;
    }

    hostStateRef.current.workspaceRootOptions = nextOptions;
    hostStateRef.current.activeWorkspaceRoots = nextActiveRoots;
    if (optionsChanged) {
      sendHostMessage({ type: 'workspace-root-options-updated' });
    }
    if (activeChanged) {
      sendHostMessage({ type: 'active-workspace-roots-updated' });
    }
  }

  function mergeWorkspaceRootsFromResult(result: unknown) {
    const payload = asObject(result);
    if (!payload) {
      return;
    }

    const thread = asObject(payload.thread);
    const data = Array.isArray(payload.data) ? payload.data : null;
    const candidates: string[] = [];

    const collectThreadRoots = (value: Record<string, unknown> | null) => {
      if (!value) {
        return;
      }
      const cwd = normalizeWorkspaceRootCandidate(value.cwd);
      if (cwd) {
        candidates.push(cwd);
      }
    };

    const topLevelCwd = normalizeWorkspaceRootCandidate(payload.cwd);
    if (topLevelCwd) {
      candidates.push(topLevelCwd);
    }
    collectThreadRoots(thread);
    for (const item of data ?? []) {
      collectThreadRoots(asObject(item));
    }

    if (candidates.length > 0) {
      mergeWorkspaceRoots(candidates, candidates[0] ?? null);
    }
  }

  function integrateAppServerResult(method: string, result: unknown) {
    switch (method) {
      case 'account/read':
        hostStateRef.current.account = result;
        return;
      case 'account/rateLimits/read':
        hostStateRef.current.rateLimit = result;
        return;
      case 'thread/list':
        prefetchRecentThreadReads(result);
        mergeWorkspaceRootsFromResult(result);
        return;
      case 'thread/read':
      case 'thread/start':
      case 'thread/resume':
        mergeWorkspaceRootsFromResult(result);
        return;
      default:
        return;
    }
  }

  function rememberPendingTurnCompletion(threadId: string | null, turnId: string | null) {
    if (!threadId || !turnId) {
      return;
    }
    pendingTurnCompletionRef.current.set(threadId, turnId);
  }

  function clearPendingTurnCompletion(threadId: string | null, turnId?: string | null) {
    if (!threadId) {
      return;
    }
    const pendingTurnId = pendingTurnCompletionRef.current.get(threadId);
    if (!pendingTurnId) {
      return;
    }
    if (turnId && pendingTurnId !== turnId) {
      return;
    }
    pendingTurnCompletionRef.current.delete(threadId);
  }

  function isThreadStatusTerminal(status: AppServerThreadStatus | null | undefined) {
    return status?.type === 'idle' || status?.type === 'systemError';
  }

  function emitSyntheticAppServerNotification(notification: AppServerServerNotification) {
    if (ENABLE_SERVER_DEBUG_LOGS) {
      console.log('[app-server-notification][synthetic]', summarizeAppServerNotification(notification));
    }
    handleAppServerNotification(notification);
  }

  function emitSyntheticTurnCompletion(threadId: string, turn: AppServerTurn) {
    emitSyntheticAppServerNotification({
      method: 'turn/started',
      params: {
        threadId,
        turn,
      },
    });

    for (const item of turn.items) {
      if (item.type === 'userMessage') {
        continue;
      }

      emitSyntheticAppServerNotification({
        method: 'item/started',
        params: {
          threadId,
          turnId: turn.id,
          item,
        },
      });
      emitSyntheticAppServerNotification({
        method: 'item/completed',
        params: {
          threadId,
          turnId: turn.id,
          item,
        },
      });
    }

    emitSyntheticAppServerNotification({
      method: 'turn/completed',
      params: {
        threadId,
        turn,
      },
    });
  }

  function reconcilePendingTurnCompletion(threadId: string, status: AppServerThreadStatus) {
    if (!isThreadStatusTerminal(status)) {
      return;
    }

    const pendingTurnId = pendingTurnCompletionRef.current.get(threadId);
    if (!pendingTurnId) {
      return;
    }

    const inflight = pendingTurnReconciliationsRef.current.get(threadId);
    if (inflight) {
      return;
    }

    const reconciliationPromise = (async () => {
      try {
        const result = await sendAppServerRequest<AppServerThreadReadResponse>('thread/read', {
          threadId,
          includeTurns: true,
        });
        integrateAppServerResult('thread/read', result);

        const thread = result.thread;
        const latestTurn = Array.isArray(thread.turns) ? thread.turns[thread.turns.length - 1] : null;
        if (!latestTurn || latestTurn.status === 'inProgress') {
          return;
        }

        if (latestTurn.id !== pendingTurnId) {
          const completedPendingTurn = thread.turns.find((turn) => turn.id === pendingTurnId) ?? null;
          if (!completedPendingTurn || completedPendingTurn.status === 'inProgress') {
            return;
          }
          emitSyntheticTurnCompletion(threadId, completedPendingTurn);
          clearPendingTurnCompletion(threadId, completedPendingTurn.id);
          return;
        }

        emitSyntheticTurnCompletion(threadId, latestTurn);
        clearPendingTurnCompletion(threadId, latestTurn.id);
      } catch (error) {
        console.warn('Failed to reconcile pending thread completion', {
          threadId,
          pendingTurnId,
          error,
        });
      } finally {
        pendingTurnReconciliationsRef.current.delete(threadId);
      }
    })();

    pendingTurnReconciliationsRef.current.set(threadId, reconciliationPromise);
  }

  async function proxyHttpFetch(
    message: Record<string, unknown>,
    requestId: string,
    rawUrl: string,
  ) {
    const method =
      typeof message.method === 'string' && message.method.trim().length > 0
        ? message.method.trim().toUpperCase()
        : 'GET';
    const headersObject = asObject(message.headers);
    const headers: Record<string, string> = {};
    for (const [key, value] of Object.entries(headersObject ?? {})) {
      if (typeof value === 'string') {
        headers[key] = value;
      }
    }
    if (activeAuthToken) {
      if (!headers.Authorization) {
        headers.Authorization = `Bearer ${activeAuthToken}`;
      }
    }

    const requestInit: RequestInit = {
      method,
      headers,
    };
    if (typeof message.body === 'string') {
      requestInit.body = message.body;
    }

    const response = await fetch(rawUrl, requestInit);
    if (response.status === 401) {
      void handleUnauthorizedBridgeAccess('A bridge request was rejected with HTTP 401.');
    }
    const responseText = await response.text();
    let responseBody: unknown = null;
    if (responseText.trim().length > 0) {
      try {
        responseBody = JSON.parse(responseText);
      } catch {
        responseBody = responseText;
      }
    }

    const responseHeaders: Record<string, string> = {};
    response.headers.forEach((value, key) => {
      responseHeaders[key] = value;
    });

    sendHostMessage(
      buildFetchSuccessResponse(
        requestId,
        responseBody,
        response.status,
        responseHeaders,
      ),
    );
  }

  function resolveFetchMethodPayload(method: string, params: unknown) {
    const payload = asObject(params);
    switch (method) {
      case 'get-global-state':
        return {
          value:
            payload && typeof payload.key === 'string'
              ? (globalStateRef.current[payload.key] ?? null)
              : null,
        };
      case 'set-global-state': {
        const key = payload && typeof payload.key === 'string' ? payload.key : null;
        if (!key) {
          return {
            success: false,
          };
        }

        const nextState = { ...globalStateRef.current };
        if (payload?.value === undefined) {
          delete nextState[key];
        } else {
          nextState[key] = payload.value;
        }
        globalStateRef.current = nextState;
        return {
          success: true,
        };
      }
      case 'get-configuration':
        return {
          value:
            payload && typeof payload.key === 'string'
              ? (configurationStateRef.current[payload.key] ?? null)
              : null,
        };
      case 'set-configuration': {
        const key = payload && typeof payload.key === 'string' ? payload.key : null;
        if (!key) {
          return {
            success: false,
          };
        }

        const nextState = { ...configurationStateRef.current };
        if (payload?.value === undefined) {
          delete nextState[key];
        } else {
          nextState[key] = payload.value;
        }
        configurationStateRef.current = nextState;
        return {
          success: true,
        };
      }
      case 'list-pinned-threads':
        return {
          threadIds: hostStateRef.current.pinnedThreadIds,
        };
      case 'set-thread-pinned': {
        const threadId =
          payload && typeof payload.threadId === 'string' ? payload.threadId : null;
        const pinned = payload?.pinned === true;
        if (!threadId) {
          return {
            success: false,
            threadIds: hostStateRef.current.pinnedThreadIds,
          };
        }
        const nextThreadIds = pinned
          ? uniqueStringValues([...hostStateRef.current.pinnedThreadIds, threadId])
          : hostStateRef.current.pinnedThreadIds.filter((value) => value !== threadId);
        hostStateRef.current.pinnedThreadIds = nextThreadIds;
        return {
          success: true,
          threadIds: nextThreadIds,
        };
      }
      case 'set-pinned-threads-order': {
        const threadIds = Array.isArray(payload?.threadIds)
          ? uniqueStringValues(payload.threadIds)
          : hostStateRef.current.pinnedThreadIds;
        hostStateRef.current.pinnedThreadIds = threadIds;
        return {
          success: true,
          threadIds,
        };
      }
      case 'extension-info':
        return DESKTOP_EXTENSION_INFO;
      case 'is-copilot-api-available':
        return {
          available: false,
        };
      case 'account-info':
        return asObject(hostStateRef.current.account) ?? {
          plan: null,
          accountId: null,
        };
      case 'list-pending-automation-run-threads':
        return {
          threadIds: [],
        };
      case 'inbox-items':
        return {
          items: [],
        };
      case 'list-automations':
        return {
          items: [],
        };
      case 'local-environments':
        return {
          environments: [],
        };
      case 'open-in-targets':
        return {
          targets: [],
        };
      case 'gh-cli-status':
        return {
          isInstalled: false,
          isAuthenticated: false,
        };
      case 'gh-pr-status':
        return {
          status: 'success',
          hasOpenPr: false,
          isDraft: false,
          url: null,
          canMerge: false,
          ciStatus: null,
        };
      case 'recommended-skills':
        return {
          skills: [],
          error: null,
        };
      case 'ide-context':
        return {
          status: 'disconnected',
          connected: false,
          context: null,
        };
      case 'local-custom-agents':
        return {
          agents: [],
        };
      case 'hotkey-window-hotkey-state':
        return {
          supported: false,
          configuredHotkey: null,
          state: null,
        };
      case 'fast-mode-rollout-metrics':
        return {
          enabled: true,
          estimatedSavedMs: 0,
          rolloutCountWithCompletedTurns: 0,
        };
      case 'os-info':
        return {
          platform: 'linux',
          isMacOS: false,
          isWindows: false,
          isLinux: true,
        };
      case 'locale-info':
        return deriveLocaleInfo();
      case 'active-workspace-roots':
        return {
          roots: hostStateRef.current.activeWorkspaceRoots,
        };
      case 'workspace-root-options':
        return {
          roots: hostStateRef.current.workspaceRootOptions,
          activeRoots: hostStateRef.current.activeWorkspaceRoots,
          labels: hostStateRef.current.workspaceRootLabels,
        };
      case 'has-custom-cli-executable':
        return {
          hasCustomCliExecutable: false,
        };
      case 'get-copilot-api-proxy-info':
        return null;
      case 'codex-home':
        return {
          codexHome: deriveCodexHome(),
          config: null,
          layers: [],
          origins: null,
        };
      case 'git-origins':
        return {
          origins: [],
        };
      case 'paths-exist': {
        const paths = Array.isArray(payload?.paths) ? payload.paths : [];
        return {
          existingPaths: uniqueStringValues(
            paths
              .filter((value): value is string => typeof value === 'string')
              .map(normalizePathString),
          ),
        };
      }
      case 'mcp-codex-config':
        return {
          config: null,
        };
      case 'config/read':
        return {
          config: { ...configurationStateRef.current },
          layers: [],
          origins: null,
        };
      case 'worktree-shell-environment-config':
        return {
          shellEnvironment: null,
        };
      case 'developer-instructions':
        return {
          instructions:
            typeof payload?.baseInstructions === 'string'
              ? payload.baseInstructions
              : null,
        };
      case 'thread-terminal-snapshot':
        return {
          session: {
            cwd: '',
            shell: 'unknown',
            buffer: '',
            truncated: false,
          },
        };
      case 'remote-workspace-directory-entries':
        return {
          directoryPath:
            typeof payload?.directoryPath === 'string'
              ? payload.directoryPath
              : '',
          entries: [],
        };
      default:
        return UNHANDLED_LOCAL_METHOD;
    }
  }

  function handleAppServerNotification(notification: AppServerServerNotification) {
    if (shouldTraceAppServerNotification(notification.method)) {
      if (ENABLE_SERVER_DEBUG_LOGS) {
        console.log('[app-server-notification]', summarizeAppServerNotification(notification));
      }
    }

    switch (notification.method) {
      case 'thread/started':
        invalidateThreadCaches(notification.params.thread.id);
        break;
      case 'thread/status/changed':
        invalidateThreadCaches(notification.params.threadId);
        reconcilePendingTurnCompletion(notification.params.threadId, notification.params.status);
        break;
      case 'turn/started':
        rememberPendingTurnCompletion(notification.params.threadId, notification.params.turn.id);
        invalidateThreadCaches(notification.params.threadId);
        break;
      case 'turn/completed':
        clearPendingTurnCompletion(notification.params.threadId, notification.params.turn.id);
        invalidateThreadCaches(notification.params.threadId);
        break;
      case 'turn/diff/updated':
      case 'turn/plan/updated':
      case 'turn/diff/changed':
      case 'thread/token-usage/updated':
      case 'thread/token-usage/changed':
      case 'thread/tokenUsage/updated':
      case 'thread/tokenUsage/changed':
      case 'item/started':
      case 'item/completed':
      case 'item/agentMessage/delta':
      case 'item/plan/delta':
      case 'item/reasoning/summaryTextDelta':
      case 'item/reasoning/summaryPartAdded':
      case 'item/reasoning/textDelta':
      case 'item/commandExecution/outputDelta':
      case 'item/fileChange/outputDelta':
      case 'serverRequest/resolved':
      case 'error':
        invalidateThreadCaches(
          'threadId' in notification.params && typeof notification.params.threadId === 'string'
            ? notification.params.threadId
            : null,
        );
        break;
      case 'account/updated':
      case 'account/changed':
        hostStateRef.current.account = notification.params.account;
        break;
      case 'account/rateLimits/updated':
      case 'rate-limit/updated':
      case 'rate-limit/changed':
      case 'rateLimit/updated':
      case 'rateLimit/changed':
        hostStateRef.current.rateLimit = notification.params.rateLimit;
        break;
      default:
        break;
    }

    sendHostMessage({
      type: 'mcp-notification',
      hostId: LOCAL_HOST_ID,
      method: notification.method,
      params: notification.params,
      notification,
    });
  }

  function refreshAppServerSnapshots(
    client: CodexAppServerClient,
    connectionVersion: number,
  ) {
    void Promise.allSettled([
      client.readAccount().then((account) => {
        if (connectionVersion !== appServerConnectionVersionRef.current) {
          return;
        }
        hostStateRef.current.account = account;
      }),
      client.readRateLimits().then((rateLimit) => {
        if (connectionVersion !== appServerConnectionVersionRef.current) {
          return;
        }
        hostStateRef.current.rateLimit = rateLimit;
      }),
    ]).then((results) => {
      const [accountResult, rateLimitResult] = results;
      if (accountResult.status === 'rejected') {
        console.warn('Failed to read account from app-server', accountResult.reason);
      }
      if (rateLimitResult.status === 'rejected') {
        console.warn('Failed to read rate limits from app-server', rateLimitResult.reason);
      }
    });
  }

function buildAppServerRequestCacheKey(method: string, params: unknown) {
  return `${method}:${JSON.stringify(normalizeAppServerRequestParams(method, params))}`;
}

  function getAppServerRequestCacheTtlMs(method: string) {
    switch (method) {
      case 'thread/list':
        return 5_000;
      case 'config/read':
      case 'app/list':
      case 'mcpServerStatus/list':
      case 'model/list':
        return 60_000;
      default:
        return 0;
    }
  }

  function primeAppServerResultCache(method: string, params: unknown, result: unknown) {
    const ttlMs = getAppServerRequestCacheTtlMs(method);
    if (ttlMs <= 0) {
      return;
    }

    const expiresAt = Date.now() + ttlMs;
    appServerResultCacheRef.current.set(buildAppServerRequestCacheKey(method, params), {
      expiresAt,
      result,
    });

    if (method === 'thread/list') {
      const payload = asObject(result);
      const data = Array.isArray(payload?.data) ? payload.data : [];
      for (const entry of data) {
        const thread = asObject(entry);
        const threadId = thread && typeof thread.id === 'string' ? thread.id : null;
        if (!threadId) {
          continue;
        }
        appServerResultCacheRef.current.set(
          buildAppServerRequestCacheKey('thread/read', { threadId, includeTurns: true }),
          {
            expiresAt,
            result: { thread },
          },
        );
      }
    }

    if (method === 'thread/resume' || method === 'thread/start') {
      const payload = asObject(result);
      const thread = asObject(payload?.thread);
      const threadId = thread && typeof thread.id === 'string' ? thread.id : null;
      if (threadId) {
        appServerResultCacheRef.current.set(
          buildAppServerRequestCacheKey('thread/read', { threadId, includeTurns: true }),
          {
            expiresAt,
            result: { thread },
          },
        );
      }
    }
  }

  function invalidateAppServerResultCache(predicate: (cacheKey: string) => boolean) {
    for (const cacheKey of Array.from(appServerResultCacheRef.current.keys())) {
      if (predicate(cacheKey)) {
        appServerResultCacheRef.current.delete(cacheKey);
      }
    }

    for (const cacheKey of Array.from(appServerPendingRequestsRef.current.keys())) {
      if (predicate(cacheKey)) {
        appServerPendingRequestsRef.current.delete(cacheKey);
      }
    }
  }

  function invalidateThreadCaches(threadId?: string | null) {
    invalidateAppServerResultCache((cacheKey) => {
      if (cacheKey.startsWith('thread/list:')) {
        return true;
      }

      if (!threadId) {
        return false;
      }

      return (
        cacheKey ===
          buildAppServerRequestCacheKey('thread/read', {
            threadId,
            includeTurns: true,
          }) ||
        cacheKey ===
          buildAppServerRequestCacheKey('thread/resume', {
            threadId,
          })
      );
    });
  }

  async function sendAppServerRequest<T = unknown>(method: string, params: unknown): Promise<T> {
    const cacheKey = buildAppServerRequestCacheKey(method, params);
    const ttlMs = getAppServerRequestCacheTtlMs(method);
    if (ttlMs > 0) {
      const cached = appServerResultCacheRef.current.get(cacheKey);
      if (cached && cached.expiresAt > Date.now()) {
        if (shouldTraceAppServerMethod(method)) {
          if (ENABLE_SERVER_DEBUG_LOGS) {
            console.log('[app-server-rpc]', method, 'cache-hit', summarizeAppServerParams(params));
          }
        }
        return cached.result as T;
      }
      appServerResultCacheRef.current.delete(cacheKey);

      const inflight = appServerPendingRequestsRef.current.get(cacheKey);
      if (inflight) {
        if (shouldTraceAppServerMethod(method)) {
          if (ENABLE_SERVER_DEBUG_LOGS) {
            console.log('[app-server-rpc]', method, 'join-inflight', summarizeAppServerParams(params));
          }
        }
        return inflight as Promise<T>;
      }
    }

    const startedAt = Date.now();
    if (shouldTraceAppServerMethod(method)) {
      if (ENABLE_SERVER_DEBUG_LOGS) {
        console.log('[app-server-rpc]', method, 'start', summarizeAppServerParams(params));
      }
    }
    const requestPromise = (async () => {
      const client = await ensureAppServerClient();
      const result = await client.sendRequest<T>(method, params ?? {});
      const durationMs = Date.now() - startedAt;
      if (shouldTraceAppServerMethod(method) || durationMs >= APP_SERVER_REQUEST_LOG_THRESHOLD_MS) {
        if (ENABLE_SERVER_DEBUG_LOGS) {
          console.log(
            '[app-server-rpc]',
            method,
            `${durationMs}ms`,
            summarizeAppServerParams(params),
          );
        }
      }
      primeAppServerResultCache(method, params, result);
      return result;
    })();

    if (ttlMs > 0) {
      appServerPendingRequestsRef.current.set(cacheKey, requestPromise);
      requestPromise.finally(() => {
        appServerPendingRequestsRef.current.delete(cacheKey);
      });
    }

    return requestPromise;
  }

  function prefetchRecentThreadReads(result: unknown) {
    const payload = asObject(result);
    const data = Array.isArray(payload?.data) ? payload.data : [];
    const recentThreadIds = data
      .map((entry) => asObject(entry))
      .map((thread) => (thread && typeof thread.id === 'string' ? thread.id : null))
      .filter((threadId): threadId is string => threadId !== null)
      .slice(0, 5);

    for (const threadId of recentThreadIds) {
      void sendAppServerRequest('thread/read', {
        threadId,
        includeTurns: true,
      }).catch(() => {
        // Ignore opportunistic prefetch failures.
      });
    }
  }

  async function ensureAppServerClient() {
    const currentClient = appServerClientRef.current;
    if (currentClient?.isConnected()) {
      return currentClient;
    }

    if (appServerConnectPromiseRef.current) {
      return appServerConnectPromiseRef.current;
    }

    const connectionVersion = appServerConnectionVersionRef.current;
    const nextClient = new CodexAppServerClient({
      endpoint: resolvedServerEndpoint,
      headers: buildAuthHeaders(activeAuthToken),
      onNotification: (notification) => {
        if (connectionVersion !== appServerConnectionVersionRef.current) {
          return;
        }
        handleAppServerNotification(notification);
      },
      onServerRequest: (request) =>
        new Promise((resolve, reject) => {
          if (ENABLE_SERVER_DEBUG_LOGS) {
            console.log('[app-server-server-request]', {
              method: request.method,
              id: request.id,
            });
          }
          pendingServerRequestResolversRef.current.set(request.id, {
            resolve,
            reject,
            method: request.method,
          });
          sendHostMessage({
            type: 'mcp-request',
            hostId: LOCAL_HOST_ID,
            request,
          });
        }),
      onClose: () => {
        if (connectionVersion !== appServerConnectionVersionRef.current) {
          return;
        }
        console.warn('App-server connection closed.');
        void handleUnauthorizedBridgeAccess('The bridge closed the app-server session.');
      },
      onError: (error) => {
        if (connectionVersion !== appServerConnectionVersionRef.current) {
          return;
        }
        console.warn('App-server transport error', error);
      },
    });

    const connectPromise = nextClient
      .connect()
      .then(() => {
        if (connectionVersion !== appServerConnectionVersionRef.current) {
          nextClient.disconnect();
          throw new Error('Discarded stale app-server connection.');
        }

        appServerClientRef.current = nextClient;
        appServerConnectPromiseRef.current = null;
        return nextClient;
      })
      .catch((error) => {
        if (appServerClientRef.current === nextClient) {
          appServerClientRef.current = null;
        }
        appServerConnectPromiseRef.current = null;
        void handleUnauthorizedBridgeAccess(
          'The bridge rejected the app-server websocket connection.',
        );
        throw error;
      });

    appServerConnectPromiseRef.current = connectPromise;
    return connectPromise;
  }

  async function handleFetchMessage(message: Record<string, unknown>) {
    const requestId =
      typeof message.requestId === 'string' ? message.requestId : null;
    const rawUrl = typeof message.url === 'string' ? message.url : null;
    if (!requestId || !rawUrl) {
      return;
    }

    const method = resolveRequestMethodName(rawUrl);
    if (!method) {
      const proxyUrl = resolveServerFetchUrl(rawUrl, resolvedServerEndpoint);
      if (proxyUrl) {
        try {
          await proxyHttpFetch(message, requestId, proxyUrl);
        } catch (error) {
          console.warn('HTTP fetch server request failed', {
            rawUrl,
            proxyUrl,
            error,
          });
          sendHostMessage(
            buildFetchErrorResponse(requestId, normalizeErrorMessage(error), 500),
          );
        }
        return;
      }

      sendHostMessage(
        buildFetchErrorResponse(requestId, `Unsupported fetch URL: ${rawUrl}`, 501),
      );
      return;
    }

    const params = parseJsonBody(message.body);
    const startedAt = Date.now();
    try {
      const handled = resolveFetchMethodPayload(method, params);
      if (handled !== UNHANDLED_LOCAL_METHOD) {
        sendHostMessage(buildFetchSuccessResponse(requestId, handled));
        return;
      }

      const result = await sendAppServerRequest(method, params ?? {});
      integrateAppServerResult(method, result);
      sendHostMessage(buildFetchSuccessResponse(requestId, result));
      if (shouldTraceAppServerMethod(method)) {
        if (ENABLE_SERVER_DEBUG_LOGS) {
          console.log('[server-fetch]', method, `${Date.now() - startedAt}ms`);
        }
      }
    } catch (error) {
      console.warn('Fetch server request failed', { method, rawUrl, error });
      sendHostMessage(
        buildFetchErrorResponse(requestId, normalizeErrorMessage(error), 500),
      );
    }
  }

  async function handleMcpRequestMessage(message: Record<string, unknown>) {
    const request = asObject(message.request);
    const requestId = normalizeRequestId(request?.id);
    const method =
      request && typeof request.method === 'string' ? request.method : null;
    if (!request || requestId == null || !method) {
      return;
    }

    const startedAt = Date.now();
    try {
      if (
        method === 'turn/start' ||
        method === 'turn/interrupt' ||
        method === 'thread/start'
      ) {
        const threadId =
          typeof asObject(request.params)?.threadId === 'string'
            ? (asObject(request.params)?.threadId as string)
            : null;
        invalidateThreadCaches(threadId);
      }

      const handled = resolveFetchMethodPayload(method, request.params ?? {});
      if (handled !== UNHANDLED_LOCAL_METHOD) {
        sendHostMessage({
          type: 'mcp-response',
          hostId: LOCAL_HOST_ID,
          id: requestId,
          result: handled,
          message: {
            id: requestId,
            result: handled,
          },
          response: {
            id: requestId,
            result: handled,
          },
        });
        return;
      }

      const result = await sendAppServerRequest(method, request.params ?? {});
      integrateAppServerResult(method, result);
      if (method === 'turn/start') {
        const threadId =
          typeof asObject(request.params)?.threadId === 'string'
            ? (asObject(request.params)?.threadId as string)
            : null;
        const turnId =
          typeof asObject(asObject(result)?.turn)?.id === 'string'
            ? (asObject(asObject(result)?.turn)?.id as string)
            : null;
        rememberPendingTurnCompletion(threadId, turnId);
        console.log('[server-mcp-result]', summarizeAppServerResult(method, result));
      }
      sendHostMessage({
        type: 'mcp-response',
        hostId: LOCAL_HOST_ID,
        id: requestId,
        result,
        message: {
          id: requestId,
          result,
        },
        response: {
          id: requestId,
          result,
        },
      });
      if (shouldTraceAppServerMethod(method)) {
        if (ENABLE_SERVER_DEBUG_LOGS) {
          console.log('[server-mcp]', method, `${Date.now() - startedAt}ms`);
        }
      }
    } catch (error) {
      console.warn('MCP server request failed', {
        method,
        requestId,
        error,
      });
      sendHostMessage({
        type: 'mcp-response',
        hostId: LOCAL_HOST_ID,
        id: requestId,
        error: {
          message: normalizeErrorMessage(error),
        },
        message: {
          id: requestId,
          error: {
            message: normalizeErrorMessage(error),
          },
        },
        response: {
          id: requestId,
          error: {
            message: normalizeErrorMessage(error),
          },
        },
      });
    }
  }

  function ensureTerminalSnapshot(sessionId: string, cwd: string) {
    if (terminalSnapshotsRef.current[sessionId]) {
      return terminalSnapshotsRef.current[sessionId];
    }
    const snapshot: TerminalSnapshot = {
      cwd,
      shell: 'zsh',
      buffer: '',
      truncated: false,
    };
    terminalSnapshotsRef.current = {
      ...terminalSnapshotsRef.current,
      [sessionId]: snapshot,
    };
    return snapshot;
  }

  function handleTerminalBridgeMessage(message: Record<string, unknown>) {
    const sessionId =
      typeof message.sessionId === 'string' ? message.sessionId : null;
    const cwd = typeof message.cwd === 'string' ? message.cwd : '';
    if (!sessionId) {
      return;
    }

    switch (message.type) {
      case 'terminal-create':
      case 'terminal-attach': {
        const snapshot = ensureTerminalSnapshot(sessionId, cwd);
        sendHostMessage({
          type: 'terminal-attached',
          sessionId,
          cwd: snapshot.cwd,
          shell: snapshot.shell,
        });
        sendHostMessage({
          type: 'terminal-init-log',
          sessionId,
          log: snapshot.buffer,
        });
        return;
      }
      case 'terminal-close': {
        const nextSnapshots = { ...terminalSnapshotsRef.current };
        delete nextSnapshots[sessionId];
        terminalSnapshotsRef.current = nextSnapshots;
        sendHostMessage({
          type: 'terminal-exit',
          sessionId,
          code: 0,
          signal: null,
        });
        return;
      }
      case 'terminal-write':
      case 'terminal-run-action':
      case 'terminal-resize':
        ensureTerminalSnapshot(sessionId, cwd);
        return;
      default:
        return;
    }
  }

  function buildGitWorkerStableMetadata(cwd: string) {
    return {
      cwd,
      root: cwd,
      commonDir: cwd,
      gitDir: null,
      branch: null,
      upstreamBranch: null,
      headSha: null,
      originUrl: null,
      isRepository: false,
      isWorktree: false,
      worktreeRoot: cwd,
    };
  }

  function handleWorkerBridgeMessage(payload: Record<string, unknown>) {
    const workerId =
      typeof payload.workerId === 'string' ? payload.workerId : null;
    const workerPayload = asObject(payload.payload);
    if (!workerId || !workerPayload) {
      return;
    }

    if (workerPayload.type === 'worker-request-cancel') {
      return;
    }

    if (workerPayload.type !== 'worker-request') {
      return;
    }

    const request = asObject(workerPayload.request);
    const requestId =
      request && typeof request.id === 'string' ? request.id : null;
    const method =
      request && typeof request.method === 'string' ? request.method : null;
    const params = asObject(request?.params);
    if (!requestId || !method) {
      return;
    }

    if (workerId === 'git' && method === 'stable-metadata') {
      const cwd = typeof params?.cwd === 'string' ? params.cwd : '';
      sendWorkerMessage(workerId, {
        type: 'worker-response',
        workerId,
        response: {
          id: requestId,
          method,
          result: {
            type: 'ok',
            value: buildGitWorkerStableMetadata(cwd),
          },
        },
      });
      return;
    }

    sendWorkerMessage(workerId, {
      type: 'worker-response',
      workerId,
      response: {
        id: requestId,
        method,
        result: {
          type: 'error',
          error: {
            message: `Unsupported worker request: ${workerId}/${method}`,
          },
        },
      },
    });
  }

  function handleRendererReady() {
    syncViewportToRenderer(viewportScale);
    sendPersistedAtomSync();
    sendHostMessage({
      type: 'custom-prompts-updated',
      prompts: [],
    });
    sendHostMessage({
      type: 'app-update-ready-changed',
      isUpdateReady: false,
    });
    sendHostMessage({
      type: 'electron-window-focus-changed',
      isFocused: true,
    });
    sendHostMessage({ type: 'workspace-root-options-updated' });
    sendHostMessage({ type: 'active-workspace-roots-updated' });
    for (const [key, value] of Object.entries(sharedObjectsRef.current)) {
      broadcastSharedObjectUpdate(key, value);
    }
  }

  async function handleOpenExternalUrl(payload: unknown) {
    const url = asObject(payload)?.url;
    if (typeof url !== 'string' || url.length === 0) {
      return;
    }

    try {
      await Linking.openURL(url);
    } catch (error) {
      console.warn('Failed to open external url', error);
    }
  }

  function handleRendererMessage(message: Record<string, unknown>) {
    const type = typeof message.type === 'string' ? message.type : null;
    if (!type) {
      return;
    }

    switch (type) {
      case 'ready':
        handleRendererReady();
        return;
      case 'persisted-atom-sync-request':
        sendPersistedAtomSync();
        return;
      case 'persisted-atom-update': {
        const key = typeof message.key === 'string' ? message.key : null;
        if (!key) {
          return;
        }
        const deleted = message.deleted === true;
        const nextState = { ...persistedAtomStateRef.current };
        if (deleted) {
          delete nextState[key];
        } else {
          nextState[key] = message.value;
        }
        persistedAtomStateRef.current = nextState;
        sendHostMessage({
          type: 'persisted-atom-updated',
          key,
          value: deleted ? null : message.value,
          deleted,
        });
        return;
      }
      case 'persisted-atom-reset':
        persistedAtomStateRef.current = { ...DEFAULT_PERSISTED_ATOM_STATE };
        sendPersistedAtomSync();
        return;
      case 'shared-object-subscribe': {
        const key = typeof message.key === 'string' ? message.key : null;
        if (!key) {
          return;
        }
        const currentCount = sharedObjectSubscribersRef.current.get(key) ?? 0;
        sharedObjectSubscribersRef.current.set(key, currentCount + 1);
        broadcastSharedObjectUpdate(key, sharedObjectsRef.current[key] ?? null);
        return;
      }
      case 'shared-object-unsubscribe': {
        const key = typeof message.key === 'string' ? message.key : null;
        if (!key) {
          return;
        }
        const currentCount = sharedObjectSubscribersRef.current.get(key) ?? 0;
        if (currentCount <= 1) {
          sharedObjectSubscribersRef.current.delete(key);
        } else {
          sharedObjectSubscribersRef.current.set(key, currentCount - 1);
        }
        return;
      }
      case 'shared-object-set': {
        const key = typeof message.key === 'string' ? message.key : null;
        if (!key) {
          return;
        }
        sharedObjectsRef.current = {
          ...sharedObjectsRef.current,
          [key]: message.value,
        };
        broadcastSharedObjectUpdate(key, message.value);
        return;
      }
      case 'show-settings':
        openSettingsForBridge(activeBridge);
        return;
      case 'open-in-browser':
        void handleOpenExternalUrl(message);
        return;
      case 'fetch':
        void handleFetchMessage(message);
        return;
      case 'cancel-fetch':
        return;
      case 'fetch-stream': {
        const requestId =
          typeof message.requestId === 'string' ? message.requestId : null;
        if (requestId) {
          sendHostMessage({
            type: 'fetch-stream-error',
            requestId,
            error: 'Streaming fetch is not supported in Codex Mobile yet.',
          });
        }
        return;
      }
      case 'cancel-fetch-stream':
        return;
      case 'mcp-request':
        void handleMcpRequestMessage(message);
        return;
      case 'mcp-response': {
        const response = asObject(message.response);
        const requestId = normalizeRequestId(response?.id);
        if (requestId == null) {
          return;
        }

        if (ENABLE_SERVER_DEBUG_LOGS) {
          console.log('[renderer->host:mcp-response]', {
            id: requestId,
            hasError: asObject(response?.error) != null,
          });
        }

        const pending = pendingServerRequestResolversRef.current.get(requestId);
        if (!pending) {
          return;
        }

        pendingServerRequestResolversRef.current.delete(requestId);
        const errorObject = asObject(response?.error);
        if (errorObject) {
          pending.reject(
            new Error(
              typeof errorObject.message === 'string'
                ? errorObject.message
                : `Renderer declined ${pending.method}.`,
            ),
          );
          return;
        }

        pending.resolve(response?.result ?? {});
        return;
      }
      case 'show-diff':
        sendHostMessage({
          type: 'toggle-diff-panel',
          open: true,
        });
        return;
      case 'workspace-root-option-picked': {
        const root = typeof message.root === 'string' ? message.root.trim() : '';
        if (!root) {
          return;
        }
        setActiveWorkspaceRoot(root);
        return;
      }
      case 'electron-update-workspace-root-options': {
        const roots = Array.isArray(message.roots)
          ? message.roots.filter((value): value is string => typeof value === 'string')
          : [];
        updateWorkspaceRoots(roots);
        return;
      }
      case 'electron-rename-workspace-root-option': {
        const root = normalizeWorkspaceRootCandidate(message.root);
        if (
          !root ||
          !hostStateRef.current.workspaceRootOptions.includes(root)
        ) {
          return;
        }

        const label =
          typeof message.label === 'string' ? message.label.trim() : '';
        const nextLabels = { ...hostStateRef.current.workspaceRootLabels };
        if (label.length === 0) {
          delete nextLabels[root];
        } else {
          nextLabels[root] = label;
        }
        hostStateRef.current.workspaceRootLabels = nextLabels;
        sendHostMessage({ type: 'workspace-root-options-updated' });
        return;
      }
      case 'electron-set-active-workspace-root': {
        const root = typeof message.root === 'string' ? message.root.trim() : '';
        if (!root) {
          return;
        }
        setActiveWorkspaceRoot(root);
        return;
      }
      case 'electron-window-focus-request':
        sendHostMessage({
          type: 'electron-window-focus-changed',
          isFocused: true,
        });
        return;
      case 'terminal-create':
      case 'terminal-attach':
      case 'terminal-write':
      case 'terminal-run-action':
      case 'terminal-resize':
      case 'terminal-close':
        handleTerminalBridgeMessage(message);
        return;
      case 'log-message':
        console.log('[remote-shell]', message.level ?? 'info', message.message ?? '');
        return;
      case 'bridge-unimplemented':
      case 'view-focused':
      case 'power-save-blocker-set':
      case 'electron-set-window-mode':
      case 'electron-request-microphone-permission':
      case 'electron-set-badge-count':
      case 'desktop-notification-hide':
      case 'desktop-notification-show':
      case 'install-app-update':
      case 'open-debug-window':
      case 'open-thread-overlay':
      case 'thread-stream-state-changed':
      case 'set-telemetry-user':
      case 'toggle-trace-recording':
      case 'hotkey-window-enabled-changed':
      case 'electron-desktop-features-changed':
        return;
      default:
        console.log('[remote-shell] unhandled host message', type, message);
        return;
    }
  }

  function handleEnvelope(event: WebViewMessageEvent) {
    let envelope: NativeEnvelope;
    try {
      envelope = JSON.parse(event.nativeEvent.data) as NativeEnvelope;
    } catch {
      return;
    }

    if (!envelope.__codexMobile || typeof envelope.kind !== 'string') {
      return;
    }

    switch (envelope.kind) {
      case 'preload-ready':
        console.log('[remote-shell:preload-ready]', envelope.payload);
        return;
      case 'console':
        console.log('[remote-shell]', envelope.payload);
        return;
      case 'runtime-error':
        console.warn('[remote-shell:error]', envelope.payload);
        return;
      case 'bridge-send-message': {
        const payload = asObject(envelope.payload);
        if (payload) {
          handleRendererMessage(payload);
        }
        return;
      }
      case 'bridge-show-context-menu': {
        presentBridgeMenu('context', envelope.payload);
        return;
      }
      case 'bridge-show-application-menu': {
        presentBridgeMenu('application', envelope.payload);
        return;
      }
      case 'bridge-send-worker-message':
        {
          const payload = asObject(envelope.payload);
          if (payload) {
            handleWorkerBridgeMessage(payload);
          }
        }
        return;
      default:
        console.log('[remote-shell] unhandled native envelope', envelope.kind, envelope.payload);
        return;
    }
  }

  function sendBridgeResponse(requestId: string, result: unknown) {
    webViewRef.current?.injectJavaScript(
      [
        '(function () {',
        '  var host = window.__codexMobileHost;',
        '  if (host && typeof host.resolveBridgeRequest === "function") {',
        `    host.resolveBridgeRequest(${JSON.stringify(requestId)}, ${JSON.stringify(result ?? null)});`,
        '  }',
        '})();',
        'true;',
      ].join('\n'),
    );
  }

  function closeActiveBridgeMenu(result: Record<string, unknown> | null = null) {
    if (!activeBridgeMenu) {
      return;
    }
    const requestId = activeBridgeMenu.requestId;
    setActiveBridgeMenu(null);
    sendBridgeResponse(requestId, result);
  }

  function presentBridgeMenu(kind: BridgeMenuKind, payload: unknown) {
    const request = asObject(payload);
    const requestId =
      request && typeof request.requestId === 'string' ? request.requestId : null;
    if (!requestId) {
      console.warn('[bridge-menu] missing request id', payload);
      return;
    }

    const menuId =
      request && typeof request.menuId === 'string' ? request.menuId : null;
    const items = normalizeBridgeMenuItems(request?.items);
    if (activeBridgeMenu) {
      sendBridgeResponse(activeBridgeMenu.requestId, null);
    }

    if (items.length === 0) {
      console.log('[bridge-menu] no menu items', { kind, menuId, payload });
      sendBridgeResponse(requestId, null);
      setActiveBridgeMenu(null);
      return;
    }

    setActiveBridgeMenu({
      kind,
      menuId,
      requestId,
      stack: [
        {
          title: menuId,
          items,
        },
      ],
    });
  }

  function openBridgeSubmenu(item: BridgeMenuItem) {
    if (!item.submenu || item.submenu.length === 0) {
      return;
    }
    setActiveBridgeMenu((current) => {
      if (!current) {
        return current;
      }
      return {
        ...current,
        stack: [
          ...current.stack,
          {
            title: item.label || null,
            items: item.submenu ?? [],
          },
        ],
      };
    });
  }

  function popBridgeMenuLevel() {
    setActiveBridgeMenu((current) => {
      if (!current || current.stack.length <= 1) {
        return current;
      }
      return {
        ...current,
        stack: current.stack.slice(0, -1),
      };
    });
  }

  const currentBridgeMenuLevel =
    activeBridgeMenu && activeBridgeMenu.stack.length > 0
      ? activeBridgeMenu.stack[activeBridgeMenu.stack.length - 1]
      : null;
  const isEditingActiveBridge = editingBridgeId === activeBridge.id;

  function loadBridgeDraft(bridge: BridgeProfile) {
    setEditingBridgeId(bridge.id);
    setBridgeNameDraft(bridge.name);
    setServerEndpointDraft(bridge.serverEndpoint);
    setPairingCodeDraft('');
  }

  function openSettingsForBridge(bridge: BridgeProfile) {
    loadBridgeDraft(bridge);
    setSettingsOpen(true);
  }

  async function handleActivateBridge(bridge: BridgeProfile) {
    loadBridgeDraft(bridge);
    if (bridge.id === activeBridge.id) {
      return;
    }

    setSettingsBusy(true);
    try {
      const connection = await fetchConnectionTarget(bridge.serverEndpoint, bridge.authToken);
      if (connection.authMode === 'device-token' && !bridge.authToken) {
        setResolvedAuthMode(connection.authMode);
        setResolvedLocalAuthPage(connection.localAuthPage);
        Alert.alert(
          'Authentication required',
          connection.localAuthPage
            ? `Enter a pairing code before connecting. On the bridge host, open ${connection.localAuthPage} to generate and approve devices.`
            : 'Enter a pairing code before connecting to this bridge.',
        );
        return;
      }

      setPreferences((current) => ({
        ...current,
        activeBridgeId: bridge.id,
      }));
      setResolvedServerEndpoint(connection.recommendedServerEndpoint);
      setResolvedAuthMode(connection.authMode);
      setResolvedLocalAuthPage(connection.localAuthPage);
      setLoadError(null);
      setSettingsOpen(false);
    } catch (error) {
      Alert.alert('Connection failed', normalizeErrorMessage(error));
    } finally {
      setSettingsBusy(false);
    }
  }

  function handleCreateBridgeDraft() {
    setEditingBridgeId(generateBridgeId());
    setBridgeNameDraft(`Bridge ${preferences.bridges.length + 1}`);
    setServerEndpointDraft('');
    setPairingCodeDraft('');
  }

  function handleDeleteBridge(bridgeId: string) {
    const bridge = preferences.bridges.find((entry) => entry.id === bridgeId) ?? null;
    if (!bridge) {
      return;
    }
    if (preferences.bridges.length <= 1) {
      Alert.alert('Cannot delete bridge', 'At least one bridge profile must remain.');
      return;
    }

    const fallbackBridge =
      preferences.bridges.find((entry) => entry.id !== bridgeId) ?? activeBridge;

    Alert.alert('Delete bridge?', `Remove ${bridge.name} from this device?`, [
      {
        text: 'Cancel',
        style: 'cancel',
      },
      {
        text: 'Delete',
        style: 'destructive',
        onPress: () => {
          setPreferences((current) => {
            const remainingBridges = current.bridges.filter((entry) => entry.id !== bridgeId);
            const nextActiveBridgeId =
              current.activeBridgeId === bridgeId
                ? remainingBridges[0]?.id ?? createDefaultBridgeProfile().id
                : current.activeBridgeId;
            return {
              ...current,
              activeBridgeId: nextActiveBridgeId,
              bridges: remainingBridges.length > 0 ? remainingBridges : [createDefaultBridgeProfile()],
            };
          });
          loadBridgeDraft(fallbackBridge);
        },
      },
    ]);
  }

  async function handleSaveSettings() {
    const serverEndpoint = normalizeEndpoint(serverEndpointDraft);
    if (!serverEndpoint) {
      Alert.alert('Invalid server endpoint', 'Server endpoint cannot be empty.');
      return;
    }

    setSettingsBusy(true);
    try {
      const trimmedName = bridgeNameDraft.trim();
      const existingBridge = editingBridge;
      const existingEndpoint = existingBridge
        ? normalizeEndpoint(existingBridge.serverEndpoint)
        : null;
      let authToken =
        existingEndpoint === serverEndpoint ? existingBridge?.authToken ?? null : null;
      let connection = await fetchConnectionTarget(serverEndpoint, authToken);

      if (pairingCodeDraft.trim().length > 0) {
        const pairing = await completeDevicePairing(
          serverEndpoint,
          pairingCodeDraft,
          authToken,
        );
        authToken = pairing.accessToken;
        connection = await fetchConnectionTarget(serverEndpoint, authToken);
        if (!pairing.approved) {
          Alert.alert(
            'Pairing pending approval',
            connection.localAuthPage
              ? `This device has been registered and is waiting for approval on the bridge host. On that host, open ${connection.localAuthPage}.`
              : 'This device has been registered and is waiting for approval on the bridge host.',
          );
        }
      } else if (connection.authMode === 'device-token' && !authToken) {
        Alert.alert(
          'Authentication required',
          connection.localAuthPage
            ? `Enter a pairing code before connecting. On the bridge host, open ${connection.localAuthPage} to generate and approve devices.`
            : 'Enter a pairing code before connecting to this bridge.',
        );
        return;
      }

      const nextBridge = createBridgeProfile({
        id: editingBridgeId,
        name: trimmedName,
        serverEndpoint,
        authToken,
      });

      setPreferences((current) => {
        const bridgeExists = current.bridges.some((bridge) => bridge.id === nextBridge.id);
        return {
          ...current,
          activeBridgeId: nextBridge.id,
          bridges: bridgeExists
            ? current.bridges.map((bridge) => (bridge.id === nextBridge.id ? nextBridge : bridge))
            : [...current.bridges, nextBridge],
        };
      });
      setEditingBridgeId(nextBridge.id);
      setBridgeNameDraft(nextBridge.name);
      setServerEndpointDraft(nextBridge.serverEndpoint);
      setResolvedServerEndpoint(connection.recommendedServerEndpoint);
      setResolvedAuthMode(connection.authMode);
      setResolvedLocalAuthPage(connection.localAuthPage);
      setPairingCodeDraft('');
      setLoadError(null);
      setSettingsOpen(false);
    } catch (error) {
      Alert.alert('Connection failed', normalizeErrorMessage(error));
    } finally {
      setSettingsBusy(false);
    }
  }

  return (
    <View style={styles.root}>
      <StatusBar style={themeVariant === 'dark' ? 'light' : 'dark'} />

      <WebView
        ref={webViewRef}
        originWhitelist={['*']}
        source={{ uri: remoteShellUrl, headers: buildAuthHeaders(activeAuthToken) }}
        onMessage={handleEnvelope}
        javaScriptEnabled
        domStorageEnabled
        mixedContentMode="always"
        setBuiltInZoomControls
        setDisplayZoomControls={false}
        setSupportMultipleWindows={false}
        onError={(event) => {
          setLoadError(event.nativeEvent.description);
        }}
        onHttpError={(event) => {
          if (event.nativeEvent.statusCode === 401) {
            setLoadError('Authentication required. Checking bridge pairing status...');
            void handleUnauthorizedBridgeAccess('The remote shell returned HTTP 401.', {
              showBanner: true,
            });
            return;
          }
          setLoadError(`HTTP ${event.nativeEvent.statusCode}`);
        }}
        onLoadStart={() => {
          setLoadError(null);
        }}
        style={styles.webView}
      />

      <View style={styles.overlayRow}>
        <Pressable
          onPress={() => {
            openSettingsForBridge(activeBridge);
          }}
          style={styles.serverButton}
        >
          <Text style={styles.serverButtonText}>{activeBridge.name}</Text>
        </Pressable>
      </View>

      {loadError ? (
        <View style={styles.errorBanner}>
          <Text style={styles.errorTitle}>Remote shell failed to load</Text>
          <Text style={styles.errorBody}>{loadError}</Text>
          <Text style={styles.errorBody}>{remoteShellUrl}</Text>
        </View>
      ) : null}

      <Modal
        visible={activeBridgeMenu !== null}
        transparent
        animationType="fade"
        onRequestClose={() => closeActiveBridgeMenu()}
      >
        <Pressable style={styles.menuScrim} onPress={() => closeActiveBridgeMenu()}>
          <Pressable style={styles.menuSheet} onPress={() => {}}>
            {activeBridgeMenu && activeBridgeMenu.stack.length > 1 ? (
              <Pressable style={styles.menuBackButton} onPress={popBridgeMenuLevel}>
                <Text style={styles.menuBackText}>Back</Text>
              </Pressable>
            ) : null}

            <Text style={styles.menuTitle}>
              {currentBridgeMenuLevel?.title ||
                (activeBridgeMenu?.kind === 'context' ? 'Context menu' : 'Menu')}
            </Text>

            <View style={styles.menuList}>
              {currentBridgeMenuLevel?.items.map((item, index) =>
                item.type === 'separator' ? (
                  <View
                    key={item.id ?? `separator-${index}`}
                    style={styles.menuSeparator}
                  />
                ) : (
                  <Pressable
                    key={item.id ?? `${item.label}-${index}`}
                    disabled={!item.enabled}
                    onPress={() => {
                      if (item.submenu && item.submenu.length > 0) {
                        openBridgeSubmenu(item);
                        return;
                      }
                      if (item.id) {
                        closeActiveBridgeMenu({ id: item.id });
                        return;
                      }
                      closeActiveBridgeMenu();
                    }}
                    style={[
                      styles.menuItem,
                      !item.enabled && styles.menuItemDisabled,
                    ]}
                  >
                    <Text
                      style={[
                        styles.menuItemText,
                        !item.enabled && styles.menuItemTextDisabled,
                      ]}
                    >
                      {item.label}
                    </Text>
                    {item.submenu && item.submenu.length > 0 ? (
                      <Text style={styles.menuChevron}>›</Text>
                    ) : null}
                  </Pressable>
                ),
              )}
            </View>
          </Pressable>
        </Pressable>
      </Modal>

      <Modal
        visible={settingsOpen}
        transparent
        animationType="fade"
        onRequestClose={() => setSettingsOpen(false)}
      >
        <View style={styles.modalScrim}>
          <ScrollView
            style={styles.modalCard}
            contentContainerStyle={styles.modalContent}
            keyboardShouldPersistTaps="handled"
          >
            <Text style={styles.modalTitle}>Server Settings</Text>
            <Text style={styles.modalHint}>Active bridge: {activeBridge.name}</Text>
            <Text style={styles.modalHint}>
              Tap a saved bridge to connect. Edit the fields below, then use Save & Connect to
              update the selected bridge.
            </Text>
            <Text style={styles.modalLabel}>Bridges</Text>
            <View style={styles.bridgeList}>
              {preferences.bridges.map((bridge) => {
                const selected = bridge.id === editingBridgeId;
                const isActive = bridge.id === activeBridge.id;
                return (
                  <Pressable
                    key={bridge.id}
                    disabled={settingsBusy}
                    onPress={() => {
                      void handleActivateBridge(bridge);
                    }}
                    style={[
                      styles.bridgeRow,
                      selected && styles.bridgeRowSelected,
                      settingsBusy && styles.bridgeRowDisabled,
                    ]}
                  >
                    <View style={styles.bridgeRowBody}>
                      <Text style={styles.bridgeRowTitle}>{bridge.name}</Text>
                      <Text style={styles.bridgeRowSubtitle}>{bridge.serverEndpoint}</Text>
                    </View>
                    <Text style={[styles.bridgeBadge, isActive && styles.bridgeBadgeActive]}>
                      {isActive ? 'Active' : selected ? 'Selected' : 'Saved'}
                    </Text>
                  </Pressable>
                );
              })}
            </View>
            <Pressable style={styles.addBridgeButton} onPress={handleCreateBridgeDraft}>
              <Text style={styles.addBridgeButtonText}>New Bridge</Text>
            </Pressable>
            <Text style={styles.modalLabel}>Bridge Name</Text>
            <TextInput
              value={bridgeNameDraft}
              onChangeText={setBridgeNameDraft}
              autoCapitalize="words"
              autoCorrect={false}
              style={styles.input}
              placeholder="Home Mac"
              placeholderTextColor="#8e958f"
            />
            <Text style={styles.modalLabel}>Server Endpoint</Text>
            <TextInput
              value={serverEndpointDraft}
              onChangeText={setServerEndpointDraft}
              autoCapitalize="none"
              autoCorrect={false}
              style={styles.input}
              placeholder="ws://192.168.x.x:8787"
              placeholderTextColor="#8e958f"
            />
            <Text style={styles.modalHint}>
              The mobile app loads remote shell assets and Codex protocol traffic from this single
              service address.
            </Text>
            <Text style={styles.modalHint}>
              {isEditingActiveBridge
                ? `Resolved endpoint: ${resolvedServerEndpoint}`
                : 'Save this bridge to connect and resolve its public endpoint.'}
            </Text>
            <Text style={styles.modalHint}>
              {isEditingActiveBridge
                ? `Auth mode: ${resolvedAuthMode}`
                : 'Auth mode is shown for the active bridge only.'}
            </Text>
            {isEditingActiveBridge && resolvedLocalAuthPage ? (
              <Text style={styles.modalHint}>
                On the bridge host, open {resolvedLocalAuthPage} to generate pairing codes and
                approve devices.
              </Text>
            ) : null}
            <Text style={styles.modalLabel}>Pairing Code</Text>
            <TextInput
              value={pairingCodeDraft}
              onChangeText={setPairingCodeDraft}
              autoCapitalize="none"
              autoCorrect={false}
              style={styles.input}
              placeholder="Optional 8-digit code"
              placeholderTextColor="#8e958f"
            />
            <Text style={styles.modalHint}>
              Enter a pairing code only when the bridge asks for device pairing.
            </Text>
            {editingBridge ? (
              <Pressable
                style={[styles.actionButton, styles.dangerButton]}
                onPress={() => handleDeleteBridge(editingBridge.id)}
              >
                <Text style={styles.dangerButtonText}>Delete This Bridge</Text>
              </Pressable>
            ) : null}

            <Pressable
              style={styles.toggleRow}
              onPress={() => {
                setPreferences((current) => ({
                  ...current,
                  prefersDarkMode: !current.prefersDarkMode,
                }));
              }}
            >
              <Text style={styles.toggleLabel}>Force dark theme</Text>
              <Text style={styles.toggleValue}>
                {preferences.prefersDarkMode ? 'On' : 'Off'}
              </Text>
            </Pressable>

            <View style={styles.actionsRow}>
              <Pressable
                style={[styles.actionButton, styles.secondaryButton]}
                disabled={settingsBusy}
                onPress={() => setSettingsOpen(false)}
              >
                <Text style={styles.secondaryButtonText}>Cancel</Text>
              </Pressable>
              <Pressable
                style={[styles.actionButton, styles.primaryButton]}
                disabled={settingsBusy}
                onPress={() => {
                  void handleSaveSettings();
                }}
              >
                <Text style={styles.primaryButtonText}>
                  {settingsBusy ? 'Connecting…' : 'Save & Connect'}
                </Text>
              </Pressable>
            </View>
          </ScrollView>
        </View>
      </Modal>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#050706',
  },
  webView: {
    flex: 1,
    backgroundColor: 'transparent',
  },
  overlayRow: {
    position: 'absolute',
    right: 16,
    top: 56,
  },
  serverButton: {
    backgroundColor: 'rgba(12, 16, 14, 0.82)',
    borderColor: 'rgba(255, 255, 255, 0.12)',
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 8,
  },
  serverButtonText: {
    color: '#f1f4ef',
    fontSize: 12,
    fontWeight: '600',
  },
  errorBanner: {
    position: 'absolute',
    left: 16,
    right: 16,
    bottom: 24,
    backgroundColor: 'rgba(28, 7, 7, 0.94)',
    borderColor: 'rgba(240, 92, 92, 0.28)',
    borderRadius: 16,
    borderWidth: 1,
    padding: 16,
    gap: 6,
  },
  errorTitle: {
    color: '#ffd7d7',
    fontSize: 14,
    fontWeight: '700',
  },
  errorBody: {
    color: '#efb6b6',
    fontSize: 12,
    lineHeight: 18,
  },
  menuScrim: {
    flex: 1,
    justifyContent: 'flex-end',
    backgroundColor: 'rgba(0, 0, 0, 0.42)',
    padding: 16,
  },
  menuSheet: {
    backgroundColor: '#0d1110',
    borderColor: 'rgba(255, 255, 255, 0.08)',
    borderRadius: 22,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 16,
    gap: 10,
  },
  menuBackButton: {
    alignSelf: 'flex-start',
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  menuBackText: {
    color: '#9eb6a1',
    fontSize: 13,
    fontWeight: '600',
  },
  menuTitle: {
    color: '#f1f4ef',
    fontSize: 16,
    fontWeight: '700',
  },
  menuList: {
    gap: 6,
  },
  menuSeparator: {
    borderTopColor: 'rgba(255, 255, 255, 0.08)',
    borderTopWidth: 1,
    marginVertical: 4,
  },
  menuItem: {
    alignItems: 'center',
    borderRadius: 14,
    flexDirection: 'row',
    justifyContent: 'space-between',
    minHeight: 48,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  menuItemDisabled: {
    opacity: 0.42,
  },
  menuItemText: {
    color: '#eef2ec',
    flex: 1,
    fontSize: 15,
    fontWeight: '500',
  },
  menuItemTextDisabled: {
    color: '#728176',
  },
  menuChevron: {
    color: '#8fa794',
    fontSize: 22,
    lineHeight: 22,
    marginLeft: 12,
  },
  modalScrim: {
    flex: 1,
    alignItems: 'center',
    backgroundColor: 'rgba(0, 0, 0, 0.58)',
    justifyContent: 'center',
    padding: 20,
  },
  modalCard: {
    width: '100%',
    maxWidth: 420,
    maxHeight: '88%',
    backgroundColor: '#0d1110',
    borderColor: 'rgba(255, 255, 255, 0.08)',
    borderRadius: 20,
    borderWidth: 1,
  },
  modalContent: {
    padding: 20,
    gap: 12,
  },
  modalTitle: {
    color: '#f1f4ef',
    fontSize: 20,
    fontWeight: '700',
  },
  modalLabel: {
    color: '#cfd5cf',
    fontSize: 12,
    fontWeight: '600',
    letterSpacing: 0.4,
    textTransform: 'uppercase',
  },
  modalHint: {
    color: '#8e958f',
    fontSize: 12,
    lineHeight: 18,
  },
  bridgeList: {
    gap: 8,
  },
  bridgeRow: {
    alignItems: 'center',
    backgroundColor: '#111715',
    borderColor: 'rgba(255, 255, 255, 0.08)',
    borderRadius: 14,
    borderWidth: 1,
    flexDirection: 'row',
    gap: 10,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  bridgeRowSelected: {
    borderColor: 'rgba(128, 215, 165, 0.68)',
    backgroundColor: '#13211b',
  },
  bridgeRowDisabled: {
    opacity: 0.6,
  },
  bridgeRowBody: {
    flex: 1,
    gap: 3,
  },
  bridgeRowTitle: {
    color: '#eef2ec',
    fontSize: 14,
    fontWeight: '600',
  },
  bridgeRowSubtitle: {
    color: '#8e958f',
    fontSize: 12,
  },
  bridgeBadge: {
    color: '#9eb6a1',
    fontSize: 11,
    fontWeight: '700',
    letterSpacing: 0.3,
    textTransform: 'uppercase',
  },
  bridgeBadgeActive: {
    color: '#89c08b',
  },
  addBridgeButton: {
    alignItems: 'center',
    backgroundColor: '#171d1b',
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  addBridgeButtonText: {
    color: '#d5dbd6',
    fontSize: 14,
    fontWeight: '600',
  },
  input: {
    backgroundColor: '#111715',
    borderColor: 'rgba(255, 255, 255, 0.08)',
    borderRadius: 14,
    borderWidth: 1,
    color: '#f4f8f2',
    fontSize: 15,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  toggleRow: {
    alignItems: 'center',
    backgroundColor: '#111715',
    borderColor: 'rgba(255, 255, 255, 0.08)',
    borderRadius: 14,
    borderWidth: 1,
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 14,
    paddingVertical: 14,
  },
  toggleLabel: {
    color: '#eef2ec',
    fontSize: 14,
    fontWeight: '500',
  },
  toggleValue: {
    color: '#89c08b',
    fontSize: 13,
    fontWeight: '700',
  },
  actionsRow: {
    flexDirection: 'row',
    gap: 10,
    justifyContent: 'flex-end',
    marginTop: 4,
  },
  actionButton: {
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  secondaryButton: {
    backgroundColor: '#171d1b',
  },
  secondaryButtonText: {
    color: '#d5dbd6',
    fontSize: 14,
    fontWeight: '600',
  },
  primaryButton: {
    backgroundColor: '#f2f5ef',
  },
  primaryButtonText: {
    color: '#0b100f',
    fontSize: 14,
    fontWeight: '700',
  },
  dangerButton: {
    alignItems: 'center',
    backgroundColor: '#2a1313',
  },
  dangerButtonText: {
    color: '#ffc5c5',
    fontSize: 14,
    fontWeight: '700',
  },
});
