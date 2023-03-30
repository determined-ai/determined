import React from 'react';

import css from './index.module.scss';

const Header: React.FC<React.PropsWithChildren> = ({ children }) => {
  return <h1 className={css.header}>{children}</h1>;
};

export default Header;
