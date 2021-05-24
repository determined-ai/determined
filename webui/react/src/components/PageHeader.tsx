import { Breadcrumb, Tooltip } from 'antd';
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

export type BreadcrumbRoute = Route

const tooLongExperimentDescription = 40;

const isExperimentIdWithDescCrumb = (breadcrumbName: string) => {
  const crumbElements = breadcrumbName.split(' ');
  // The experiment description may have spaces in it, so there should be at least 3 elements
  if (crumbElements.length < 3
      || crumbElements[0] !== 'Experiment'
      || isNaN(parseInt(crumbElements[1]))) {
    return false;
  }

  const expDescWithParens = crumbElements.slice(2).join(' ');
  return (expDescWithParens.startsWith('(') && expDescWithParens.endsWith(')'));
};

const buildExperimentInfoCrumbElement = (breadcrumbName: string) => {
  const breadcrumbParts = breadcrumbName.split(' ');
  const experimentId = breadcrumbParts[1];
  // Reconstitute the breadcrumbName minus 'Experiment $ID', then remove the parentheses
  const experimentDesc = breadcrumbParts.slice(2).join(' ').slice(1).slice(0, -1);

  if (experimentDesc.length > tooLongExperimentDescription) {
    const truncatedDesc = experimentDesc.slice(0, tooLongExperimentDescription);
    return <Tooltip title={experimentDesc}>
      <span>{`Experiment ${experimentId} (${truncatedDesc}...)`}</span>
    </Tooltip>;
  } else {
    return <>{breadcrumbName}</>;
  }
};

const breadCrumbRender = (route: Route, params: unknown, routes: Route[]) => {

  const last = routes.indexOf(route) === routes.length - 1;

  return last ? (
    <span>{route.breadcrumbName}</span>
  ) : (
    <Link path={route.path}>{
      isExperimentIdWithDescCrumb(route.breadcrumbName)
        ? buildExperimentInfoCrumbElement(route.breadcrumbName)
        : route.breadcrumbName}
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
