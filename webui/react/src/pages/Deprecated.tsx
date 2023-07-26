import React from 'react';

import Spinner from 'components/kit/Spinner';

/** placeholder page for depracated pages. displayed while the user is redirected */
const Redirect: React.FC = () => <Spinner spinning tip="This page is deprecated. Redirecting..." />;

export default Redirect;
