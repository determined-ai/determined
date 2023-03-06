import 'micro-observables/batchingForReactDom';
import React, { ReactNode } from 'react';
import { createRoot } from 'react-dom/client';
import { createBrowserRouter, RouterProvider } from 'react-router-dom';

/* Import the styles first to allow components to override styles. */
import 'shared/styles/index.scss';
import 'uplot/dist/uPlot.min.css';

import App from './App';
import * as serviceWorker from './serviceWorker';
import 'shared/prototypes';
import 'dev';

// redirect to basename if needed
if (process.env.PUBLIC_URL && window.location.pathname === '/') {
  window.location.href = process.env.PUBLIC_URL;
}

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
    router.navigate('/logout');
  }

  render(): React.ReactNode {
    if (this.state.hasError) {
      // You can render any custom fallback UI
      return <div />;
    }

    return <App />;
  }
}

export const router = createBrowserRouter(
  [
    // match everything with "*"
    { element: <ErrorBoundary />, path: '*' },
  ],
  { basename: process.env.PUBLIC_URL },
);

const container = document.getElementById('root');
// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const root = createRoot(container!);

root.render(
  // <React.StrictMode>
  <RouterProvider router={router} />,
  // </React.StrictMode>,
);

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
