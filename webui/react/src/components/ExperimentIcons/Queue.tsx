import React, { CSSProperties, useMemo } from 'react';

import css from './Queue.module.scss';

interface Props {
  // only height, width, opacity, and backgroundColor are available
  style?: CSSProperties;
}

const Queue: React.FC<Props> = ({ style }) => {
  const spinnerStyle = useMemo(() => {
    return { backgroundColor: style?.backgroundColor, opacity: style?.opacity };
  }, [style?.backgroundColor, style?.opacity]);

  return (
    <div className={css.base} style={{ height: style?.height, width: style?.width }}>
      <div className={css.spinner} style={spinnerStyle} />
      <div className={css.innerSpinner} style={spinnerStyle} />
    </div>
  );
};

export default Queue;
