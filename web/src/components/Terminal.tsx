import { useState, useRef, useEffect, useCallback } from "react";
import { useParams } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "./ui/card";
import { Button, Input } from "./ui";
import { TerminalSquare, Trash2, Copy, Check, Loader2 } from "lucide-react";
import { terminalApi, type TerminalExecResponse } from "../services/api/terminal";

interface OutputLine {
  type: "command" | "stdout" | "stderr" | "info";
  content: string;
}

const HISTORY_KEY = "wild-terminal-history";
const MAX_HISTORY = 50;

export function Terminal() {
  const { instanceId } = useParams<{ instanceId: string }>();
  const [command, setCommand] = useState("");
  const [output, setOutput] = useState<OutputLine[]>([
    { type: "info", content: "Welcome to Wild Central Terminal. Type commands to execute on the server." },
  ]);
  const [history, setHistory] = useState<string[]>(() => {
    try {
      const saved = localStorage.getItem(HISTORY_KEY);
      return saved ? JSON.parse(saved) : [];
    } catch {
      return [];
    }
  });
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [copied, setCopied] = useState(false);
  const outputRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // Auto-scroll to bottom when output changes
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [output]);

  // Save history to localStorage
  useEffect(() => {
    try {
      localStorage.setItem(HISTORY_KEY, JSON.stringify(history.slice(-MAX_HISTORY)));
    } catch {
      // Ignore localStorage errors
    }
  }, [history]);

  const execMutation = useMutation({
    mutationFn: (cmd: string) => terminalApi.exec(instanceId || '', cmd),
    onSuccess: (data: TerminalExecResponse) => {
      const newLines: OutputLine[] = [];

      if (data.stdout) {
        data.stdout.split("\n").forEach((line) => {
          if (line) newLines.push({ type: "stdout", content: line });
        });
      }

      if (data.stderr) {
        data.stderr.split("\n").forEach((line) => {
          if (line) newLines.push({ type: "stderr", content: line });
        });
      }

      if (data.exit_code !== 0) {
        newLines.push({ type: "info", content: `Exit code: ${data.exit_code}` });
      }

      setOutput((prev) => [...prev, ...newLines]);
    },
    onError: (error: Error) => {
      setOutput((prev) => [
        ...prev,
        { type: "stderr", content: `Error: ${error.message}` },
      ]);
    },
  });

  // Refocus input after command execution completes
  useEffect(() => {
    if (!execMutation.isPending) {
      inputRef.current?.focus();
    }
  }, [execMutation.isPending]);

  const handleExecute = useCallback(() => {
    const trimmedCommand = command.trim();
    if (!trimmedCommand || execMutation.isPending) return;

    // Add command to output
    setOutput((prev) => [
      ...prev,
      { type: "command", content: `$ ${trimmedCommand}` },
    ]);

    // Add to history (avoid duplicates at the end)
    setHistory((prev) => {
      const filtered = prev.filter((h) => h !== trimmedCommand);
      return [...filtered, trimmedCommand];
    });
    setHistoryIndex(-1);

    // Execute
    execMutation.mutate(trimmedCommand);
    setCommand("");
  }, [command, execMutation]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === "Enter") {
        handleExecute();
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        if (history.length === 0) return;
        const newIndex = historyIndex === -1 ? history.length - 1 : Math.max(0, historyIndex - 1);
        setHistoryIndex(newIndex);
        setCommand(history[newIndex]);
      } else if (e.key === "ArrowDown") {
        e.preventDefault();
        if (historyIndex === -1) return;
        const newIndex = historyIndex + 1;
        if (newIndex >= history.length) {
          setHistoryIndex(-1);
          setCommand("");
        } else {
          setHistoryIndex(newIndex);
          setCommand(history[newIndex]);
        }
      }
    },
    [handleExecute, history, historyIndex]
  );

  const handleClear = () => {
    setOutput([]);
  };

  const handleCopy = async () => {
    const text = output.map((line) => line.content).join("\n");
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Fallback for older browsers
      const textArea = document.createElement("textarea");
      textArea.value = text;
      textArea.style.position = "fixed";
      textArea.style.left = "-999999px";
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      textArea.remove();
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const getLineClassName = (type: OutputLine["type"]) => {
    switch (type) {
      case "command":
        return "text-blue-400";
      case "stdout":
        return "text-green-400";
      case "stderr":
        return "text-red-400";
      case "info":
        return "text-gray-400 italic";
      default:
        return "text-green-400";
    }
  };

  return (
    <Card className="flex flex-col h-full">
      <CardHeader className="shrink-0">
        <CardTitle className="flex items-center gap-2">
          <TerminalSquare className="h-5 w-5" />
          Terminal
        </CardTitle>
        <CardDescription>
          Execute commands on Wild Central
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col flex-1 min-h-0">
        {/* Output area */}
        <div
          ref={outputRef}
          className="flex-1 min-h-0 bg-slate-950 dark:bg-slate-900 rounded-md p-4 font-mono text-sm overflow-y-auto"
          onClick={() => inputRef.current?.focus()}
        >
          {output.map((line, index) => (
            <div
              key={index}
              className={`whitespace-pre-wrap break-all ${getLineClassName(line.type)}`}
            >
              {line.content}
            </div>
          ))}
          {execMutation.isPending && (
            <div className="flex items-center gap-2 text-gray-400">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>Running...</span>
            </div>
          )}
        </div>

        {/* Input area */}
        <div className="flex items-center gap-2 mt-3">
          <span className="text-green-500 font-mono font-bold">$</span>
          <Input
            ref={inputRef}
            value={command}
            onChange={(e) => setCommand(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Enter command..."
            className="font-mono flex-1"
            disabled={execMutation.isPending}
            autoComplete="off"
            spellCheck={false}
          />
          <Button
            onClick={handleExecute}
            disabled={execMutation.isPending || !command.trim()}
          >
            {execMutation.isPending ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              "Run"
            )}
          </Button>
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 mt-3">
          <Button
            variant="outline"
            size="sm"
            onClick={handleClear}
            disabled={output.length === 0}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            Clear
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleCopy}
            disabled={output.length === 0}
          >
            {copied ? (
              <>
                <Check className="h-4 w-4 mr-1" />
                Copied!
              </>
            ) : (
              <>
                <Copy className="h-4 w-4 mr-1" />
                Copy
              </>
            )}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
