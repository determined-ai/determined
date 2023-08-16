import { Skeleton } from 'antd';
import React, { useMemo } from 'react';

import SkeletonSection, { Props as SkeletonSectionProps } from 'components/SkeletonSection';
import { isNumber } from 'utils/data';

import css from './SkeletonTable.module.scss';

interface Props extends SkeletonSectionProps {
  columns?: number | React.CSSProperties[];
  rows?: number;
}

const SkeletonTable: React.FC<Props> = ({ columns = 10, rows = 10, ...props }: Props) => {
  const columnProps = useMemo(() => {
    if (isNumber(columns)) return new Array(columns).fill({});
    return columns;
  }, [columns]);
  return (
    <SkeletonSection {...props}>
      <div className={css.base}>
        {new Array(rows).fill(null).map((_, rowIndex) => (
          <div className={css.row} key={rowIndex}>
            {columnProps.map((colProps, colIndex) => (
              <div className={css.col} key={colIndex} style={colProps}>
                <Skeleton paragraph={{ rows: 1, width: '100%' }} title={false} />
              </div>
            ))}
          </div>
        ))}
      </div>
    </SkeletonSection>
  );
};

export default SkeletonTable;
