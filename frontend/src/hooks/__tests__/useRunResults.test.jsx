import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useRunResults } from "../useRunResults.js";

afterEach(() => {
  vi.restoreAllMocks();
  vi.useRealTimers();
});

describe("useRunResults", () => {
  it("autosaves, starts a run, polls until terminal, and stops polling", async () => {
    vi.useFakeTimers();
    const save = vi.fn().mockResolvedValue({});
    const fetch = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(jsonResponse([]))
      .mockResolvedValueOnce(
        jsonResponse({ submissionID: "sub-1" }, { status: 202 }),
      )
      .mockResolvedValueOnce(
        jsonResponse({ id: "sub-1", status: "queued", reports: [] }),
      )
      .mockResolvedValueOnce(jsonResponse([]))
      .mockResolvedValueOnce(
        jsonResponse({ id: "sub-1", status: "running", reports: [] }),
      )
      .mockResolvedValueOnce(
        jsonResponse({
          id: "sub-1",
          status: "passed",
          reports: [{ seed: 1, passed: true }],
        }),
      )
      .mockResolvedValueOnce(
        jsonResponse([
          {
            id: "sub-1",
            status: "passed",
            reports: [{ seed: 1, passed: true }],
          },
        ]),
      );

    render(<Harness save={save} />);
    await act(flushPromises);
    expect(fetch).toHaveBeenCalledWith(
      "/api/problems/perfect-link/submissions",
      undefined,
    );

    fireEvent.click(screen.getByRole("button", { name: "run" }));
    await act(flushPromises);

    expect(save).toHaveBeenCalledTimes(1);
    expect(fetch).toHaveBeenCalledWith("/api/problems/perfect-link/run", {
      method: "POST",
    });
    expect(screen.getByText("queued")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1500);
    });
    expect(screen.getByText("running")).toBeInTheDocument();

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1500);
    });
    expect(screen.getByText("passed")).toBeInTheDocument();

    const callsAtTerminal = fetch.mock.calls.length;
    await act(async () => {
      await vi.advanceTimersByTimeAsync(4500);
    });
    expect(fetch.mock.calls).toHaveLength(callsAtTerminal);
  });
});

function Harness({ save }) {
  const runs = useRunResults("perfect-link", {
    busy: false,
    dirty: true,
    save,
  });
  return (
    <>
      <button type="button" onClick={runs.run}>
        run
      </button>
      <output>{runs.submission?.status || "none"}</output>
    </>
  );
}

function jsonResponse(body, { ok = true, status = 200 } = {}) {
  return {
    ok,
    status,
    json: async () => body,
  };
}

function flushPromises() {
  return Promise.resolve();
}
