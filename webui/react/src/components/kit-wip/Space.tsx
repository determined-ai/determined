import { Space as AntdSpace } from 'antd';
import React, { ReactNode } from 'react';

type SpaceSize = 'small' | 'middle' | 'large' | number;

interface SpaceProps {
  align?: 'start' | 'end' | 'center' | 'baseline';
  children?: ReactNode;
  className?: string;
  onClick?: () => void;
  size?: SpaceSize | [SpaceSize, SpaceSize];
  wrap?: boolean;
}

const Space: React.FC<SpaceProps> = ({ size = 'small', ...props }: SpaceProps) => {
  return (
    <AntdSpace size={size} {...props} />
  );
};

export default Space;
