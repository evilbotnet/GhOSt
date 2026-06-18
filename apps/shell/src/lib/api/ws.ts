// Single multiplexed WebSocket to ghostd. Envelope: {topic, event, payload}.
// Subscribers register per-topic; the socket reconnects with backoff.
import { ensureToken, getToken } from './client';

export interface Envelope {
  topic: string;
  event: string;
  payload: unknown;
}

type Handler = (ev: Envelope) => void;

const handlers = new Map<string, Set<Handler>>();
const openCbs = new Set<() => void>();
let socket: WebSocket | null = null;
let backoff = 500;
let wanted = false;

function url(): string {
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  return `${proto}://${location.host}/api/v1/ws?token=${getToken()}`;
}

async function connect() {
  wanted = true;
  await ensureToken();
  if (socket && socket.readyState <= WebSocket.OPEN) return;
  socket = new WebSocket(url());
  socket.binaryType = 'arraybuffer';
  socket.onopen = () => {
    backoff = 500;
    for (const topic of handlers.keys()) {
      socket!.send(JSON.stringify({ topic, event: 'subscribe' }));
    }
    openCbs.forEach((cb) => cb());
  };
  socket.onmessage = (e) => {
    if (typeof e.data !== 'string') return;
    const env = JSON.parse(e.data) as Envelope;
    handlers.get(env.topic)?.forEach((h) => h(env));
    // prefix handlers, e.g. subscribe('term.') for all pty sessions
    for (const [key, set] of handlers) {
      if (key.endsWith('.') && env.topic.startsWith(key)) set.forEach((h) => h(env));
    }
  };
  socket.onclose = () => {
    socket = null;
    if (!wanted) return;
    setTimeout(connect, backoff);
    backoff = Math.min(backoff * 2, 8000);
  };
}

export function subscribe(topic: string, handler: Handler): () => void {
  let set = handlers.get(topic);
  if (!set) {
    set = new Set();
    handlers.set(topic, set);
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify({ topic, event: 'subscribe' }));
    }
  }
  set.add(handler);
  void connect();
  return () => {
    set!.delete(handler);
    if (set!.size === 0) {
      handlers.delete(topic);
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ topic, event: 'unsubscribe' }));
      }
    }
  };
}

export function send(topic: string, event: string, payload?: unknown) {
  if (socket?.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ topic, event, payload }));
  }
}

/** Run fn every time the socket (re)opens — for clients that must re-announce
 * state to the daemon after a reconnect (e.g. client-tool registration). */
export function onOpen(fn: () => void): () => void {
  openCbs.add(fn);
  return () => openCbs.delete(fn);
}
