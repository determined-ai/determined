import { Skeleton } from 'antd';
import React from 'react';

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
      <Skeleton.Image className={classes.join(' ')} />
    </SkeletonSection>
  );
};

export default SkeletonChart;
