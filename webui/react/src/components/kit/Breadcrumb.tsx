import { Breadcrumb as AntdBreadcrumb } from 'antd';
import { ItemType } from 'antd/es/breadcrumb/Breadcrumb';
import React, { ReactNode } from 'react';

interface BreadcrumbProps {
  items: ItemType[];
  separator?: ReactNode;
}

type Breadcrumb = React.FC<BreadcrumbProps>;

const Breadcrumb: Breadcrumb = AntdBreadcrumb;

export default Breadcrumb;
