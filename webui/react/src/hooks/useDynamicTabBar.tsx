import { TabsProps } from 'antd';
import React from 'react';

export type DynamicTabBarProps = Omit<TabsProps, 'tabBarExtraContent'>

const DynamicTabBar: React.FC<DynamicTabBarProps> = ({
  type,
  className,
  size: propSize,
  onEdit,
  hideAdd,
  centered,
  addIcon,
  ...props
}: TabsProps): JSX.Element => {
  return <></>;

};

export default DynamicTabBar;
