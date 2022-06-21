import React from 'react';

import Spinner from 'shared/components/Spinner';

/** placeholder page for depracated pages. displayed while the user is redirected */
const Redirect: React.FC = () => (
  <Spinner tip="Deprecated page. Redirecting" />
);

export default Redirect;
