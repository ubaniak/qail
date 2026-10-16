// Generic data hooks ported from qail_ui — callService for reads,
// mutateService for writes. The pattern keeps every page's data flow
// uniform: every read returns { loading, error, result, refresh }, every
// write returns a fire-and-forget shape that invokes onSuccess/onError.

import { useCallback, useEffect, useRef, useState } from "react";

export type CallServiceProps<T> = {
  call: () => Promise<T>;
};

export type CallServiceResult<T> = {
  loading: boolean;
  error?: Error;
  result: T;
  refresh: () => void;
};

export const callService = <T,>(
  props: CallServiceProps<T>,
  initial: T,
  depsKey?: unknown
): CallServiceResult<T> => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [result, setResult] = useState<T>(initial);
  const [refreshKey, setRefreshKey] = useState(0);
  const depsSig = depsKey === undefined ? "" : JSON.stringify(depsKey);
  // JSON of the last result, so a refresh that returns the same data
  // doesn't hand React a new object and re-render every page.
  const lastJson = useRef<string | undefined>(undefined);

  useEffect(() => {
    let cancelled = false;
    // Only the first load (and a deps change) shows as loading; a refresh
    // keeps the current list on screen and swaps it when the new one
    // lands, so it costs one render instead of three.
    if (refreshKey === 0) setLoading(true);
    props
      .call()
      .then((res) => {
        if (cancelled) return;
        const json = JSON.stringify(res);
        if (json !== lastJson.current) {
          lastJson.current = json;
          setResult(res);
        }
        setError(undefined);
        setLoading(false);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err as Error);
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshKey, depsSig]);

  const refresh = useCallback(() => setRefreshKey((k) => k + 1), []);
  return { loading, error, result, refresh };
};

export type MutateServiceProps = {
  call: () => Promise<void>;
  onSuccess?: () => void;
  onError?: (error: Error) => void;
};

export const mutateService = (props: MutateServiceProps): void => {
  (async () => {
    try {
      await props.call();
      props.onSuccess?.();
    } catch (err) {
      props.onError?.(err as Error);
    }
  })();
};
