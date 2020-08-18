import { Breadcrumb, PageHeader } from 'antd';
import { Route } from 'antd/lib/breadcrumb/Breadcrumb';
import React, { useCallback } from 'react';

import history from 'routes/history';
import { CommonProps } from 'types';

import Link from './Link';
import css from './Page.module.scss';

interface BreadCrumbRoute {
  path: string;
  breadcrumbName: string;
}

interface Props extends CommonProps {
  breadcrumb?: BreadCrumbRoute[];
  backPath?: string;
  headerInfo?: React.ReactNode;
  id?: string;
  options?: React.ReactNode;
  subTitle?: React.ReactNode;
  title?: string;
  showDivider?: boolean;
}

const breadCrumbRender = (route: Route, params: unknown, routes: Route[]) => {
  const last = routes.indexOf(route) === routes.length - 1;
  return last ? (
    <span>{route.breadcrumbName}</span>
  ) : (
    <Link path={route.path}>{route.breadcrumbName}</Link>
  );
};

const Page: React.FC<Props> = (props: Props) => {
  const showHeader = props.breadcrumb || props.title || props.backPath;
  const classes = [ props.className, css.base ];

  const handleBack = useCallback(() => {
    if (props.backPath) history.push(props.backPath);
  }, [ props.backPath ]);

  return (
    <main className={classes.join(' ')} id={props.id}>
      {props.breadcrumb && <div className={css.breadcrumbs}>
        <Breadcrumb itemRender={breadCrumbRender} routes={props.breadcrumb} />
      </div>}
      {showHeader && <PageHeader
        extra={props.options}
        subTitle={props.subTitle}
        title={props.title}
        onBack={props.backPath ? handleBack : undefined}>
        {props.headerInfo}
      </PageHeader>}
      {props.showDivider && <div className={css.divider} />}
      <div className={css.body}>{props.children}</div>
    </main>
  );
};

export default Page;
