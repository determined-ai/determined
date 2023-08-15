import React, { useMemo } from 'react';

import Breadcrumb from 'components/kit/Breadcrumb';
import { MenuItem } from 'components/kit/Dropdown';
import Tooltip from 'components/kit/Tooltip';
import Link from 'components/Link';
import { BreadCrumbRoute } from 'components/Page';
import { CommonProps } from 'types';

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
      return (
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
  }, [props.breadcrumb]);

  return (
    <>
      {showPageHeader && (
        <div className={classes.join(' ')}>
          <Breadcrumb menuItems={props.menuItems} onClickMenu={props.onClickMenu}>
            {breadcrumbItems}
          </Breadcrumb>
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
