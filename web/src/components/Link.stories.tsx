import React from 'react';

import Link from './Link';

export default {
  component: Link,
  title: 'Link',
};

export const Default = (): React.ReactNode => (
  <Link path="test">Plain Link</Link>
);

export const Popout = (): React.ReactNode => (
  <Link path="test" popout>Plain Link</Link>
);

export const Disabled = (): React.ReactNode => (
  <Link disabled path="test">Plain Link</Link>
);
