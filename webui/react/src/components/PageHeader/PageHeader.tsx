import React, { useMemo } from 'react';

import Breadcrumb from 'components/kit/Breadcrumb';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import Tooltip from 'components/kit/Tooltip';
import { BreadCrumbRoute } from 'components/Page';
import { CommonProps } from 'shared/types';

import Link from '../Link';

import css from './PageHeader.module.scss';

export interface Props extends CommonProps {
  breadcrumb: BreadCrumbRoute[];
  docTitle?: string;
  menuItems?: MenuItem[];
  onClickMenu?: (key: string) => void;
  options?: React.ReactNode;
  sticky?: boolean;
  subTitle?: React.ReactNode;
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
        <Breadcrumb.Item>
          {route.breadcrumbName}
          {props.menuItems && (
            <Dropdown menu={props.menuItems} onClick={props.onClickMenu}>
              <div style={{ cursor: 'pointer' }}>
                <Icon name="arrow-down" size="tiny" title="Action menu" />
              </div>
            </Dropdown>
          )}
        </Breadcrumb.Item>
      ) : (
        <Breadcrumb.Item>
          <Link path={route.path}>
            {route.breadcrumbTooltip ? (
              <Tooltip content={route.breadcrumbTooltip}>
                <span>{route.breadcrumbName}</span>
              </Tooltip>
            ) : (
              route.breadcrumbName
            )}
          </Link>
        </Breadcrumb.Item>
      );
    });
  }, [props.breadcrumb, props.menuItems, props.onClickMenu]);

  return (
    <div className={classes.join(' ')}>
      {props.breadcrumb && (
        <div className={css.breadcrumbs}>
          <Breadcrumb>{breadcrumbItems}</Breadcrumb>
        </div>
      )}
      {showHeader && (
        <div className={css.header}>
          <div className={css.subTitle}>{props.subTitle}</div>
          <div className={css.options}>{props.options}</div>
        </div>
      )}
    </div>
  );
};

export default PageHeader;
