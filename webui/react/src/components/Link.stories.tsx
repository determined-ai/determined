import React from 'react';

import Link from './Link';

export default {
  component: Link,
  title: 'Determined/Link',
};

export const Default = (): React.ReactNode => <Link path="">Plain Link</Link>;

export const Popout = (): React.ReactNode => (
  <Link path="" popout>
    Plain Link
  </Link>
);

export const Disabled = (): React.ReactNode => (
  <Link disabled path="">
    Disabled Plain Link
  </Link>
);

export const DisabledButton = (): React.ReactNode => (
  <Link disabled isButton path="">
    Disabled Button Link
  </Link>
);
