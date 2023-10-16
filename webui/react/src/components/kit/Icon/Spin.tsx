import React from 'react';

import css from './Spin.module.scss';

interface Props {
  type: 'bowtie' | 'half' | 'shadow';
}

const Spin: React.FC<Props> = ({ type }) => {
  const classnames = [css.spinner, css[`spinner__${type}`]];
  return (
    <div className={css.base}>
      <div className={classnames.join(' ')} />
    </div>
  );
};

export const SpinBowtie: React.FC = () => <Spin type="bowtie" />;
export const SpinHalf: React.FC = () => <Spin type="half" />;
export const SpinShadow: React.FC = () => <Spin type="shadow" />;

export default Spin;
