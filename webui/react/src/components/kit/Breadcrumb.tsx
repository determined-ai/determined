import { Breadcrumb as AntdBreadcrumb } from 'antd';
import React, { ReactNode } from 'react';

import css from './Breadcrumb.module.scss';

interface BreadcrumbProps {
  children?: ReactNode;
  separator?: ReactNode;
}

interface BreadcrumbItemProps {
  children?: ReactNode;
}

type BreadcrumbItem = React.FC<BreadcrumbItemProps>;
type BreadcrumbSeparator = React.FC;
type Breadcrumb = React.FC<BreadcrumbProps> & {
  Item: BreadcrumbItem;
  Separator: BreadcrumbSeparator;
};

const Breadcrumb: Breadcrumb = AntdBreadcrumb;

Breadcrumb.Item = AntdBreadcrumb.Item;
Breadcrumb.Separator = AntdBreadcrumb.Separator;

export const BreadcrumbBar: React.FC<React.PropsWithChildren> = ({ children }) => {
  return <div className={css.base}>{children}</div>;
};

export default Breadcrumb;
