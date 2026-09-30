export const getErrorMessage = (err: unknown, fallback = 'Request failed'): string => {
  if (err instanceof Error && err.message) return err.message;
  if (typeof err === 'object' && err) {
    const response = (err as { response?: { data?: unknown } }).response;
    const data = response?.data as { error?: unknown; message?: unknown } | undefined;
    if (data) {
      if (typeof data.error === 'string') return data.error;
      if (typeof data.message === 'string') return data.message;
    }
    const message = (err as { message?: unknown }).message;
    if (typeof message === 'string') return message;
  }
  return fallback;
};

export const getErrorStatus = (err: unknown): number | undefined => {
  if (typeof err === 'object' && err) {
    const responseStatus = (err as { response?: { status?: unknown } }).response?.status;
    if (typeof responseStatus === 'number') return responseStatus;
    const status = (err as { status?: unknown }).status;
    if (typeof status === 'number') return status;
  }
  return undefined;
};
