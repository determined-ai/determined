import React, { ReactNode } from 'react';
import { createBrowserRouter } from 'react-router-dom';

import { paths } from 'routes/utils';
import { isAuthFailure } from 'shared/utils/service';

import App from './App';

interface State {
  hasError: boolean;
}

interface NodeProps {
  children?: ReactNode[];
}

class ErrorBoundary extends React.Component<NodeProps, State> {
  constructor(props: NodeProps, state: State) {
    super(props, state);
  }

  // Error, React.ErrorInfo
  componentDidCatch(e: Error): void {
    console.log('in componentDidCatch');
    if (isAuthFailure(e)) {
      router.navigate(paths.logout());
    }
  }

  render(): React.ReactNode {
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
