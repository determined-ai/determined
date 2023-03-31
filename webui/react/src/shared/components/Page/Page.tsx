import React, { MutableRefObject } from 'react';

import Spinner from 'shared/components/Spinner';
import { CommonProps } from 'shared/types';

import css from './Page.module.scss';

export interface BreadCrumbRoute {
  breadcrumbName: string;
  breadcrumbTooltip?: string;
  path: string;
}

export interface Props extends CommonProps {
  bodyNoPadding?: boolean;
  breadcrumb?: BreadCrumbRoute[];
  containerRef?: MutableRefObject<HTMLElement | null>;
  headerComponent?: React.ReactNode;
  id?: string;
  loading?: boolean;
  options?: React.ReactNode;
  pageHeader?: React.ReactNode;
  stickyHeader?: boolean;
  subTitle?: React.ReactNode;
  title?: string;
}

const Page: React.FC<Props> = (props: Props) => {
  const classes = [props.className, css.base];

  const showHeader = !props.headerComponent && (props.breadcrumb || props.title);

  if (props.bodyNoPadding) classes.push(css.bodyNoPadding);
  if (props.stickyHeader) classes.push(css.stickyHeader);

  return (
    <article className={classes.join(' ')} id={props.id} ref={props.containerRef}>
      {props.headerComponent}
      {showHeader && props.pageHeader}
      <div className={css.body}>
        <Spinner spinning={!!props.loading}>{props.children}</Spinner>
      </div>
    </article>
  );
};

export default Page;
