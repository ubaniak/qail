// Generic data hooks ported from qail_ui — callService for reads,
// mutateService for writes. The pattern keeps every page's data flow
// uniform: every read returns { loading, error, result, refresh }, every
// write returns a fire-and-forget shape that invokes onSuccess/onError.

import { useEffect, useState } from "react";

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
  initial: T
): CallServiceResult<T> => {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [result, setResult] = useState<T>(initial);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(undefined);
    props
      .call()
      .then((res) => {
        if (!cancelled) setResult(res);
      })
      .catch((err) => {
        if (!cancelled) setError(err as Error);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshKey]);

  return {
    loading,
    error,
    result,
    refresh: () => setRefreshKey((k) => k + 1),
  };
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
