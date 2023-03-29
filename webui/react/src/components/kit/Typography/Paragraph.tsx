import React from 'react';

import css from './index.module.scss';

interface Props {
  classes?: string;
}

const Paragraph: React.FC<React.PropsWithChildren<Props>> = ({ classes = '', children }) => {
  const style = [css.base, classes];
  return <p className={style.join(' ')}>{children}</p>;
};

export default Paragraph;
