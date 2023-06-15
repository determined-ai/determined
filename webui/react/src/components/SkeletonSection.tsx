import { Skeleton } from 'antd';
import { SkeletonTitleProps } from 'antd/lib/skeleton/Title';
import React, { useMemo } from 'react';

import iconChart from 'assets/images/icon-chart.svg';
import iconLogs from 'assets/images/icon-logs.svg';
import { ValueOf } from 'types';
import { isNumber } from 'utils/data';

import css from './SkeletonSection.module.scss';

export const ContentType = {
  Chart: 'Chart',
  Logs: 'Logs',
} as const;

export type ContentType = ValueOf<typeof ContentType>;

export interface Props {
  children?: React.ReactNode;
  contentType?: ContentType;
  filters?: boolean | number | SkeletonTitleProps | SkeletonTitleProps[];
  size?: 'small' | 'medium' | 'large' | 'max';
  title?: boolean | SkeletonTitleProps;
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
  const classes = [css.base, css[size]];
  const showHeader = !!title || !!filters;

  const titleSkeleton = useMemo(() => {
    if (!title) return null;
    return renderTitle(title);
  }, [title]);

  const filterSkeleton = useMemo(() => {
    if (!filters) return null;

    let content: React.ReactNode[] = [];
    if (isNumber(filters)) {
      content = new Array(filters).fill(null).map((_, index) => renderFilter(index));
    } else {
      const filterProps = (Array.isArray(filters) ? filters : [filters]) as SkeletonTitleProps[];
      content = filterProps.map((props, index) => renderFilter(index, props));
    }
    return <div className={css.filters}>{content}</div>;
  }, [filters]);

  const contentSkeleton = useMemo(() => {
    if (React.isValidElement(children)) return children;

    let content: React.ReactNode = undefined;
    if (contentType === ContentType.Chart) content = <img src={iconChart} />;
    if (contentType === ContentType.Logs) content = <img src={iconLogs} />;
    return <div className={css.content}>{content}</div>;
  }, [children, contentType]);

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
