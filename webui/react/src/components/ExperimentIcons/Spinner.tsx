import React, { CSSProperties } from 'react';

import css from './Spinner.module.scss';

interface Props {
  height?: CSSProperties['height'];
  type: 'bowtie' | 'half' | 'split' | 'shadow';
  width?: CSSProperties['width'];
}

const Spinner: React.FC<Props> = ({ type, height, width }) => {
  const classnames = [css.spinner, css[`spinner__${type}`]];
  return (
    <div className={css.base} style={{ height, width }}>
      <div className={classnames.join(' ')} style={{ height, width }} />
    </div>
  );
};

export default Spinner;
