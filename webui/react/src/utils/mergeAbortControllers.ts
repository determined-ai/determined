export const mergeAbortControllers = (...controllers: AbortController[]): AbortController => {
  const mergedController = new AbortController();

  controllers.forEach((c) => {
    // resulting controller is aborted, just ignore the rest
    if (mergedController.signal.aborted) return;
    // preemptively abort if the signal's already aborted
    if (c.signal.aborted) return mergedController.abort(c.signal.reason);

    const abort = () => {
      mergedController.abort(c.signal.reason);
      c.signal.removeEventListener('abort', abort);
    };
    c.signal.addEventListener('abort', abort);
  });

  return mergedController;
};

export default mergeAbortControllers;
