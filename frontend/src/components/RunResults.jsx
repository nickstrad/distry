import { useEffect, useMemo, useRef, useState } from "react";
import { Button } from "./ui/button";

const STATUS_LABELS = {
  queued: "Queued",
  compiling: "Compiling",
  running: "Running",
  passed: "PASSED",
  failed: "FAILED",
  error: "ERROR",
};

const SEED_STATS = [
  ["events", (stats) => `${stats.events ?? 0} events`],
  ["sent", (stats) => `${stats.messagesSent ?? 0} sent`],
  ["dropped", (stats) => `${stats.messagesDropped ?? 0} dropped`],
  ["duration", (stats) => formatDuration(stats.virtualDuration)],
];

const TRACE_COLUMNS = [
  ["seq", "Seq", (event) => event.seq],
  ["time", "Time", (event) => formatDuration(event.time)],
  ["kind", "Kind", (event) => <span className="kind-pill">{event.kind}</span>],
  ["node", "Node", (event) => formatNode(event.node)],
  ["peer", "Peer", (event) => formatNode(event.peer)],
  ["msgType", "Msg type", (event) => event.msgType || "-"],
  ["detail", "Detail", (event) => event.detail || "-"],
];

export default function RunResults({
  error,
  history,
  loadingHistory,
  loadingSubmission,
  onSelectSubmission,
  onReplay,
  submission,
}) {
  return (
    <section className="results-panel" aria-label="Run results">
      <div className="results-main">
        <StatusLine loading={loadingSubmission} submission={submission} />
        {error && <p className="error results-error">{error}</p>}
        {!submission && !error && (
          <div className="empty-results">
            <strong>No run yet</strong>
            <span>
              Run the saved solution to see seed outcomes and traces here.
            </span>
          </div>
        )}
        {submission?.compile_output && (
          <pre
            className={
              submission.status === "error"
                ? "compile-output error-output"
                : "compile-output"
            }
          >
            <code>{submission.compile_output}</code>
          </pre>
        )}
        {submission?.reports?.length > 0 && (
          <SeedResults
            onReplay={onReplay}
            reports={submission.reports}
            submissionID={submission.id}
          />
        )}
      </div>
      <SubmissionHistory
        history={history}
        loading={loadingHistory}
        selectedID={submission?.id}
        onSelect={onSelectSubmission}
      />
    </section>
  );
}

function StatusLine({ loading, submission }) {
  const status = submission?.status;
  if (!submission) {
    return (
      <div className="status-line idle">
        <span className="status-dot" />
        Ready
      </div>
    );
  }
  return (
    <div className={`status-line status-${status}`}>
      <span className="status-dot" />
      <strong>{statusLabel(status)}</strong>
      <span>{submissionTiming(submission)}</span>
      {loading && <span>Refreshing...</span>}
    </div>
  );
}

function SeedResults({ onReplay, reports, submissionID }) {
  const firstFailed = reports.find((report) => !report.passed)?.seed;
  const [expandedSeed, setExpandedSeed] = useState(
    firstFailed ?? reports[0]?.seed,
  );

  useEffect(() => {
    setExpandedSeed(firstFailed ?? reports[0]?.seed);
  }, [firstFailed, reports]);

  return (
    <div className="seed-results">
      {reports.map((report) => (
        <SeedResult
          expanded={expandedSeed === report.seed}
          key={report.seed}
          onToggle={() =>
            setExpandedSeed(expandedSeed === report.seed ? null : report.seed)
          }
          onReplay={onReplay}
          report={report}
          submissionID={submissionID}
        />
      ))}
    </div>
  );
}

function SeedResult({ expanded, onReplay, onToggle, report, submissionID }) {
  const failed = !report.passed;
  const stats = report.stats || {};
  const [replay, setReplay] = useState(null);
  const [replaying, setReplaying] = useState(false);
  async function replaySeed() {
    if (!onReplay || !submissionID) return;
    setReplaying(true);
    try {
      setReplay(await onReplay(submissionID, report.seed));
    } finally {
      setReplaying(false);
    }
  }
  return (
    <article className={failed ? "seed-row failed" : "seed-row"}>
      <button
        className="seed-summary"
        type="button"
        onClick={onToggle}
        aria-expanded={expanded}
      >
        <span
          className={
            failed ? "seed-result mark-failed" : "seed-result mark-passed"
          }
        >
          {failed ? "✗" : "✓"}
        </span>
        <strong>Seed {report.seed}</strong>
        {SEED_STATS.map(([key, label]) => (
          <span key={key}>{label(stats)}</span>
        ))}
      </button>
      {expanded && failed && (
        <div className="seed-detail">
          <Button
            className="replay-button"
            type="button"
            variant="outline"
            onClick={replaySeed}
            disabled={replaying}
          >
            {replaying ? "Replaying..." : "Replay"}
          </Button>
          <ViolationList violations={report.violations || []} />
          <TraceViewer report={report} />
          <ReplayResult report={replay} />
        </div>
      )}
    </article>
  );
}

function ReplayResult({ report }) {
  if (!report) return null;
  return (
    <div className="replay-result">
      <strong>Replay</strong>
      <ViolationList violations={report.violations || []} />
      <TraceViewer report={report} />
    </div>
  );
}

