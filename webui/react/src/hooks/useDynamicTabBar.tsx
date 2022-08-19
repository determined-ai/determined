import { TabsProps } from 'antd';

export interface DynamicTabBarProps extends Omit<TabsProps, 'tabBarExtraContent'>

const DynamicBarTab = ({
  type,
  className,
  size: propSize,
  onEdit,
  hideAdd,
  centered,
  addIcon,
  ...props
}: TabsProps): JSX.Element => {

};
