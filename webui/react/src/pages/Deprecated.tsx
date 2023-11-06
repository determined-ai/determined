import Spinner from 'hew/Spinner';
import React from 'react';

/** placeholder page for depracated pages. displayed while the user is redirected */
const Redirect: React.FC = () => <Spinner spinning tip="This page is deprecated. Redirecting..." />;

export default Redirect;
