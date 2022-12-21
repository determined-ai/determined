import { Breadcrumb } from 'antd';
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
type BreadcrumbComponent = React.FC<BreadcrumbProps> & { Item: typeof BreadcrumbItem, Separator: typeof BreadcrumbSeparator };

const BreadcrumbComponent: BreadcrumbComponent = ({ separator = '/', ...props }: BreadcrumbProps): JSX.Element => {
  return (
    <Breadcrumb separator={separator} {...props} />);
};

const BreadcrumbItem: BreadcrumbItem = (props: BreadcrumbItemProps) => {
  return (
    <Breadcrumb.Item {...props} />
  );
};

const BreadcrumbSeparator: BreadcrumbSeparator = () => {
  return (
    <Breadcrumb.Separator />
  );
};

BreadcrumbComponent.Item = BreadcrumbItem;
BreadcrumbComponent.Separator = BreadcrumbSeparator;

export default BreadcrumbComponent;
