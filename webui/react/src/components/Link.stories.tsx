import React from 'react';

import RouterDecorator from 'storybook/RouterDecorator';
import { StoryMetadata } from 'storybook/types';

import Link from './Link';

export default {
  component: Link,
  decorators: [ RouterDecorator ],
  title: 'Link',
} as StoryMetadata;

export const Default = (): React.ReactNode => (
  <Link path="test">Plain Link</Link>
);

export const Popout = (): React.ReactNode => (
  <Link path="test" popout>Plain Link</Link>
);

export const Disabled = (): React.ReactNode => (
  <Link disabled path="test">Plain Link</Link>
);
