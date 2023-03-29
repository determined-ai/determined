import React from 'react';

import css from './index.module.scss';

interface Props {
  classes?: string;
  subHeader?: boolean;
}

const Header: React.FC<React.PropsWithChildren<Props>> = ({
  classes = '',
  children,
  subHeader,
}) => {
  const style = [css.headerBase];
  let element = '';

  if (subHeader) {
    element = 'h2';

    style.push(css.header2);
  } else {
    element = 'h1';

    style.push(css.header1);
  }

  style.push(classes);

  return React.createElement(element, { className: style.join(' ') }, children);
};

export default Header;
