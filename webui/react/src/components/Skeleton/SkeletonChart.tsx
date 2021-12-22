import React from 'react';

import iconChart from 'assets/images/icon-chart.svg';

import css from './SkeletonChart.module.scss';
import SkeletonSection, { Props as SkeletonSectionProps } from './SkeletonSection';

interface Props extends SkeletonSectionProps {
  size?: 'small' | 'medium' | 'large';
}

const SkeletonChart: React.FC<Props> = ({ size = 'medium', ...props }: Props) => {
  const classes = [ css.base ];

  if (size) classes.push(css[size]);

  return (
    <SkeletonSection {...props}>
      <div className={classes.join(' ')}>
        <img src={iconChart} />
      </div>
    </SkeletonSection>
  );
};

export default SkeletonChart;
