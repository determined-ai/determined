import { Breadcrumb as AntdBreadcrumb } from 'antd';
import React, { ReactNode } from 'react';

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

export default Breadcrumb;
