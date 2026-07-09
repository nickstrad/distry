import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import RunResults from "../RunResults.jsx";

describe("RunResults", () => {
  it("renders a passed report with seed stats", () => {
    render(
      <RunResults
        history={[]}
        onSelectSubmission={vi.fn()}
        submission={passedSubmission()}
      />,
    );

    expect(screen.getByText("PASSED")).toBeInTheDocument();
    expect(screen.getByText("Seed 7")).toBeInTheDocument();
    expect(screen.getByText("14 events")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("renders compile output for error submissions", () => {
    render(
      <RunResults
        history={[]}
        onSelectSubmission={vi.fn()}
        submission={compileErrorSubmission()}
      />,
    );

    expect(screen.getByText("ERROR")).toBeInTheDocument();
    expect(screen.getByText(/undefined: SendLater/)).toBeInTheDocument();
  });

  it("expands failed seeds with violations and a filtered highlighted trace", async () => {
    render(
      <RunResults
        history={[]}
        onSelectSubmission={vi.fn()}
        submission={failedSubmission()}
      />,
    );

    expect(screen.getByText("FAILED")).toBeInTheDocument();
    expect(screen.getByText("perfect-link.no-dup")).toBeInTheDocument();
    expect(screen.getByText("at event #3")).toBeInTheDocument();

    const table = screen.getByRole("table");
    expect(within(table).getByText("duplicate delivery")).toBeInTheDocument();

    await userEvent.selectOptions(screen.getByLabelText("Kind"), "drop");
    expect(within(table).getByText("network drop")).toBeInTheDocument();
    expect(
      within(table).queryByText("duplicate delivery"),
    ).not.toBeInTheDocument();
  });
});

function passedSubmission() {
  return {
    id: "sub-pass",
    status: "passed",
    created_at: "2026-07-08T10:00:00Z",
    finished_at: "2026-07-08T10:00:01Z",
    reports: [
      {
        seed: 7,
        passed: true,
        stats: {
          events: 14,
          messagesSent: 4,
          messagesDropped: 0,
          virtualDuration: 12_000_000,
        },
      },
    ],
  };
}

function compileErrorSubmission() {
  return {
    id: "sub-error",
    status: "error",
    compile_output: "./solution.go:12: undefined: SendLater",
    created_at: "2026-07-08T10:00:00Z",
    finished_at: "2026-07-08T10:00:01Z",
    reports: [],
  };
}

function failedSubmission() {
  return {
    id: "sub-fail",
    status: "failed",
    created_at: "2026-07-08T10:00:00Z",
    finished_at: "2026-07-08T10:00:02Z",
    reports: [
      {
        seed: 11,
        passed: false,
        stats: {
          events: 4,
          messagesSent: 2,
          messagesDropped: 1,
          virtualDuration: 1_500_000,
        },
        violations: [
          {
            checker: "perfect-link.no-dup",
            message: "duplicate delivery",
            eventSeq: 3,
          },
        ],
        trace: [
          {
            seq: 1,
            time: 0,
            kind: "send",
            node: 0,
            peer: 1,
            msgType: "DATA",
            detail: "send DATA",
          },
          {
            seq: 2,
            time: 500_000,
            kind: "drop",
            node: 0,
            peer: 1,
            msgType: "DATA",
            detail: "network drop",
          },
          {
            seq: 3,
            time: 1_500_000,
            kind: "deliver",
            node: 1,
            peer: 0,
            msgType: "DATA",
            detail: "duplicate delivery",
          },
        ],
      },
    ],
  };
}
