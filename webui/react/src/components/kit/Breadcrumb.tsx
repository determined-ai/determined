import { Breadcrumb as AntdBreadcrumb } from 'antd';
import React, { ReactNode } from 'react';

import Button from 'components/kit/Button';
import { Column, Columns } from 'components/kit/Columns';
import Dropdown from 'components/kit/Dropdown';
import { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';

import css from './Breadcrumb.module.css';
interface BreadcrumbProps {
  children?: ReactNode;
  separator?: ReactNode;
  menuItems?: MenuItem[];
  onClickMenu?: (key: string) => void;
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

const Breadcrumb: Breadcrumb = (props: BreadcrumbProps) => {
  return (
    <div className={css.base}>
      <Columns>
        <Column>
          <AntdBreadcrumb separator={props.separator}>{props.children}</AntdBreadcrumb>
        </Column>
        {props.menuItems && (
          <Column align="left">
            <Dropdown menu={props.menuItems} onClick={props.onClickMenu}>
              <Button
                icon={<Icon name="arrow-down" size="tiny" title="Action menu" />}
                size="small"
                type="text"
              />
            </Dropdown>
          </Column>
        )}
      </Columns>
    </div>
  );
};

Breadcrumb.Item = AntdBreadcrumb.Item;
Breadcrumb.Separator = AntdBreadcrumb.Separator;

export default Breadcrumb;
