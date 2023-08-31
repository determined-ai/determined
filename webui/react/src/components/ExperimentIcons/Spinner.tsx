import React, { CSSProperties } from 'react';

import css from 'components/ExperimentIcons/Spinner.module.scss';

interface Props {
  style?: CSSProperties;
  type: 'bowtie' | 'half' | 'split' | 'shadow';
}

const Spinner: React.FC<Props> = ({ type, style }) => {
  const classnames = [css.spinner, css[`spinner__${type}`]];
  return (
    <div className={css.base} style={style}>
      <div className={classnames.join(' ')} style={style} />
    </div>
  );
};

export default Spinner;
