import { Breadcrumb, PageHeader } from 'antd';
import { Route } from 'antd/lib/breadcrumb/Breadcrumb';
import React from 'react';
import { Helmet } from 'react-helmet';

import { CommonProps } from 'types';

import Info from '../contexts/Info';

import Link from './Link';
import css from './Page.module.scss';

export interface BreadCrumbRoute {
  breadcrumbName: string;
  path: string;
}

export interface Props extends CommonProps {
  breadcrumb?: BreadCrumbRoute[];
  docTitle?: string;
  headerInfo?: React.ReactNode;
  id?: string;
  options?: React.ReactNode;
  showDivider?: boolean;
  subTitle?: React.ReactNode;
  title?: string;
}

const breadCrumbRender = (route: Route, params: unknown, routes: Route[]) => {
  const last = routes.indexOf(route) === routes.length - 1;
  return last ? (
    <span>{route.breadcrumbName}</span>
  ) : (
    <Link path={route.path}>{route.breadcrumbName}</Link>
  );
};

const getFullDocTitle = (title?: string, clusterName?: string) => {
  const segmentList = [];
  if (title) {
    segmentList.push(title);
  }
  if (clusterName) {
    segmentList.push(clusterName);
  }
  segmentList.push('Determined');

  return segmentList.join(' - ');
};

const Page: React.FC<Props> = (props: Props) => {
  const classes = [ props.className, css.base ];
  const info = Info.useStateContext();
  const showHeader = props.breadcrumb || props.title;

  const docTitle = getFullDocTitle(
    props.docTitle || props.title,
    info.clusterName,
  );

  return (
    <main className={classes.join(' ')} id={props.id}>
      <Helmet>
        <title>{docTitle}</title>
      </Helmet>
      {props.breadcrumb && <div className={css.breadcrumbs}>
        <Breadcrumb itemRender={breadCrumbRender} routes={props.breadcrumb} />
      </div>}
      {showHeader && <PageHeader
        backIcon={false}
        extra={props.options}
        subTitle={props.subTitle}
        title={props.title}>
        {props.headerInfo}
      </PageHeader>}
      {props.showDivider && <div className={css.divider} />}
      <div className={css.body}>{props.children}</div>
    </main>
  );
};

export default Page;
