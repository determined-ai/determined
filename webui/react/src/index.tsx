import 'micro-observables/batchingForReactDom';
import { createRoot } from 'react-dom/client';
import { RouterProvider } from 'react-router-dom';

/* Import the styles first to allow components to override styles. */
import 'uplot/dist/uPlot.min.css';

import App from 'App';
import router from 'router';

import * as serviceWorker from './serviceWorker';

import 'utils/prototypes';
import 'dev';

// redirect to basename if needed
if (process.env.PUBLIC_URL && window.location.pathname === '/') {
  window.history.replaceState({}, '', process.env.PUBLIC_URL);
}
router.initRouter(<App />);
const container = document.getElementById('root');
// eslint-disable-next-line @typescript-eslint/no-non-null-assertion
const root = createRoot(container!);

root.render(
  // <React.StrictMode>
  <RouterProvider router={router.getRouter()} />,
  // </React.StrictMode>,
);

/*
 * If you want your app to work offline and load faster, you can change
 * unregister() to register() below. Note this comes with some pitfalls.
 * Learn more about service workers: https://bit.ly/CRA-PWA
 */
serviceWorker.unregister();
