import React from 'react';

import HelmetDecorator from 'storybook/HelmetDecorator';
import StoreDecorator from 'storybook/StoreDecorator';

import Page from '../../../components/Page';

import Spinner from './Spinner';

export default {
  component: Spinner,
  decorators: [ HelmetDecorator, StoreDecorator ],
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
