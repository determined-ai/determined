import { Tabs } from 'antd';
import React, { KeyboardEvent, MouseEvent, ReactNode } from 'react';

import css from './Pivot.module.scss';

export type TabItem = {
  children?: ReactNode;
  key: string;
  label: ReactNode;
};

interface PivotProps {
  activeKey?: string;
  defaultActiveKey?: string;
  destroyInactiveTabPane?: boolean;
  items?: TabItem[];
  onChange?: (activeKey: string) => void;
  onTabClick?: (key: string, event: MouseEvent | KeyboardEvent) => void;
  tabBarExtraContent?: ReactNode;
  type?: 'line' | 'card';
}

const Pivot: React.FC<PivotProps> = ({ type = 'line', ...props }: PivotProps) => {
  return (
    <div className={css.base}>
      <Tabs type={type} {...props} />
    </div>
  );
};

export default Pivot;
