import { useEffect, useRef, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { SerializeAddon } from '@xterm/addon-serialize';
import '@xterm/xterm/css/xterm.css';
import { getApiBaseUrl } from '@/services/api/config';

interface XTerminalProps {
  instanceId: string;
}

const API_BASE_URL = getApiBaseUrl();

// Storage keys for terminal history per instance
const getHistoryKey = (instanceId: string) => `wild-terminal-history-${instanceId}`;

// Save terminal history to localStorage
const saveHistory = (instanceId: string, content: string) => {
  try {
    // Don't save empty content - this can happen during React Strict Mode double-invoke
    // where cleanup runs and terminal is already disposed
    if (!content || content.length === 0) {
      return;
    }

    // Limit storage size to prevent localStorage overflow (~500KB per instance)
    const maxSize = 500 * 1024;
    const trimmedContent = content.length > maxSize
      ? content.slice(-maxSize)
      : content;
    localStorage.setItem(getHistoryKey(instanceId), trimmedContent);
  } catch {
    // Ignore localStorage errors (quota exceeded, etc.)
  }
};

// Load terminal history from localStorage
const loadHistory = (instanceId: string): string | null => {
  try {
    return localStorage.getItem(getHistoryKey(instanceId));
  } catch {
    return null;
  }
};

export function XTerminal({ instanceId }: XTerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const serializeAddonRef = useRef<SerializeAddon | null>(null);
  const reconnectTimeoutRef = useRef<number | null>(null);
  const isCleaningUpRef = useRef(false);
  const currentInstanceIdRef = useRef<string>(instanceId);

  const connect = useCallback((term: Terminal, isNewSession: boolean) => {
    if (isCleaningUpRef.current) return;

    // Close any existing connection before creating a new one
    // Clear handlers first to prevent reconnect loop
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.onerror = null;
      wsRef.current.onmessage = null;
      wsRef.current.close();
      wsRef.current = null;
    }

    const wsUrl = API_BASE_URL.replace(/^http/, 'ws') +
                  `/api/v1/instances/${instanceId}/terminal/ws`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      // Send initial terminal size
      ws.send(JSON.stringify({
        type: 'resize',
        cols: term.cols,
        rows: term.rows,
      }));

      // Only show welcome message on new sessions (no restored history)
      if (isNewSession) {
        term.write('\x1b[32mConnected to Wild Central Terminal\x1b[0m\r\n');
        term.write('\x1b[90mWorking directory: instance data directory\x1b[0m\r\n');
        term.write('\x1b[90mkubectl, talosctl, and wild are configured for this instance.\x1b[0m\r\n\r\n');
      }
    };

    ws.onmessage = (event) => {
      if (event.data instanceof Blob) {
        event.data.text().then(text => term.write(text));
      } else {
        term.write(event.data);
      }
    };

    ws.onclose = () => {
      if (isCleaningUpRef.current) return;
      // Only reconnect if this is still the active WebSocket
      if (wsRef.current !== ws) return;

      term.write('\r\n\x1b[33mConnection lost. Reconnecting...\x1b[0m\r\n');
      // Auto-reconnect after 2 seconds (not a new session since we have history)
      reconnectTimeoutRef.current = window.setTimeout(() => {
        if (xtermRef.current && !isCleaningUpRef.current) {
          connect(xtermRef.current, false);
        }
      }, 2000);
    };

    ws.onerror = () => {
      // Error handling - onclose will be called after this
    };
  }, [instanceId]);

  useEffect(() => {
    if (!terminalRef.current) return;

    isCleaningUpRef.current = false;
    currentInstanceIdRef.current = instanceId;

    // Initialize xterm.js
    const term = new Terminal({
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
      fontSize: 14,
      scrollback: 5000,
      theme: {
        background: '#0f172a',
        foreground: '#e2e8f0',
        cursor: '#e2e8f0',
        cursorAccent: '#0f172a',
        selectionBackground: '#334155',
        black: '#1e293b',
        red: '#f87171',
        green: '#4ade80',
        yellow: '#facc15',
        blue: '#60a5fa',
        magenta: '#c084fc',
        cyan: '#22d3ee',
        white: '#f1f5f9',
        brightBlack: '#475569',
        brightRed: '#fca5a5',
        brightGreen: '#86efac',
        brightYellow: '#fde047',
        brightBlue: '#93c5fd',
        brightMagenta: '#d8b4fe',
        brightCyan: '#67e8f9',
        brightWhite: '#f8fafc',
      },
    });

    // Load addons
    const fitAddon = new FitAddon();
    const serializeAddon = new SerializeAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(serializeAddon);

    term.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;
    serializeAddonRef.current = serializeAddon;

    // Restore saved history for this instance
    const savedHistory = loadHistory(instanceId);
    const isNewSession = !savedHistory;
    if (savedHistory) {
      term.write(savedHistory);
      term.write('\r\n\x1b[90m--- Session restored ---\x1b[0m\r\n\r\n');
    }

    // Connect WebSocket
    connect(term, isNewSession);

    // Send input to WebSocket
    term.onData((data) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(data);
      }
    });

    // Send resize event to backend
    const sendResize = () => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({
          type: 'resize',
          cols: term.cols,
          rows: term.rows,
        }));
      }
    };

    // Handle resize
    const handleResize = () => {
      fitAddon.fit();
      sendResize();
    };
    window.addEventListener('resize', handleResize);

    // Also fit when the component container might have resized
    const resizeObserver = new ResizeObserver(() => {
      fitAddon.fit();
      sendResize();
    });
    if (terminalRef.current) {
      resizeObserver.observe(terminalRef.current);
    }

    return () => {
      isCleaningUpRef.current = true;
      window.removeEventListener('resize', handleResize);
      resizeObserver.disconnect();

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }

      // Save terminal history before cleanup
      if (serializeAddonRef.current && currentInstanceIdRef.current) {
        try {
          const content = serializeAddonRef.current.serialize();
          saveHistory(currentInstanceIdRef.current, content);
        } catch {
          // Ignore serialization errors
        }
      }

      if (wsRef.current) {
        const ws = wsRef.current;
        ws.onmessage = null;
        ws.onerror = null;
        ws.onclose = null;
        if (ws.readyState === WebSocket.CONNECTING) {
          // Let it finish connecting, then close immediately to avoid
          // "WebSocket is closed before the connection is established" warning
          ws.onopen = () => ws.close();
        } else {
          ws.close();
        }
        wsRef.current = null;
      }
      term.dispose();
    };
  }, [instanceId, connect]);

  return (
    <div
      ref={terminalRef}
      className="h-full w-full rounded-md overflow-hidden"
      style={{ padding: '8px', backgroundColor: '#0f172a' }}
    />
  );
}