function ViolationList({ violations }) {
  if (!violations.length) return null;
  return (
    <div className="violation-list">
      {violations.map((violation, index) => (
        <div
          className="violation-card"
          key={`${violation.checker}-${violation.eventSeq}-${index}`}
        >
          <div>
            <strong>{violation.checker}</strong>
            <p>{violation.message}</p>
          </div>
          <span>at event #{violation.eventSeq}</span>
        </div>
      ))}
    </div>
  );
}

function TraceViewer({ report }) {
  const trace = report.trace || [];
  const [node, setNode] = useState("all");
  const [kind, setKind] = useState("all");
  const highlightSeq = firstViolationSeq(report);
  const highlightRef = useRef(null);
  const nodes = useMemo(
    () =>
      unique(
        trace.map((event) => event.node).filter((value) => value !== undefined),
      ),
    [trace],
  );
  const kinds = useMemo(
    () => unique(trace.map((event) => event.kind)),
    [trace],
  );
  const filteredTrace = useMemo(
    () =>
      trace.filter(
        (event) =>
          (node === "all" || String(event.node) === node) &&
          (kind === "all" || event.kind === kind),
      ),
    [kind, node, trace],
  );

  useEffect(() => {
    highlightRef.current?.scrollIntoView?.({ block: "center" });
  }, [filteredTrace, highlightSeq]);

  if (!trace.length) {
    return <p className="trace-empty">No trace captured for this seed.</p>;
  }

  return (
    <div className="trace-viewer">
      {report.truncated && (
        <p className="trace-truncated">
          Trace truncated to the first {trace.length} events.
        </p>
      )}
      <div className="trace-filters">
        <TraceFilter
          label="Node"
          options={nodes}
          value={node}
          onChange={setNode}
          formatOption={(item) => `n${item}`}
        />
        <TraceFilter
          label="Kind"
          options={kinds}
          value={kind}
          onChange={setKind}
        />
      </div>
      <div className="trace-table-wrap">
        <table className="trace-table">
          <thead>
            <tr>
              {TRACE_COLUMNS.map(([key, label]) => (
                <th key={key}>{label}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filteredTrace.map((event) => {
              const highlighted = Number(event.seq) === highlightSeq;
              return (
                <tr
                  className={
                    highlighted
                      ? `trace-${event.kind} highlighted`
                      : `trace-${event.kind}`
                  }
                  key={event.seq}
                  ref={highlighted ? highlightRef : null}
                >
                  {TRACE_COLUMNS.map(([key, , render]) => (
                    <td key={key}>{render(event)}</td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function TraceFilter({
  label,
  options,
  value,
  onChange,
  formatOption = String,
}) {
  return (
    <label>
      {label}
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="all">All</option>
        {options.map((item) => (
          <option key={item} value={item}>
            {formatOption(item)}
          </option>
        ))}
      </select>
    </label>
  );
}

function SubmissionHistory({ history, loading, selectedID, onSelect }) {
  return (
    <aside className="submission-history" aria-label="Submission history">
      <div className="history-heading">
        <strong>History</strong>
        {loading && <span>Loading...</span>}
      </div>
      {!history.length && !loading && <p>No submissions</p>}
      {history.map((item) => (
        <Button
          className={
            item.id === selectedID ? "history-item active" : "history-item"
          }
          key={item.id}
          type="button"
          variant="ghost"
          onClick={() => onSelect(item.id)}
        >
          <span>{statusLabel(item.status)}</span>
          <small>{historySeeds(item)}</small>
        </Button>
      ))}
    </aside>
  );
}

function firstViolationSeq(report) {
  const seq = report.violations?.find(
    (violation) => violation.eventSeq >= 0,
  )?.eventSeq;
  return seq === undefined ? null : Number(seq);
}

function formatDuration(value) {
  const ns = Number(value || 0);
  if (ns >= 1_000_000_000) return `${trim(ns / 1_000_000_000)}s`;
  if (ns >= 1_000_000) return `${trim(ns / 1_000_000)}ms`;
  if (ns >= 1_000) return `${trim(ns / 1_000)}us`;
  return `${ns}ns`;
}

function statusLabel(status) {
  return STATUS_LABELS[status] || status;
}

function trim(value) {
  return Number.isInteger(value) ? value : value.toFixed(2);
}

function formatNode(value) {
  return value === undefined || value === null ? "-" : `n${value}`;
}

function historySeeds(submission) {
  const seeds = submission.reports?.map((report) => report.seed);
  return seeds?.length ? `seeds ${seeds.join(", ")}` : "seeds pending";
}

function submissionTiming(submission) {
  if (!submission.created_at) return "";
  if (!submission.finished_at) return "in progress";
  const ms =
    new Date(submission.finished_at).getTime() -
    new Date(submission.created_at).getTime();
  if (!Number.isFinite(ms) || ms < 0) return "";
  return `${trim(ms / 1000)}s`;
}

function unique(values) {
  return [...new Set(values)].sort((a, b) =>
    String(a).localeCompare(String(b), undefined, { numeric: true }),
  );
}
