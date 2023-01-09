import { Tabs } from 'antd';
import React, { CSSProperties, KeyboardEvent, MouseEvent, ReactNode } from 'react';

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
  tabBarStyle?: CSSProperties;
  type?: 'line' | 'card' | 'editable-card';
}

const Pivot: React.FC<PivotProps> = ({ type = 'line', ...props }: PivotProps) => {
  return <Tabs type={type} {...props} />;
};

export default Pivot;
