import React from 'react';

import Spinner from 'shared/components/Spinner';

/** placeholder page for depracated pages. displayed while the user is redirected */
const Redirect: React.FC = () => <Spinner tip="This page is deprecated. Redirecting..." />;

export default Redirect;
