import React from 'react';
import { createBrowserRouter } from 'react-router-dom';

import 'styles/index.scss';

class AppRouter {
  routerInstance: ReturnType<typeof createBrowserRouter> | null = null;
  getRouter() {
    if (this.routerInstance) return this.routerInstance;
    throw new Error('Router called before instantiation -- call AppRouter#initRouter first');
  }
  initRouter(app: React.ReactElement) {
    this.routerInstance = createBrowserRouter(
      [
        // match everything with "*"
        { element: app, path: '*' },
      ],
      { basename: process.env.PUBLIC_URL },
    );
    return this.routerInstance;
  }
}

export default new AppRouter();
