import React, { ReactNode } from 'react';
import { createBrowserRouter } from 'react-router-dom';

import { paths } from 'routes/utils';

import App from './App';

interface State {
  hasError: boolean;
}

interface NodeProps {
  children?: ReactNode[];
}

class ErrorBoundary extends React.Component<NodeProps, State> {
  constructor(props: NodeProps) {
    super(props);
    this.state = { hasError: false };
  }

  componentDidCatch(): void {
    // You can also log the error to an error reporting service
    // logErrorToMyService(error, errorInfo);
    // console.log('error: ' + error);
    // console.log('errorInfo: ' + JSON.stringify(errorInfo));
    // console.log('componentStack: ' + errorInfo.componentStack);
    router.navigate(paths.logout());
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      // You can render any custom fallback UI
      return <div />;
    }

    return <App />;
  }
}

const router = createBrowserRouter(
  [
    // match everything with "*"
    { element: <ErrorBoundary />, path: '*' },
  ],
  { basename: process.env.PUBLIC_URL },
);

export default router;
