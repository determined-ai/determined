import React, { useMemo } from 'react';

import Breadcrumb from 'components/kit/Breadcrumb';
import Button from 'components/kit/Button';
import { Column, Columns } from 'components/kit/Columns';
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
}

const PageHeader: React.FC<Props> = (props: Props) => {
  const classes = [css.base, props.className];

  const showHeader = props.options;

  // TODO: The breadcrumb is required on every Page component
  // however the SignIn and DesignKit pages are special cases
  // where a breadcrumb is not shown. Once both components are changed
  // these two checks should be removed.
  const showBreadcrumb = props.breadcrumb.length > 0;
  const showPageHeader = showBreadcrumb || showHeader;

  if (props.sticky) classes.push(css.sticky);

  const breadcrumbItems = useMemo(() => {
    const routes = props.breadcrumb ?? [];
    return routes.map((route) => {
      const last = routes.indexOf(route) === routes.length - 1;
      return last ? (
        <Breadcrumb.Item key={route.breadcrumbName}>
          <Columns>
            <Column>{route.breadcrumbName}</Column>
            {props.menuItems && (
              <Column>
                <Dropdown menu={props.menuItems} onClick={props.onClickMenu}>
                  <div className={css.options}>
                    <Button
                      icon={<Icon name="arrow-down" size="tiny" title="Action menu" />}
                      size="small"
                      type="text"
                    />
                  </div>
                </Dropdown>
              </Column>
            )}
          </Columns>
        </Breadcrumb.Item>
      ) : (
        <Breadcrumb.Item key={route.breadcrumbName}>
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
    <>
      {showPageHeader && (
        <div className={classes.join(' ')}>
          <div className={css.breadcrumbs}>
            <Breadcrumb>{breadcrumbItems}</Breadcrumb>
          </div>
          {showHeader && (
            <div className={css.header}>
              <div className={css.options}>{props.options}</div>
            </div>
          )}
        </div>
      )}
    </>
  );
};

export default PageHeader;
