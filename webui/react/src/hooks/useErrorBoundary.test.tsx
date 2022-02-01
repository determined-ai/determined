import { render } from '@testing-library/react';
import React from 'react';

import useErrorBoundary from './useErrorBoundary';

const handleGlobalError = jest.fn();
const WrapperComponent = () => {
  useErrorBoundary(handleGlobalError);
  return <div />;
};

describe('useErrorBoundary', () => {
  it('should add global erro handler upon mounting', () => {
    render(<WrapperComponent />);
    expect(window.onerror).toBeDefined();
  });

  it('should catch uncaught error', () => {
    const ERROR_MESSAGE = 'my-oh-my';
    render(<WrapperComponent />);

    try {
      throw new Error(ERROR_MESSAGE);
    } catch (e) {
      window.onerror?.call(window, (e as Error).toString());
    }

    expect(handleGlobalError).toHaveBeenCalledWith(`Error: ${ERROR_MESSAGE}`);
  });

  it('should remove global error handler upon unmount', () => {
    const { unmount } = render(<WrapperComponent />);
    unmount();
    expect(window.onerror).toBeNull();
  });
});
