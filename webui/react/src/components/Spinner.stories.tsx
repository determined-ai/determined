import React from 'react';

import { InfoDecorator } from 'storybook/ContextDecorators';

import Page from './Page';
import Spinner from './Spinner';

export default {
  component: Spinner,
  decorators: [ InfoDecorator ],
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
