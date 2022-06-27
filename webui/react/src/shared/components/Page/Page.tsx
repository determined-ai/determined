import React, { MutableRefObject } from 'react';

import PageHeader from 'shared/components/PageHeader/PageHeader';
import Spinner from 'shared/components/Spinner/Spinner';
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
  containerRef?: MutableRefObject<HTMLElement | null>,
  headerComponent?: React.ReactNode,
  id?: string;
  loading?: boolean;
  options?: React.ReactNode;
  stickyHeader?: boolean;
  subTitle?: React.ReactNode;
  title?: string;
}

const Page: React.FC<Props> = (props: Props) => {
  const classes = [ props.className, css.base ];

  const showHeader = !props.headerComponent && (props.breadcrumb || props.title);

  if (props.bodyNoPadding) classes.push(css.bodyNoPadding);
  if (props.stickyHeader) classes.push(css.stickyHeader);

  return (
    <main className={classes.join(' ')} id={props.id} ref={props.containerRef}>
      {props.headerComponent}
      {showHeader && (
        <PageHeader
          breadcrumb={props.breadcrumb}
          options={props.options}
          sticky={props.stickyHeader}
          subTitle={props.subTitle}
          title={props.title}
        />
      )}
      <div className={css.body}>
        <Spinner spinning={!!props.loading}>{props.children}</Spinner>
      </div>
    </main>
  );
};

export default Page;
