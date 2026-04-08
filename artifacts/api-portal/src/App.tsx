import { useState, useEffect, useCallback } from "react";

const DARK_BG = "hsl(222, 47%, 11%)";
const CARD_BG = "hsl(222, 47%, 15%)";
const BORDER = "hsl(222, 47%, 22%)";
const TEXT = "hsl(210, 40%, 96%)";
const MUTED = "hsl(215, 16%, 57%)";
const OPENAI_BLUE = "#3B82F6";
const ANTHROPIC_ORANGE = "#F97316";
const SUCCESS_GREEN = "#22C55E";

const models = [
  { id: "gpt-5.2", provider: "OpenAI" },
  { id: "gpt-5-mini", provider: "OpenAI" },
  { id: "gpt-5-nano", provider: "OpenAI" },
  { id: "o4-mini", provider: "OpenAI" },
  { id: "o3", provider: "OpenAI" },
  { id: "claude-opus-4-6", provider: "Anthropic" },
  { id: "claude-sonnet-4-6", provider: "Anthropic" },
  { id: "claude-haiku-4-5", provider: "Anthropic" },
];

const endpoints = [
  {
    method: "GET",
    path: "/v1/models",
    type: "Both",
    description: "List all available models from OpenAI and Anthropic",
  },
  {
    method: "POST",
    path: "/v1/chat/completions",
    type: "OpenAI",
    description:
      "OpenAI-compatible chat completions. Supports streaming, tool calls, and both OpenAI and Anthropic models.",
  },
  {
    method: "POST",
    path: "/v1/messages",
    type: "Anthropic",
    description:
      "Anthropic Messages API native format. Supports streaming, tool calls, and both Claude and OpenAI models.",
  },
];

const steps = [
  {
    n: 1,
    title: "Add Provider",
    desc: 'In CherryStudio, go to Settings → Model Providers → click "+" to add a new provider.',
  },
  {
    n: 2,
    title: "Choose Format",
    desc: 'Select "OpenAI" as the provider type for /v1/chat/completions, or "Anthropic" for /v1/messages native format.',
  },
  {
    n: 3,
    title: "Enter Base URL & Key",
    desc: "Set the API Base URL to your deployment domain (e.g. https://your-app.replit.app) and paste your PROXY_API_KEY as the API Key.",
  },
  {
    n: 4,
    title: "Start Chatting",
    desc: "Pick any model from the list above. All requests are proxied via Replit AI Integrations — no personal API keys needed.",
  },
];

function copyToClipboard(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text);
  }
  return new Promise((resolve) => {
    const el = document.createElement("textarea");
    el.value = text;
    el.style.position = "fixed";
    el.style.opacity = "0";
    document.body.appendChild(el);
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
    resolve();
  });
}

