import { Skeleton } from 'antd';
import { SkeletonTitleProps } from 'antd/lib/skeleton/Title';
import React, { PropsWithChildren, useMemo } from 'react';

import { isNumber } from 'utils/data';

import css from './SkeletonSection.module.scss';

export interface Props {
  filters?: boolean | number | SkeletonTitleProps | SkeletonTitleProps[];
  maxHeight?: boolean;
  title?: boolean | SkeletonTitleProps;
}

const renderTitle = (title?: boolean | SkeletonTitleProps) => (
  <Skeleton className={css.title} paragraph={false} title={title} />
);

const renderFilter = (key: number, title?: SkeletonTitleProps) => (
  <Skeleton className={css.filter} key={key} paragraph={false} title={title} />
);

const DEFAULT_CONTENT = <div className={css.content} />;

const SkeletonSection: React.FC<Props> = ({
  children = DEFAULT_CONTENT,
  filters,
  maxHeight,
  title,
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];
  const showHeader = !!title || !!filters;

  if (maxHeight) classes.push(css.maxHeight);

  const titleSkeleton = useMemo(() => {
    if (!title) return null;
    return renderTitle(title);
  }, [ title ]);

  const filterSkeleton = useMemo(() => {
    if (!filters) return null;

    let content = null;
    if (isNumber(filters)) {
      content = new Array(filters).fill(null).map((_, index) => renderFilter(index));
    } else {
      const filterProps = (Array.isArray(filters) ? filters : [ filters ]) as SkeletonTitleProps[];
      content = filterProps.map((props, index) => renderFilter(index, props));
    }
    return <div className={css.filters}>{content}</div>;
  }, [ filters ]);

  return (
    <div className={classes.join(' ')}>
      {showHeader && (
        <div className={css.header}>
          {titleSkeleton}
          {filterSkeleton}
        </div>
      )}
      {children}
    </div>
  );
};

export default SkeletonSection;
