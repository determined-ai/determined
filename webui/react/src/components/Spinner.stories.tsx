import React from 'react';

import Page from './Page';
import Spinner from './Spinner';

export default {
  component: Spinner,
  title: 'Spinner',
};

export const Default = (): React.ReactNode => <Spinner />;

export const FullPageSpinner = (): React.ReactNode => (
  <Spinner spinning={true}>
    <Page title="Page Title">
      Some page content
    </Page>
  </Spinner>
);
FullPageSpinner.parameters = { layout: 'fullscreen' };
