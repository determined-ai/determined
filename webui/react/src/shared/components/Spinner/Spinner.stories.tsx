import React from 'react';

import Page from 'shared/components/Page';

import Spinner from './Spinner';

export default {
  component: Spinner,
  title: 'Spinner',
};

export const Default = (): React.ReactNode => <Spinner />;

export const WithTip = (): React.ReactNode => <Spinner tip="Fetching trials." />;

export const FullPageSpinner = (): React.ReactNode => (
  <Spinner spinning={true}>
    <Page title="Page Title">
      Some page content
    </Page>
  </Spinner>
);
FullPageSpinner.parameters = { layout: 'fullscreen' };
