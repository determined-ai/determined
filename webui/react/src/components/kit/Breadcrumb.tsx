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
  Item: typeof BreadcrumbItem;
  Separator: typeof BreadcrumbSeparator;
};

const Breadcrumb: Breadcrumb = AntdBreadcrumb;

const BreadcrumbItem: BreadcrumbItem = AntdBreadcrumb.Item;

const BreadcrumbSeparator: BreadcrumbSeparator = AntdBreadcrumb.Separator;

Breadcrumb.Item = BreadcrumbItem;
Breadcrumb.Separator = BreadcrumbSeparator;

export default Breadcrumb;
