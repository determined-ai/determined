import * as process from 'process';

import { createBrowserRouter } from 'react-router-dom';

/**
 * Avoid using of importActual to avoid circular dependencies
 * that can cause test workers to hang indefinitely. Also simplify
 * the app router to a deterministic router instance (no null instances).
 * https://github.com/vitest-dev/vitest/issues/546
 */
class AppRouter {
  routerInstance = createBrowserRouter([{ element: <div />, path: '*' }], {
    basename: process.env.PUBLIC_URL,
  });

  getRouter() {
    return this.routerInstance;
  }
}
export default new AppRouter();
