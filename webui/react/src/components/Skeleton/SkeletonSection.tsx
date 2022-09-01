import { Skeleton } from 'antd';
import { SkeletonTitleProps } from 'antd/lib/skeleton/Title';
import React, { useMemo } from 'react';

import iconChart from 'shared/assets/images/icon-chart.svg';
import iconLogs from 'shared/assets/images/icon-logs.svg';
import { isNumber } from 'shared/utils/data';

import css from './SkeletonSection.module.scss';

export interface Props {
  children?: React.ReactNode;
  contentType?: ContentType;
  filters?: boolean | number | SkeletonTitleProps | SkeletonTitleProps[];
  size?: 'small' | 'medium' | 'large' | 'max';
  title?: boolean | SkeletonTitleProps;
}

export enum ContentType {
  Chart = 'Chart',
  Logs = 'Logs',
}

const renderTitle = (title?: boolean | SkeletonTitleProps) => (
  <Skeleton className={css.title} paragraph={false} title={title} />
);

const renderFilter = (key: number, title?: SkeletonTitleProps) => (
  <Skeleton className={css.filter} key={key} paragraph={false} title={title} />
);

const SkeletonSection: React.FC<Props> = ({
  children,
  contentType,
  filters,
  size = 'medium',
  title,
}: Props) => {
  const classes = [ css.base, css[size] ];
  const showHeader = !!title || !!filters;

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

  const contentSkeleton = useMemo(() => {
    if (React.isValidElement(children)) return children;

    let content = null;
    if (contentType === ContentType.Chart) content = <img src={iconChart} />;
    if (contentType === ContentType.Logs) content = <img src={iconLogs} />;
    return <div className={css.content}>{content}</div>;
  }, [ children, contentType ]);

  return (
    <div className={classes.join(' ')}>
      {showHeader && (
        <div className={css.header}>
          {titleSkeleton}
          {filterSkeleton}
        </div>
      )}
      {contentSkeleton}
    </div>
  );
};

export default SkeletonSection;
