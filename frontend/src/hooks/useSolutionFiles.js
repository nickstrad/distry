import { useCallback, useEffect, useMemo, useState } from "react";
import { api, apiMaybe } from "../api.js";

export function useSolutionFiles(problem) {
  const slug = problem?.slug;
  const templateFiles = useMemo(() => problem?.templates || {}, [problem]);
  const fileNames = useMemo(
    () => Object.keys(templateFiles).sort(),
    [templateFiles],
  );
  const [files, setFiles] = useState({});
  const [savedFiles, setSavedFiles] = useState({});
  const [activeFile, setActiveFile] = useState("");
  const [loading, setLoading] = useState(false);
  const [loadedSlug, setLoadedSlug] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const effectiveActiveFile = activeFile || fileNames[0] || "";

  const dirty = useMemo(
    () => !sameFiles(files, savedFiles),
    [files, savedFiles],
  );
  const busy = loading || saving;

  useEffect(() => {
    if (!slug) return undefined;

    let active = true;
    setLoading(true);
    setLoadedSlug("");
    setError("");
    setActiveFile(fileNames[0] || "");

    apiMaybe(solutionPath(slug))
      .then((solution) => {
        if (!active) return;
        replaceFiles(setFiles, setSavedFiles, solution?.files || templateFiles);
        setLoadedSlug(slug);
      })
      .catch((err) => {
        if (active) setError(err.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [slug, fileNames, templateFiles]);

  const persist = useCallback(
    async (nextFiles, { updateEditorFirst = false } = {}) => {
      if (!slug) return;
      if (updateEditorFirst) setFiles(nextFiles);
      setSaving(true);
      setError("");
      try {
        const solution = await api(solutionPath(slug), {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ files: nextFiles }),
        });
        replaceFiles(setFiles, setSavedFiles, solution.files);
        return solution;
      } catch (err) {
        setError(err.message);
        throw err;
      } finally {
        setSaving(false);
      }
    },
    [slug],
  );

  const save = useCallback(() => persist(files), [files, persist]);
  const reset = useCallback(
    () => persist(templateFiles, { updateEditorFirst: true }),
    [persist, templateFiles],
  );

  useEffect(() => {
    function beforeUnload(event) {
      if (!dirty) return;
      event.preventDefault();
      event.returnValue = "";
    }
    window.addEventListener("beforeunload", beforeUnload);
    return () => window.removeEventListener("beforeunload", beforeUnload);
  }, [dirty]);

  useEffect(() => {
    function onKeyDown(event) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
        event.preventDefault();
        if (dirty && !busy) {
          save();
        }
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [dirty, busy, save]);

  function updateActiveFile(content) {
    setFiles((current) => ({ ...current, [effectiveActiveFile]: content }));
  }

  return {
    busy,
    dirty,
    error,
    files,
    fileNames,
    activeFile: effectiveActiveFile,
    activeContent: files[effectiveActiveFile] || "",
    loading: loading || Boolean(slug && loadedSlug !== slug),
    reset,
    save,
    saving,
    setActiveFile,
    updateActiveFile,
  };
}

function solutionPath(slug) {
  return `/api/problems/${slug}/solution`;
}

function replaceFiles(setFiles, setSavedFiles, nextFiles) {
  setFiles(nextFiles);
  setSavedFiles(nextFiles);
}

function sameFiles(a, b) {
  const aKeys = Object.keys(a);
  const bKeys = Object.keys(b);
  return (
    aKeys.length === bKeys.length && aKeys.every((key) => a[key] === b[key])
  );
}
