import { useEffect } from 'react';

const defaultGlobalErrorHandler = () => true;

const useErrorBoundary = (handleGlobalError?: (e: unknown) => boolean): void => {
  useEffect(() => {
    /**
     * When `true` is returned, prevents the firing of the default event handler.
     * https://developer.mozilla.org/en-US/docs/Web/API/GlobalEventHandlers/onerror
     */
    window.onerror = handleGlobalError ?? defaultGlobalErrorHandler;

    return () => {
      window.onerror = null;
    };
  }, [ handleGlobalError ]);
};

export default useErrorBoundary;
