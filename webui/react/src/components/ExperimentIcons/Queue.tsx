import React, { CSSProperties } from 'react';

import css from './Queue.module.scss';

interface Props {
  style?: CSSProperties;
}

const Queue: React.FC<Props> = ({ style }) => {
  return (
    <div className={css.base} style={style}>
      <div className={css.spinner} />
      <div className={css.innerSpinner} />
    </div>
  );
};

export default Queue;
