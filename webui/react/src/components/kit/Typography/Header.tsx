import React from 'react';

import css from './index.module.scss';

interface Props {
  classes?: string;
}

const Header: React.FC<React.PropsWithChildren<Props>> = ({ classes = '', children }) => {
  const style = [css.header, classes];

  return <h1 className={style.join(' ')}>{children}</h1>;
};

export default Header;
