import { Breadcrumb } from 'antd';
import { Route } from 'antd/es/breadcrumb/Breadcrumb';
import React from 'react';

import { CommonProps } from 'types';

import Link from './Link';
import { BreadCrumbRoute } from './Page';
import css from './PageHeader.module.scss';

export interface Props extends CommonProps {
  breadcrumb?: BreadCrumbRoute[];
  docTitle?: string;
  options?: React.ReactNode;
  sticky?: boolean;
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

const PageHeader: React.FC<Props> = (props: Props) => {
  const classes = [ css.base ];

  const showHeader = props.title || props.subTitle || props.options;

  if (props.sticky) classes.push(css.sticky);

  return (
    <div className={classes.join(' ')}>
      {props.breadcrumb && (
        <div className={css.breadcrumbs}>
          <Breadcrumb itemRender={breadCrumbRender} routes={props.breadcrumb} />
        </div>
      )}
      {showHeader && (
        <div className={css.header}>
          <div className={css.title}>{props.title}</div>
          <div className={css.subTitle}>{props.subTitle}</div>
          <div className={css.options}>{props.options}</div>
        </div>
      )}
    </div>
  );
};

export default PageHeader;
