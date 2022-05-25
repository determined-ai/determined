import { Breadcrumb, Tooltip } from 'antd';
import React from 'react';

import { BreadCrumbRoute } from 'shared/components/Page';

import Link from '../../../components/Link';
import { CommonProps } from '../../types';

import css from './PageHeader.module.scss';

export interface Props extends CommonProps {
  breadcrumb?: BreadCrumbRoute[];
  docTitle?: string;
  options?: React.ReactNode;
  sticky?: boolean;
  subTitle?: React.ReactNode;
  title?: string;
}

const breadCrumbRender = (route: BreadCrumbRoute, params: unknown, routes: BreadCrumbRoute[]) => {
  const last = routes.indexOf(route) === routes.length - 1;
  return last ? (
    <span>{route.breadcrumbName}</span>
  ) : (
    <Link path={route.path}>
      {route.breadcrumbTooltip ? (
        <Tooltip title={route.breadcrumbTooltip}>
          <span>{route.breadcrumbName}</span>
        </Tooltip>
      ) : route.breadcrumbName}
    </Link>
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
