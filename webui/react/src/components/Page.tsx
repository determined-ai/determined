import { PageHeader } from 'antd';
import React, { useCallback } from 'react';

import history from 'routes/history';
import { CommonProps } from 'types';

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
  maxHeight?: boolean;
  showDivider?: boolean;
}

const Page: React.FC<Props> = (props: Props) => {
  const showHeader = props.breadcrumb || props.title || props.backPath;
  const classes = [ props.className, css.base ];

  if (props.maxHeight) classes.push(css.maxHeight);

  const handleBack = useCallback(() => {
    if (props.backPath) history.push(props.backPath);
  }, [ props.backPath ]);

  return (
    <main className={classes.join(' ')} id={props.id}>
      {showHeader && <PageHeader
        breadcrumb={props.breadcrumb && { routes: props.breadcrumb }}
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
