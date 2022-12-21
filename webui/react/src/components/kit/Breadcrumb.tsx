import { Breadcrumb as AntdBreadcrumb } from 'antd';
import React, { ReactNode } from 'react';

interface BreadcrumbProps {
  children?: ReactNode;
  className?: string;
  separator?: ReactNode;
}

interface BreadcrumbItemProps {
  children?: ReactNode;
}

type BreadcrumbItem = React.FC<BreadcrumbItemProps>;
type BreadcrumbSeparator = React.FC;
type Breadcrumb = React.FC<BreadcrumbProps> & { Item: typeof BreadcrumbItem, Separator: typeof BreadcrumbSeparator };

const Breadcrumb: Breadcrumb = ({ separator = '/', ...props }: BreadcrumbProps): JSX.Element => {
  return (
    <AntdBreadcrumb separator={separator} {...props} />);
};

const BreadcrumbItem: BreadcrumbItem = (props: BreadcrumbItemProps) => {
  return (
    <AntdBreadcrumb.Item {...props} />
  );
};

const BreadcrumbSeparator: BreadcrumbSeparator = () => {
  return (
    <AntdBreadcrumb.Separator />
  );
};

Breadcrumb.Item = BreadcrumbItem;
Breadcrumb.Separator = BreadcrumbSeparator;

export default Breadcrumb;
