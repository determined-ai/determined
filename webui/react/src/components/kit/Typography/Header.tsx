import React from 'react';

import css from './index.module.scss';

interface Props {
  className?: string;
  style?: { [k: string]: string };
}

const Header: React.FC<React.PropsWithChildren<Props>> = ({ children, className, style }) => {
  const classes = [css.header, className];

  return (
    <h1 className={classes.join(' ')} style={style}>
      {children}
    </h1>
  );
};

export default Header;