function CopyButton({ text, style }: { text: string; style?: React.CSSProperties }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(async () => {
    await copyToClipboard(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }, [text]);

  return (
    <button
      onClick={handleCopy}
      style={{
        padding: "4px 10px",
        background: copied ? "rgba(34,197,94,0.15)" : "rgba(255,255,255,0.07)",
        border: `1px solid ${copied ? "rgba(34,197,94,0.4)" : BORDER}`,
        borderRadius: 6,
        color: copied ? SUCCESS_GREEN : MUTED,
        fontSize: 12,
        cursor: "pointer",
        transition: "all 0.15s",
        whiteSpace: "nowrap",
        ...style,
      }}
    >
      {copied ? "Copied!" : "Copy"}
    </button>
  );
}

function MethodBadge({ method }: { method: string }) {
  const bg = method === "GET" ? "rgba(34,197,94,0.15)" : "rgba(168,85,247,0.15)";
  const color = method === "GET" ? SUCCESS_GREEN : "#A855F7";
  const border = method === "GET" ? "rgba(34,197,94,0.3)" : "rgba(168,85,247,0.3)";
  return (
    <span
      style={{
        padding: "2px 8px",
        background: bg,
        border: `1px solid ${border}`,
        borderRadius: 4,
        color,
        fontSize: 11,
        fontWeight: 700,
        letterSpacing: "0.05em",
        fontFamily: "monospace",
      }}
    >
      {method}
    </span>
  );
}

function TypeBadge({ type }: { type: string }) {
  const map: Record<string, { bg: string; color: string; border: string }> = {
    OpenAI: { bg: "rgba(59,130,246,0.12)", color: OPENAI_BLUE, border: "rgba(59,130,246,0.3)" },
    Anthropic: { bg: "rgba(249,115,22,0.12)", color: ANTHROPIC_ORANGE, border: "rgba(249,115,22,0.3)" },
    Both: { bg: "rgba(100,116,139,0.15)", color: "#94A3B8", border: "rgba(100,116,139,0.3)" },
  };
  const s = map[type] ?? map["Both"];
  return (
    <span
      style={{
        padding: "2px 8px",
        background: s.bg,
        border: `1px solid ${s.border}`,
        borderRadius: 4,
        color: s.color,
        fontSize: 11,
        fontWeight: 600,
        letterSpacing: "0.03em",
      }}
    >
      {type}
    </span>
  );
}

function ProviderBadge({ provider }: { provider: string }) {
  const isOpenAI = provider === "OpenAI";
  return (
    <span
      style={{
        padding: "1px 7px",
        background: isOpenAI ? "rgba(59,130,246,0.10)" : "rgba(249,115,22,0.10)",
        border: `1px solid ${isOpenAI ? "rgba(59,130,246,0.25)" : "rgba(249,115,22,0.25)"}`,
        borderRadius: 4,
        color: isOpenAI ? OPENAI_BLUE : ANTHROPIC_ORANGE,
        fontSize: 10,
        fontWeight: 600,
        letterSpacing: "0.04em",
      }}
    >
      {provider}
    </span>
  );
}

export default function App() {
  const [online, setOnline] = useState<boolean | null>(null);
  const baseUrl = window.location.origin;

  useEffect(() => {
    fetch("/api/healthz")
      .then((r) => setOnline(r.ok))
      .catch(() => setOnline(false));
  }, []);

  const curlExample = `curl ${baseUrl}/v1/chat/completions \\
  -H "Authorization: Bearer YOUR_PROXY_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.2",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ],
    "stream": false
  }'`;

  const card: React.CSSProperties = {
    background: CARD_BG,
    border: `1px solid ${BORDER}`,
    borderRadius: 12,
    padding: "20px 24px",
    marginBottom: 16,
  };

  const sectionTitle: React.CSSProperties = {
    color: TEXT,
    fontSize: 15,
    fontWeight: 700,
    marginBottom: 14,
    display: "flex",
    alignItems: "center",
    gap: 8,
  };

  return (
    <div
      style={{
        minHeight: "100vh",
        background: DARK_BG,
        color: TEXT,
        fontFamily:
          '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
        padding: "0 0 48px",
      }}
    >
      {/* Header */}
      <div
        style={{
          borderBottom: `1px solid ${BORDER}`,
          background: "rgba(15,23,42,0.7)",
          backdropFilter: "blur(12px)",
          padding: "18px 24px",
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: 12,
          position: "sticky",
          top: 0,
          zIndex: 10,
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          <div
            style={{
              width: 36,
              height: 36,
              borderRadius: 8,
              background: "linear-gradient(135deg, #3B82F6 0%, #F97316 100%)",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              fontSize: 18,
              fontWeight: 900,
              color: "#fff",
              letterSpacing: "-1px",
            }}
          >
            AI
          </div>
          <div>
            <div style={{ fontWeight: 700, fontSize: 15, color: TEXT }}>Free AI Proxy</div>
            <div style={{ fontSize: 12, color: MUTED }}>OpenAI + Anthropic via Replit AI Integrations</div>
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <div
            style={{
              width: 8,
              height: 8,
              borderRadius: "50%",
              background:
                online === null ? "#64748B" : online ? SUCCESS_GREEN : "#EF4444",
              boxShadow: online
                ? `0 0 0 3px rgba(34,197,94,0.2)`
                : online === false
                  ? "0 0 0 3px rgba(239,68,68,0.2)"
                  : "none",
              transition: "all 0.3s",
            }}
          />
          <span style={{ fontSize: 12, color: MUTED }}>
            {online === null ? "Checking..." : online ? "Online" : "Offline"}
          </span>
        </div>
      </div>

      <div style={{ maxWidth: 800, margin: "0 auto", padding: "28px 16px 0" }}>
        {/* Connection Details */}
        <div style={card}>
          <div style={sectionTitle}>
            <span style={{ opacity: 0.6 }}>&#x1F517;</span> Connection Details
          </div>
          {[
            { label: "Base URL", value: baseUrl },
            { label: "Authorization Header", value: "Bearer YOUR_PROXY_API_KEY" },
          ].map((row, i) => (
            <div
              key={i}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                gap: 12,
                padding: "10px 0",
                borderBottom: i === 0 ? `1px solid ${BORDER}` : "none",
                flexWrap: "wrap",
              }}
            >
              <span style={{ color: MUTED, fontSize: 13, minWidth: 160 }}>{row.label}</span>
              <div style={{ display: "flex", alignItems: "center", gap: 8, flex: 1, minWidth: 0 }}>
                <code
                  style={{
                    background: "rgba(0,0,0,0.3)",
                    border: `1px solid ${BORDER}`,
                    borderRadius: 6,
                    padding: "4px 10px",
                    fontSize: 12,
                    color: TEXT,
                    fontFamily: "monospace",
                    flex: 1,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                >
                  {row.value}
                </code>
                <CopyButton text={row.value} />
              </div>
            </div>
          ))}
          <div
            style={{
              marginTop: 14,
              padding: "10px 14px",
              background: "rgba(249,115,22,0.08)",
              border: "1px solid rgba(249,115,22,0.25)",
              borderRadius: 8,
              fontSize: 12,
              color: "#FCD34D",
              lineHeight: 1.5,
            }}
          >
            <strong>Important:</strong> You must set <code style={{ background: "rgba(0,0,0,0.3)", padding: "0 4px", borderRadius: 3 }}>PROXY_API_KEY</code> as an environment secret before using this proxy. Without it, all requests will return 401 Unauthorized.
          </div>
        </div>

        {/* API Endpoints */}
        <div style={card}>
          <div style={sectionTitle}>
            <span style={{ opacity: 0.6 }}>&#x26A1;</span> API Endpoints
          </div>
          {endpoints.map((ep, i) => (
            <div
              key={i}
              style={{
                borderBottom: i < endpoints.length - 1 ? `1px solid ${BORDER}` : "none",
                padding: "12px 0",
              }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6, flexWrap: "wrap" }}>
                <MethodBadge method={ep.method} />
                <TypeBadge type={ep.type} />
                <div style={{ display: "flex", alignItems: "center", gap: 6, flex: 1, minWidth: 0 }}>
                  <code
                    style={{
                      background: "rgba(0,0,0,0.3)",
                      border: `1px solid ${BORDER}`,
                      borderRadius: 6,
                      padding: "3px 8px",
                      fontSize: 12,
                      color: TEXT,
                      fontFamily: "monospace",
                    }}
                  >
                    {baseUrl}{ep.path}
                  </code>
                  <CopyButton text={`${baseUrl}${ep.path}`} />
                </div>
              </div>
              <p style={{ color: MUTED, fontSize: 12, margin: 0, lineHeight: 1.5 }}>{ep.description}</p>
            </div>
          ))}
        </div>

        {/* Available Models */}
        <div style={card}>
          <div style={sectionTitle}>
            <span style={{ opacity: 0.6 }}>&#x1F916;</span> Available Models
          </div>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fill, minmax(200px, 1fr))",
              gap: 8,
            }}
          >
            {models.map((m) => (
              <div
                key={m.id}
                style={{
                  display: "flex",
                  alignItems: "center",
                  justifyContent: "space-between",
                  gap: 8,
                  padding: "8px 12px",
                  background: "rgba(0,0,0,0.2)",
                  border: `1px solid ${BORDER}`,
                  borderRadius: 8,
                }}
              >
                <code style={{ fontSize: 12, color: TEXT, fontFamily: "monospace" }}>{m.id}</code>
                <ProviderBadge provider={m.provider} />
              </div>
            ))}
          </div>
        </div>

        {/* CherryStudio Setup */}
        <div style={card}>
          <div style={sectionTitle}>
            <span style={{ opacity: 0.6 }}>&#x1F4D6;</span> CherryStudio Setup Guide
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            {steps.map((step) => (
              <div
                key={step.n}
                style={{
                  display: "flex",
                  gap: 14,
                  padding: "12px 14px",
                  background: "rgba(0,0,0,0.15)",
                  border: `1px solid ${BORDER}`,
                  borderRadius: 8,
                  alignItems: "flex-start",
                }}
              >
                <div
                  style={{
                    minWidth: 28,
                    height: 28,
                    borderRadius: "50%",
                    background: "linear-gradient(135deg, #3B82F6, #6366F1)",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    fontSize: 13,
                    fontWeight: 700,
                    color: "#fff",
                    flexShrink: 0,
                  }}
                >
                  {step.n}
                </div>
                <div>
                  <div style={{ fontWeight: 600, fontSize: 13, color: TEXT, marginBottom: 4 }}>{step.title}</div>
                  <p style={{ color: MUTED, fontSize: 12, margin: 0, lineHeight: 1.5 }}>{step.desc}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Quick Test */}
        <div style={card}>
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
            <div style={sectionTitle}>
              <span style={{ opacity: 0.6 }}>&#x1F4BB;</span> Quick Test (curl)
            </div>
            <CopyButton text={curlExample} />
          </div>
          <pre
            style={{
              background: "rgba(0,0,0,0.4)",
              border: `1px solid ${BORDER}`,
              borderRadius: 8,
              padding: "14px 16px",
              fontSize: 12,
              fontFamily: "monospace",
              color: TEXT,
              overflowX: "auto",
              margin: 0,
              lineHeight: 1.6,
              whiteSpace: "pre",
            }}
          >
            {curlExample.split("\n").map((line, i) => {
              if (line.startsWith("curl ")) {
                return (
                  <span key={i}>
                    <span style={{ color: ANTHROPIC_ORANGE }}>curl </span>
                    <span style={{ color: "#A5F3FC" }}>{line.slice(5)}</span>
                    {"\n"}
                  </span>
                );
              }
              if (line.trim().startsWith("-H")) {
                const [flag, ...rest] = line.trim().split(" ");
                return (
                  <span key={i}>
                    {"  "}
                    <span style={{ color: OPENAI_BLUE }}>{flag}</span>{" "}
                    <span style={{ color: "#86EFAC" }}>{rest.join(" ")}</span>
                    {"\n"}
                  </span>
                );
              }
              if (line.trim().startsWith("-d")) {
                return (
                  <span key={i}>
                    {"  "}
                    <span style={{ color: OPENAI_BLUE }}>-d</span>
                    {" '"}
                    {"\n"}
                  </span>
                );
              }
              return <span key={i}>{line}{"\n"}</span>;
            })}
          </pre>
        </div>

        {/* Footer */}
        <div style={{ textAlign: "center", color: MUTED, fontSize: 12, marginTop: 8 }}>
          Powered by Replit AI Integrations &mdash; no personal API keys required.
          Billed to your Replit credits.
        </div>
      </div>
    </div>
  );
}
