import React from 'react';
import { createBrowserRouter } from 'react-router-dom';

import App from './App';

const router = createBrowserRouter(
  [
    // match everything with "*"
    { element: <App />, path: '*' },
  ],
  { basename: process.env.PUBLIC_URL },
);

export default router;
