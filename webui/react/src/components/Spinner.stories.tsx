import React from 'react';

import Spinner from './Spinner';

export default {
  component: Spinner,
  title: 'Spinner',
};

export const Default = (): React.ReactNode => <Spinner />;

export const FullPageSpinner = (): React.ReactNode => (
  <Spinner fullPage={true}>NavMain Spinner</Spinner>
);
