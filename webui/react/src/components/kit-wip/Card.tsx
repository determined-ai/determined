import { Card as AntdCard } from 'antd';
import React, { CSSProperties, ReactNode } from 'react';

interface CardProps {
  bodyStyle?: CSSProperties;
  children?: ReactNode;
  className?: string;
  extra?: ReactNode;
  headStyle?: CSSProperties;
  style?: CSSProperties;
  title?: string;
}

const Card: React.FC<CardProps> = (props: CardProps) => {
  return (
    <AntdCard {...props} />
  );
};

export default Card;
