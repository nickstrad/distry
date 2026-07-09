import { useCallback, useEffect, useState } from "react";
import { ApiError, api } from "../api.js";

const TERMINAL_STATUSES = new Set(["passed", "failed", "error"]);
const POLL_MS = 1500;

export function useRunResults(slug, solution) {
  const [submission, setSubmission] = useState(null);
  const [history, setHistory] = useState([]);
  const [runningID, setRunningID] = useState("");
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [loadingSubmission, setLoadingSubmission] = useState(false);
  const [error, setError] = useState("");

  const refreshHistory = useCallback(async () => {
    if (!slug) return;
    setLoadingHistory(true);
    try {
      const nextHistory = await api(`/api/problems/${slug}/submissions`);
      setHistory(nextHistory);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoadingHistory(false);
    }
  }, [slug]);

  const loadSubmission = useCallback(
    async (id) => {
      if (!id) return null;
      setLoadingSubmission(true);
      setError("");
      try {
        const nextSubmission = await api(`/api/submissions/${id}`);
        setSubmission(nextSubmission);
        if (isTerminal(nextSubmission.status)) {
          setRunningID("");
          await refreshHistory();
        }
        return nextSubmission;
      } catch (err) {
        setError(err.message);
        if (err instanceof ApiError && err.status === 404) {
          setRunningID("");
        }
        return null;
      } finally {
        setLoadingSubmission(false);
      }
    },
    [refreshHistory],
  );

  useEffect(() => {
    setSubmission(null);
    setHistory([]);
    setRunningID("");
    setError("");
    refreshHistory();
  }, [refreshHistory]);

  useEffect(() => {
    if (!runningID) return undefined;
    const intervalID = window.setInterval(async () => {
      const nextSubmission = await loadSubmission(runningID);
      if (nextSubmission && isTerminal(nextSubmission.status)) {
        window.clearInterval(intervalID);
      }
    }, POLL_MS);
    return () => window.clearInterval(intervalID);
  }, [loadSubmission, runningID]);

  const run = useCallback(
    async (seeds = []) => {
      if (!slug || solution.busy || runningID) return;
      setError("");
      try {
        if (solution.dirty) {
          await solution.save();
        }
        const started = await api(`/api/problems/${slug}/run`, {
          method: "POST",
          ...jsonBody({ seeds }, seeds.length > 0),
        });
        setRunningID(started.submissionID);
        await loadSubmission(started.submissionID);
        await refreshHistory();
      } catch (err) {
        setError(conflictMessage(err));
      }
    },
    [loadSubmission, refreshHistory, runningID, slug, solution],
  );

  const replay = useCallback(async (submissionID, seed) => {
    setError("");
    try {
      const report = await api(`/api/submissions/${submissionID}/replay`, {
        method: "POST",
        ...jsonBody({ seed }),
      });
      return report;
    } catch (err) {
      setError(err.message);
      return null;
    }
  }, []);

  return {
    error,
    history,
    loadingHistory,
    loadingSubmission,
    running: Boolean(runningID),
    run,
    replay,
    selectSubmission: loadSubmission,
    submission,
  };
}

function jsonBody(value, include = true) {
  if (!include) return {};
  return {
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(value),
  };
}

export function isTerminal(status) {
  return TERMINAL_STATUSES.has(status);
}

function conflictMessage(err) {
  if (err instanceof ApiError && err.status === 409) {
    return "A run is already in flight for this problem.";
  }
  return err.message;
}
