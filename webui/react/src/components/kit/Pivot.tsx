import { Tabs } from 'antd';
import React, { CSSProperties, ReactNode } from 'react';

export type TabItem = {
  children?: ReactNode;
  key: string;
  label: string;
};

interface PivotProps {
  activeKey?: string;
  className?: string;
  defaultActiveKey?: string;
  destroyInactiveTabPane?: boolean;
  items?: TabItem[];
  onChange?: () => void;
  onTabClick?: () => void;
  tabBarExtraContent?: ReactNode;
  tabBarStyle?: CSSProperties;
  type?: 'line' | 'card' | 'editable-card';
}

const Pivot: React.FC<PivotProps> = ({ type = 'line', ...props }: PivotProps) => {
  return <Tabs type={type} {...props} />;
};

export default Pivot;
