import React from 'react';

import css from './BreadcrumbBar.module.scss';

const BreadcrumbBar: React.FC<React.PropsWithChildren> = ({ children }) => {
  return <div className={css.base}>{children}</div>;
};

export default BreadcrumbBar;
