import React, { CSSProperties } from 'react';

import css from './Queue.module.scss';

interface Props {
  height?: CSSProperties['height'];
  width?: CSSProperties['width'];
}

const Queue: React.FC<Props> = ({ height, width }) => {
  return (
    <div className={css.base} style={{ height, width }}>
      <div className={css.spinner}>
        <div className={css.inner_spinner} />
      </div>
    </div>
  );
};

export default Queue;
