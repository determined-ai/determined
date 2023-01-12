import React, { useMemo } from 'react';

import Breadcrumb from 'components/kit/Breadcrumb';
import Tooltip from 'components/kit/Tooltip';
import { BreadCrumbRoute } from 'shared/components/Page';
import { CommonProps } from 'shared/types';

import Link from '../Link';

import css from './PageHeader.module.scss';

export interface Props extends CommonProps {
  breadcrumb?: BreadCrumbRoute[];
  docTitle?: string;
  options?: React.ReactNode;
  sticky?: boolean;
  subTitle?: React.ReactNode;
  title?: string;
}

const PageHeader: React.FC<Props> = (props: Props) => {
  const classes = [css.base, props.className];

  const showHeader = props.title || props.subTitle || props.options;

  if (props.sticky) classes.push(css.sticky);

  const breadcrumbItems = useMemo(() => {
    const routes = props.breadcrumb ?? [];
    return routes.map((route) => {
      const last = routes.indexOf(route) === routes.length - 1;
      return last ? (
        <Breadcrumb.Item>{route.breadcrumbName}</Breadcrumb.Item>
      ) : (
        <Breadcrumb.Item>
          <Link path={route.path}>
            {route.breadcrumbTooltip ? (
              <Tooltip title={route.breadcrumbTooltip}>
                <span>{route.breadcrumbName}</span>
              </Tooltip>
            ) : (
              route.breadcrumbName
            )}
          </Link>
        </Breadcrumb.Item>
      );
    });
  }, [props.breadcrumb]);

  return (
    <div className={classes.join(' ')}>
      {props.breadcrumb && (
        <div className={css.breadcrumbs}>
          <Breadcrumb>{breadcrumbItems}</Breadcrumb>
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
