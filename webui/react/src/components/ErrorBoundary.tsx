/* eslint-disable @typescript-eslint/no-unused-vars */
import React from 'react';

import handleError from 'utils/error';

interface Props {
  children?: React.ReactNode;
}

interface State {
  hasError: boolean;
}

class ErrorBoundary extends React.Component<Props, State> {
  public state: State = { hasError: false };

  static getDerivedStateFromError(error: Error): State {
    // Update state so the next render will show the fallback UI.
    return { hasError: true };
  }

  componentDidCatch(error: Error, _errorInfo: React.ErrorInfo): void {
    // You can also log the error to an error reporting service
    handleError(error, { silent: true });
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      return <h1>Something went wrong.</h1>;
    }
    return this.props.children;
  }
}

export default ErrorBoundary;
